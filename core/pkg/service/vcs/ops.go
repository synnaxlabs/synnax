// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package vcs

import (
	"cmp"
	"context"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const startMessage = "Start version control"

// ConnectRequest is what Connect needs.
type ConnectRequest struct {
	// Configuration is the name of the configuration to track.
	Configuration string
	// RemoteURL is the URL of the remote, or empty for none.
	RemoteURL string
	// Token authenticates to the remote. Empty keeps the stored token.
	Token string
	// Author names the first commit.
	Author Author
}

// Connect starts version control, or changes its configuration and remote. When the
// remote has no commits of the configuration, Connect commits the live configuration
// and pushes it. When the remote has them, the Core stays behind until a pull or a
// load. It returns a validation error for an invalid name and the error of a failed
// fetch, and then stores nothing.
func (s *Service) Connect(ctx context.Context, req ConnectRequest) (Status, error) {
	if err := validateName(req.Configuration); err != nil {
		return Status{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Token == "" && s.conn.Token != "" && req.RemoteURL != "" &&
		host(req.RemoteURL) != host(s.conn.RemoteURL) {
		return Status{}, validate.Errorf(
			"Enter the access token again. The Core never sends a saved token to a " +
				"new host.",
		)
	}
	conn := s.conn
	conn.Configuration = req.Configuration
	conn.RemoteURL = req.RemoteURL
	if req.Token != "" {
		conn.Token = req.Token
	}
	if conn.RemoteURL != s.conn.RemoteURL {
		// A new remote lacks the snapshots of the Core.
		tags, err := s.repo.rangeTags()
		if err != nil {
			return Status{}, err
		}
		conn.WaitingTags = tags
		if conn.RemoteURL == "" {
			conn.WaitingTags = nil
		}
	}
	if err := s.repo.setRemote(conn.RemoteURL); err != nil {
		return Status{}, err
	}
	if conn.RemoteURL != "" {
		pruneTags := len(conn.WaitingTags) == 0
		if err := s.repo.fetch(ctx, conn.remote(), pruneTags); err != nil {
			return Status{}, errors.Combine(err, s.repo.setRemote(s.conn.RemoteURL))
		}
		s.fetchedAt = telem.Now()
	}
	if err := writeConnection(s.dir, conn); err != nil {
		return Status{}, err
	}
	s.conn = conn
	s.lastErr, s.tagErr = "", ""
	if err := s.startTracking(ctx, req.Author); err != nil {
		s.cache(nil)
		return Status{}, err
	}
	if len(s.conn.WaitingTags) > 0 {
		s.pushTags(ctx)
	}
	return s.recompute(ctx)
}

// startTracking makes the first commit of the configuration when neither the Core nor
// the
// remote has one, and pushes what the remote lacks.
func (s *Service) startTracking(ctx context.Context, author Author) error {
	_, hasLocal, err := s.repo.ref(branchRef(s.conn.Configuration))
	if err != nil {
		return err
	}
	_, hasOrigin, err := s.repo.ref(remoteRef(s.conn.Configuration))
	if err != nil {
		return err
	}
	if !hasLocal && !hasOrigin {
		work, err := s.working(ctx)
		if err != nil {
			return err
		}
		h, err := s.repo.commit(work.raw, startMessage, author)
		if err != nil {
			return err
		}
		if err = s.repo.setRef(branchRef(s.conn.Configuration), h); err != nil {
			return err
		}
	}
	if hasLocal || !hasOrigin {
		s.push(ctx)
	}
	return nil
}

// host returns the host of a remote URL, or an empty string for a path.
func host(remoteURL string) string {
	u, err := url.Parse(remoteURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// recompute drops the cached status and computes it again.
func (s *Service) recompute(ctx context.Context) (Status, error) {
	s.cache(nil)
	return s.current(ctx)
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)

// validateName checks that name can name a configuration.
func validateName(name string) error {
	if !namePattern.MatchString(name) || name == "HEAD" ||
		strings.HasPrefix(name, reviewPrefix) || branchRef(name).Validate() != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"configuration: %q is not a valid name",
			name,
		)
	}
	return nil
}

// Fetch fetches the remote now. A failed fetch is in the Error of the status.
func (s *Service) Fetch(ctx context.Context) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return Status{}, err
	}
	if s.conn.RemoteURL != "" {
		s.fetch(ctx)
	}
	return s.recompute(ctx)
}

// CommitRequest is what Commit needs.
type CommitRequest struct {
	// Message is the commit message.
	Message string
	// Resources limits the commit to these resource IDs. Empty commits every change.
	Resources []string
	// PullRequest commits to a new branch and opens a pull request into the
	// configuration instead of committing to the configuration.
	PullRequest bool
	// Author is the author of the commit.
	Author Author
}

// Commit commits the uncommitted changes of the selected resources. A commit to the
// configuration pushes it, and a failed push leaves it waiting. It returns the commit
// and the web address of the pull request, if it opened one. It returns a validation
// error for an empty message, a selected resource with no change, or nothing to commit.
func (s *Service) Commit(
	ctx context.Context,
	req CommitRequest,
	check Check,
) (Commit, string, error) {
	if strings.TrimSpace(req.Message) == "" {
		return Commit{}, "", errors.Wrap(validate.ErrValidation, "message: required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return Commit{}, "", err
	}
	head, headTree, err := s.head()
	if err != nil {
		return Commit{}, "", err
	}
	work, err := s.working(ctx)
	if err != nil {
		return Commit{}, "", err
	}
	selected, err := selectChanges(work.diff(headTree), req.Resources)
	if err != nil {
		return Commit{}, "", err
	}
	if err = check(ctx, selected); err != nil {
		return Commit{}, "", err
	}
	parent, base := head, headTree
	if req.PullRequest {
		if s.conn.RemoteURL == "" {
			return Commit{}, "", errors.Wrap(
				validate.ErrValidation,
				"a pull request needs a remote",
			)
		}
		origin, ok, err := s.repo.ref(remoteRef(s.conn.Configuration))
		if err != nil {
			return Commit{}, "", err
		}
		if ok {
			if base, err = s.repo.readTree(origin); err != nil {
				return Commit{}, "", err
			}
			parent = origin
		}
	}
	h, err := s.repo.commit(
		patch(base, work, selected),
		req.Message,
		req.Author,
		parent,
	)
	if err != nil {
		return Commit{}, "", err
	}
	c, err := s.repo.toCommit(h)
	if err != nil {
		return Commit{}, "", err
	}
	if !req.PullRequest {
		if err = s.repo.setRef(branchRef(s.conn.Configuration), h); err != nil {
			return Commit{}, "", err
		}
		s.push(ctx)
		_, err = s.recompute(ctx)
		return c, "", err
	}
	review, err := s.openPullRequest(ctx, req.Message, h)
	if err != nil {
		return Commit{}, "", err
	}
	review.Resources = idStrings(selected)
	s.conn.Reviews = append(s.conn.Reviews, review)
	if err = writeConnection(s.dir, s.conn); err != nil {
		return Commit{}, "", err
	}
	if _, err = s.recompute(ctx); err != nil {
		return Commit{}, "", err
	}
	return c, review.URL, nil
}

func idStrings(ids []resource.ID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// selectChanges returns the IDs of the changes that resources names, or of every change
// when resources is empty.
func selectChanges(changes []Change, resources []string) ([]resource.ID, error) {
	changed := set.New[string]()
	for _, c := range changes {
		changed.Add(c.Resource)
	}
	if len(resources) == 0 {
		resources = slices.Collect(maps.Keys(changed))
		slices.Sort(resources)
	}
	if len(resources) == 0 {
		return nil, errors.Wrap(validate.ErrValidation, "there are no changes")
	}
	ids := make([]resource.ID, 0, len(resources))
	for _, r := range resources {
		if !changed.Contains(r) {
			return nil, errors.Wrapf(validate.ErrValidation, "%s has no change", r)
		}
		id, err := resource.ParseID(r)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// patch returns the files of base with the selected resources as in work, and the
// generated files of work.
func patch(base, work *tree, selected []resource.ID) files {
	out := maps.Clone(base.raw)
	if out == nil {
		out = files{}
	}
	for _, id := range selected {
		if it, ok := base.items[id]; ok {
			for _, p := range it.paths {
				delete(out, p)
			}
		}
		if it, ok := work.items[id]; ok {
			for _, p := range it.paths {
				out[p] = work.raw[p]
			}
		}
	}
	out[readmePath] = work.raw[readmePath]
	out[manifestPath] = work.raw[manifestPath]
	return out
}

// DefaultWrite runs apply in one transaction on the DB of the Service, with no access
// checks.
func (s *Service) DefaultWrite(
	ctx context.Context,
	apply func(context.Context, gorp.Tx) error,
) error {
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error { return apply(ctx, tx) })
}

// Discard puts the selected resources back as the head holds them, through write. It
// returns what undoes the discard. Empty resources discards every change. It returns a
// validation error for a selected resource with no change, and for a new channel.
func (s *Service) Discard(
	ctx context.Context,
	resources []string,
	write Write,
) (Inverse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return Inverse{}, err
	}
	_, headTree, err := s.head()
	if err != nil {
		return Inverse{}, err
	}
	work, err := s.working(ctx)
	if err != nil {
		return Inverse{}, err
	}
	selected, err := selectChanges(work.diff(headTree), resources)
	if err != nil {
		return Inverse{}, err
	}
	steps := make([]step, 0, len(selected))
	for _, id := range selected {
		st := step{id: id}
		if _, ok := headTree.items[id]; ok {
			st.from = headTree
		} else if id.Type == resource.TypeChannel {
			return Inverse{}, validate.Errorf(
				"Version control never deletes channels, so it cannot discard the "+
					"new channel %s.",
				work.items[id].name,
			)
		}
		steps = append(steps, st)
	}
	inv, err := s.applySteps(ctx, steps, work, write)
	if err != nil {
		return Inverse{}, err
	}
	if _, err = s.recompute(ctx); err != nil {
		return Inverse{}, err
	}
	return inv, nil
}

// applySteps runs steps on the live configuration through write, restarts each running
// task that they change, and returns what undoes them.
func (s *Service) applySteps(
	ctx context.Context,
	steps []step,
	live *tree,
	write Write,
) (Inverse, error) {
	restarts, err := s.restartsOf(ctx, steps, live)
	if err != nil {
		return Inverse{}, err
	}
	hashes, err := s.arcTaskHashes(ctx, restarts)
	if err != nil {
		return Inverse{}, err
	}
	stopped, err := s.stopRemoved(ctx, steps, live)
	if err != nil {
		return Inverse{}, err
	}
	if err = write(ctx, func(ctx context.Context, tx gorp.Tx) error {
		a, err := s.newApplier(ctx, tx, live)
		if err != nil {
			return err
		}
		return a.run(ctx, steps)
	}); err != nil {
		if len(stopped) > 0 {
			err = errors.Combine(err, s.cfg.TaskStopper.Restart(ctx, stopped))
		}
		return Inverse{}, err
	}
	if err = s.restart(ctx, restarts, hashes); err != nil {
		s.fail("restart", err)
	}
	return inverse(steps, live), nil
}

// restartsOf returns an item for each running task and Arc that steps change in place.
func (s *Service) restartsOf(
	ctx context.Context,
	steps []step,
	live *tree,
) ([]PlanItem, error) {
	items := make([]PlanItem, 0, len(steps))
	from := make(map[resource.ID]*tree, len(steps))
	for _, st := range steps {
		if st.from == nil {
			continue
		}
		if _, ok := live.items[st.id]; !ok {
			continue
		}
		from[st.id] = st.from
		items = append(items, PlanItem{
			Resource: st.id.String(),
			Type:     string(st.id.Type),
			Kind:     PlanChange,
		})
	}
	if err := s.markRunning(ctx, items, func(id resource.ID) (bool, error) {
		return sameProgram(live, from[id], id)
	}); err != nil {
		return nil, err
	}
	return lo.Filter(items, func(it PlanItem, _ int) bool { return it.Restarts }), nil
}

// Write puts the files of inv on the live configuration and removes the resources it
// names, through write, then tracks the configuration inv names. It undoes a Discard or
// an Apply. It returns query.ErrNotFound when the configuration does not exist.
func (s *Service) Write(ctx context.Context, inv Inverse, write Write) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inv.Configuration != "" {
		if err := validateName(inv.Configuration); err != nil {
			return err
		}
		if _, err := s.tip(inv.Configuration); err != nil {
			return err
		}
	}
	err := s.revert(ctx, inv, write)
	if err != nil {
		return err
	}
	if inv.Configuration != "" {
		if err = s.track(inv.Configuration); err != nil {
			return err
		}
	}
	_, err = s.recompute(ctx)
	return err
}

// revert puts the files of inv on the live configuration and removes the resources it
// names, through write. The caller holds mu.
func (s *Service) revert(ctx context.Context, inv Inverse, write Write) error {
	t, err := treeOfFiles(inv.Files)
	if err != nil {
		return err
	}
	work, err := s.working(ctx)
	if err != nil {
		return err
	}
	steps := make([]step, 0, len(t.items)+len(inv.Removed))
	for id := range t.items {
		steps = append(steps, step{id: id, from: t})
	}
	for _, r := range inv.Removed {
		id, err := resource.ParseID(r)
		if err != nil {
			return err
		}
		if _, ok := work.items[id]; ok {
			steps = append(steps, step{id: id})
		}
	}
	_, err = s.applySteps(ctx, steps, work, write)
	return err
}

// treeOf returns the files of rev: "working", "head", "configuration/<name>", a commit
// hash, or "<hash>^" for the parent of a commit. The parent of a root commit has no
// files.
func (s *Service) treeOf(ctx context.Context, rev string) (*tree, error) {
	if child, ok := strings.CutSuffix(rev, "^"); ok && child != "" {
		h, err := s.repo.resolve(child)
		if err != nil {
			return nil, errors.Wrap(query.ErrNotFound, err.Error())
		}
		c, err := s.repo.git.CommitObject(h)
		if err != nil {
			return nil, err
		}
		if len(c.ParentHashes) == 0 {
			return newTree(), nil
		}
		return s.repo.readTree(c.ParentHashes[0])
	}
	switch {
	case rev == RevWorking:
		return s.working(ctx)
	case rev == RevHead:
		_, t, err := s.head()
		if err != nil {
			return nil, err
		}
		return t, nil
	case strings.HasPrefix(rev, revConfigurationPrefix):
		h, err := s.tip(strings.TrimPrefix(rev, revConfigurationPrefix))
		if err != nil {
			return nil, err
		}
		return s.repo.readTree(h)
	}
	h, err := s.repo.resolve(rev)
	if err != nil {
		return nil, errors.Wrap(query.ErrNotFound, err.Error())
	}
	return s.repo.readTree(h)
}

// tip returns the newest commit of the configuration name: the local one, else the one
// on the remote. It returns query.ErrNotFound when neither exists.
func (s *Service) tip(name string) (plumbing.Hash, error) {
	for _, ref := range []plumbing.ReferenceName{branchRef(name), remoteRef(name)} {
		h, ok, err := s.repo.ref(ref)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		if ok {
			return h, nil
		}
	}
	return plumbing.ZeroHash, errors.Wrapf(
		query.ErrNotFound,
		"configuration %q not found",
		name,
	)
}

// Files returns the files of the resource in rev, or every file of rev when resource is
// empty.
func (s *Service) Files(ctx context.Context, rev, res string) ([]File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	t, err := s.treeOf(ctx, rev)
	if err != nil {
		return nil, err
	}
	if res == "" {
		out := make([]File, 0, len(t.raw))
		for _, p := range slices.Sorted(maps.Keys(t.raw)) {
			out = append(out, File{Path: p, Content: string(t.raw[p])})
		}
		return out, nil
	}
	id, err := resource.ParseID(res)
	if err != nil {
		return nil, err
	}
	if fs := t.filesOf(id); fs != nil {
		return fs, nil
	}
	return []File{}, nil
}

// LogRequest is what Log needs.
type LogRequest struct {
	// Resource limits the log to the commits that change this resource ID.
	Resource string
	// Configuration is the configuration to read. Empty reads the tracked one.
	Configuration string
	// Limit caps the number of commits. Zero returns every commit.
	Limit int
}

// Log returns the commits of a configuration, newest first.
func (s *Service) Log(ctx context.Context, req LogRequest) ([]Commit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	name := req.Configuration
	if name == "" {
		name = s.conn.Configuration
	}
	tip, err := s.tip(name)
	if errors.Is(err, query.ErrNotFound) && req.Configuration == "" {
		return []Commit{}, nil
	}
	if err != nil {
		return nil, err
	}
	var filter resource.ID
	if req.Resource != "" {
		if filter, err = resource.ParseID(req.Resource); err != nil {
			return nil, err
		}
	}
	out := []Commit{}
	var walkErr error
	if err = s.repo.walk(tip, func(c *object.Commit) bool {
		if req.Resource != "" {
			changes, err := s.repo.changesOf(c.Hash)
			if err != nil {
				walkErr = err
				return false
			}
			if !slices.ContainsFunc(changes, func(ch Change) bool {
				return ch.Resource == filter.String()
			}) {
				return true
			}
		}
		out = append(out, wireCommit(c))
		return req.Limit <= 0 || len(out) < req.Limit
	}); err != nil {
		return nil, err
	}
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

// Show returns the commit with the given hash and the resources it changed.
func (s *Service) Show(_ context.Context, hash string) (Commit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return Commit{}, err
	}
	h, err := s.repo.resolve(hash)
	if err != nil {
		return Commit{}, errors.Wrap(query.ErrNotFound, err.Error())
	}
	c, err := s.repo.toCommit(h)
	if err != nil {
		return Commit{}, err
	}
	if c.Changes, err = s.repo.changesOf(h); err != nil {
		return Commit{}, err
	}
	return c, nil
}

// Configurations returns every configuration of the repository and the remote.
func (s *Service) Configurations(_ context.Context) ([]Configuration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	names, err := s.repo.branches()
	if err != nil {
		return nil, err
	}
	out := make([]Configuration, 0, len(names))
	for _, n := range names {
		h, err := s.tip(n)
		if err != nil {
			return nil, err
		}
		c, err := s.repo.toCommit(h)
		if err != nil {
			return nil, err
		}
		out = append(out, Configuration{
			Name:    n,
			Head:    c,
			Current: n == s.conn.Configuration,
		})
	}
	return out, nil
}

// Clone starts a configuration with the given name from the head of the tracked one,
// tracks it, and pushes it. The uncommitted changes stay uncommitted. It returns a
// validation error for an invalid or taken name.
func (s *Service) Clone(ctx context.Context, name string) (Status, error) {
	if err := validateName(name); err != nil {
		return Status{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return Status{}, err
	}
	if _, err := s.tip(name); err == nil {
		return Status{}, errors.Wrapf(
			validate.ErrValidation,
			"configuration %q exists",
			name,
		)
	}
	head, _, err := s.head()
	if err != nil {
		return Status{}, err
	}
	if head.IsZero() {
		return Status{}, errors.Wrap(
			validate.ErrValidation,
			"the configuration has no commit to clone",
		)
	}
	if err = s.repo.setRef(branchRef(name), head); err != nil {
		return Status{}, err
	}
	if err = s.track(name); err != nil {
		return Status{}, err
	}
	s.push(ctx)
	return s.recompute(ctx)
}

// track makes name the tracked configuration.
func (s *Service) track(name string) error {
	conn := s.conn
	conn.Configuration = name
	conn.Reviews = nil
	if err := writeConnection(s.dir, conn); err != nil {
		return err
	}
	s.conn = conn
	return nil
}

// Compare returns the resources that differ from the revision from to the revision to.
func (s *Service) Compare(ctx context.Context, from, to string) ([]Change, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	a, err := s.treeOf(ctx, from)
	if err != nil {
		return nil, err
	}
	b, err := s.treeOf(ctx, to)
	if err != nil {
		return nil, err
	}
	return b.diff(a), nil
}

// Snapshot tags the live configuration for the range with the given key and pushes the
// tag. With no uncommitted changes, the tag marks the head. Else it marks a new commit
// of the live configuration that no configuration holds.
func (s *Service) Snapshot(
	ctx context.Context,
	rangeKey string,
	author Author,
	check Check,
) (Snapshot, error) {
	if rangeKey == "" || strings.Contains(rangeKey, "/") {
		return Snapshot{}, errors.Wrap(validate.ErrValidation, "range: required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return Snapshot{}, err
	}
	head, headTree, err := s.head()
	if err != nil {
		return Snapshot{}, err
	}
	work, err := s.working(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	changes := work.diff(headTree)
	ids := make([]resource.ID, 0, len(changes))
	for _, c := range changes {
		id, err := resource.ParseID(c.Resource)
		if err != nil {
			return Snapshot{}, err
		}
		ids = append(ids, id)
	}
	if err = check(ctx, ids); err != nil {
		return Snapshot{}, err
	}
	uncommitted := len(changes)
	target := head
	if uncommitted > 0 || head.IsZero() {
		if target, err = s.repo.commit(
			work.raw,
			"Snapshot for range "+rangeKey,
			author,
			head,
		); err != nil {
			return Snapshot{}, err
		}
	}
	now := time.Now()
	name := tagPrefix + rangeKey + "/" + strconv.FormatInt(now.UnixNano(), 10)
	sig := committer(now)
	if _, err = s.repo.git.CreateTag(name, target, &git.CreateTagOptions{
		Tagger:  &sig,
		Message: snapshotMessage(s.conn.Configuration, uncommitted),
	}); err != nil {
		return Snapshot{}, err
	}
	if s.conn.RemoteURL != "" {
		conn := s.conn
		conn.WaitingTags = append(slices.Clone(conn.WaitingTags), name)
		if err = writeConnection(s.dir, conn); err != nil {
			return Snapshot{}, err
		}
		s.conn = conn
		s.pushTags(ctx)
		s.refresh(ctx)
	}
	c, err := s.repo.toCommit(target)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Range:         rangeKey,
		Tag:           name,
		Commit:        c,
		Configuration: s.conn.Configuration,
		Uncommitted:   uncommitted,
	}, nil
}

const (
	configurationLine = "configuration: "
	uncommittedLine   = "uncommitted: "
)

func snapshotMessage(conf string, uncommitted int) string {
	return configurationLine + conf + "\n" + uncommittedLine +
		strconv.Itoa(uncommitted) + "\n"
}

// Snapshots returns the snapshots of the range with the given key, newest first.
func (s *Service) Snapshots(_ context.Context, rangeKey string) ([]Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	refs, err := s.repo.git.Tags()
	if err != nil {
		return nil, err
	}
	prefix := tagPrefix + rangeKey + "/"
	type dated struct {
		snap Snapshot
		// ns is the time in the tag name, which is finer than the time of the tag.
		ns int64
	}
	var found []dated
	if err = refs.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		if !strings.HasPrefix(name, prefix) {
			return nil
		}
		// The remote can hold range tags that no Core made, which are no snapshots.
		tag, err := s.repo.git.TagObject(ref.Hash())
		if errors.Is(err, plumbing.ErrObjectNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		ns, err := strconv.ParseInt(strings.TrimPrefix(name, prefix), 10, 64)
		if err != nil {
			return nil
		}
		c, err := s.repo.toCommit(tag.Target)
		if err != nil {
			return err
		}
		snap := Snapshot{Range: rangeKey, Tag: name, Commit: c}
		for line := range strings.SplitSeq(tag.Message, "\n") {
			if v, ok := strings.CutPrefix(line, configurationLine); ok {
				snap.Configuration = v
			} else if v, ok := strings.CutPrefix(line, uncommittedLine); ok {
				if snap.Uncommitted, err = strconv.Atoi(v); err != nil {
					return nil
				}
			}
		}
		found = append(found, dated{snap: snap, ns: ns})
		return nil
	}); err != nil {
		return nil, err
	}
	slices.SortFunc(found, func(a, b dated) int { return cmp.Compare(b.ns, a.ns) })
	out := make([]Snapshot, len(found))
	for i, d := range found {
		out[i] = d.snap
	}
	return out, nil
}
