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
	"bytes"
	"context"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

const (
	remoteName    = "origin"
	tagPrefix     = "range/"
	reviewPrefix  = "synnax/"
	shortHashSize = 7
)

// repo is a bare Git repository. It writes trees and commits straight to the object
// store, so it keeps no work tree. It is not safe for concurrent use.
type repo struct {
	git *git.Repository
	// trees caches parsed trees by hash. Trees never change.
	trees map[plumbing.Hash]*tree
}

func openRepo(dir string) (*repo, error) {
	r, err := git.PlainOpen(dir)
	if errors.Is(err, git.ErrRepositoryNotExists) {
		r, err = git.PlainInit(dir, true)
	}
	if err != nil {
		return nil, errors.Wrap(err, "open the repository")
	}
	return &repo{git: r, trees: map[plumbing.Hash]*tree{}}, nil
}

func branchRef(name string) plumbing.ReferenceName {
	return plumbing.NewBranchReferenceName(name)
}

func remoteRef(name string) plumbing.ReferenceName {
	return plumbing.NewRemoteReferenceName(remoteName, name)
}

// ref returns the hash that name points at, and false when name does not exist.
func (r *repo) ref(name plumbing.ReferenceName) (plumbing.Hash, bool, error) {
	ref, err := r.git.Storer.Reference(name)
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return plumbing.ZeroHash, false, nil
	}
	if err != nil {
		return plumbing.ZeroHash, false, err
	}
	return ref.Hash(), true, nil
}

func (r *repo) setRef(name plumbing.ReferenceName, h plumbing.Hash) error {
	return r.git.Storer.SetReference(plumbing.NewHashReference(name, h))
}

func (r *repo) writeBlob(b []byte) (plumbing.Hash, error) {
	obj := r.git.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	w, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	if _, err = w.Write(b); err != nil {
		return plumbing.ZeroHash, errors.Combine(err, w.Close())
	}
	if err = w.Close(); err != nil {
		return plumbing.ZeroHash, err
	}
	return r.git.Storer.SetEncodedObject(obj)
}

// writeTree stores fs as a tree and returns its hash.
func (r *repo) writeTree(fs files) (plumbing.Hash, error) {
	type dir struct {
		files map[string]plumbing.Hash
		dirs  map[string]*dir
	}
	newDir := func() *dir {
		return &dir{files: map[string]plumbing.Hash{}, dirs: map[string]*dir{}}
	}
	root := newDir()
	for p, b := range fs {
		h, err := r.writeBlob(b)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		segments := strings.Split(p, "/")
		d := root
		for _, s := range segments[:len(segments)-1] {
			next, ok := d.dirs[s]
			if !ok {
				next = newDir()
				d.dirs[s] = next
			}
			d = next
		}
		d.files[segments[len(segments)-1]] = h
	}
	var write func(d *dir) (plumbing.Hash, error)
	write = func(d *dir) (plumbing.Hash, error) {
		entries := make([]object.TreeEntry, 0, len(d.files)+len(d.dirs))
		for name, h := range d.files {
			entries = append(entries, object.TreeEntry{
				Name: name, Mode: filemode.Regular, Hash: h,
			})
		}
		for name, sub := range d.dirs {
			h, err := write(sub)
			if err != nil {
				return plumbing.ZeroHash, err
			}
			entries = append(entries, object.TreeEntry{
				Name: name, Mode: filemode.Dir, Hash: h,
			})
		}
		// Git sorts a tree entry for a directory as if its name ended in a slash.
		sortName := func(e object.TreeEntry) string {
			if e.Mode == filemode.Dir {
				return e.Name + "/"
			}
			return e.Name
		}
		slices.SortFunc(entries, func(a, b object.TreeEntry) int {
			return strings.Compare(sortName(a), sortName(b))
		})
		obj := r.git.Storer.NewEncodedObject()
		if err := (&object.Tree{Entries: entries}).Encode(obj); err != nil {
			return plumbing.ZeroHash, err
		}
		return r.git.Storer.SetEncodedObject(obj)
	}
	return write(root)
}

// readTree returns the parsed files of the commit with hash h. The zero hash reads as
// an empty tree.
func (r *repo) readTree(h plumbing.Hash) (*tree, error) {
	if h.IsZero() {
		return newTree(), nil
	}
	c, err := r.git.CommitObject(h)
	if err != nil {
		return nil, errors.Wrapf(err, "read commit %s", h)
	}
	if t, ok := r.trees[c.TreeHash]; ok {
		return t, nil
	}
	gt, err := c.Tree()
	if err != nil {
		return nil, err
	}
	raw := files{}
	if err = gt.Files().ForEach(func(f *object.File) error {
		rd, err := f.Reader()
		if err != nil {
			return err
		}
		b, err := io.ReadAll(rd)
		if err = errors.Combine(err, rd.Close()); err != nil {
			return err
		}
		raw[path.Clean(f.Name)] = b
		return nil
	}); err != nil {
		return nil, err
	}
	t, err := parseTree(raw)
	if err != nil {
		return nil, err
	}
	r.trees[c.TreeHash] = t
	return t, nil
}

// commit stores a commit of fs by author with the given parents and returns its hash.
func (r *repo) commit(
	fs files,
	message string,
	author Author,
	parents ...plumbing.Hash,
) (plumbing.Hash, error) {
	sig := object.Signature{Name: author.Name, Email: author.Email, When: time.Now()}
	return r.commitAs(fs, message, sig, parents...)
}

// commitAs stores a commit of fs with the given author signature and parents. A zero
// parent is left out.
func (r *repo) commitAs(
	fs files,
	message string,
	author object.Signature,
	parents ...plumbing.Hash,
) (plumbing.Hash, error) {
	treeHash, err := r.writeTree(fs)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	c := &object.Commit{
		Author:    author,
		Committer: committer(time.Now()),
		Message:   message,
		TreeHash:  treeHash,
		ParentHashes: slices.DeleteFunc(
			slices.Clone(parents),
			func(h plumbing.Hash) bool {
				return h.IsZero()
			},
		),
	}
	obj := r.git.Storer.NewEncodedObject()
	if err = c.Encode(obj); err != nil {
		return plumbing.ZeroHash, err
	}
	return r.git.Storer.SetEncodedObject(obj)
}

func committer(when time.Time) object.Signature {
	return object.Signature{Name: "Synnax Core", Email: "core@synnax.local", When: when}
}

// sameTree reports whether the commit with hash h stores exactly fs.
func (r *repo) sameTree(h plumbing.Hash, fs files) (bool, error) {
	t, err := r.readTree(h)
	if err != nil {
		return false, err
	}
	if len(t.raw) != len(fs) {
		return false, nil
	}
	for p, b := range fs {
		if !bytes.Equal(t.raw[p], b) {
			return false, nil
		}
	}
	return true, nil
}

// toCommit converts the commit with hash h to its wire form.
func (r *repo) toCommit(h plumbing.Hash) (Commit, error) {
	c, err := r.git.CommitObject(h)
	if err != nil {
		return Commit{}, errors.Wrapf(err, "read commit %s", h)
	}
	return wireCommit(c), nil
}

func wireCommit(c *object.Commit) Commit {
	hash := c.Hash.String()
	return Commit{
		Hash:      hash,
		ShortHash: hash[:shortHashSize],
		Message:   strings.TrimRight(c.Message, "\n"),
		Author:    c.Author.Name,
		Time:      c.Author.When.UnixNano(),
	}
}

// changesOf returns the resources that the commit with hash h changed from its first
// parent.
func (r *repo) changesOf(h plumbing.Hash) ([]Change, error) {
	c, err := r.git.CommitObject(h)
	if err != nil {
		return nil, err
	}
	parent := plumbing.ZeroHash
	if len(c.ParentHashes) > 0 {
		parent = c.ParentHashes[0]
	}
	before, err := r.readTree(parent)
	if err != nil {
		return nil, err
	}
	after, err := r.readTree(h)
	if err != nil {
		return nil, err
	}
	return after.diff(before), nil
}

// between returns the commits reachable from tip and not from base, newest first. A
// zero base returns every commit reachable from tip.
func (r *repo) between(tip, base plumbing.Hash) ([]*object.Commit, error) {
	if tip.IsZero() {
		return nil, nil
	}
	stop := set.New[plumbing.Hash]()
	if !base.IsZero() {
		if err := r.walk(base, func(c *object.Commit) bool {
			stop.Add(c.Hash)
			return true
		}); err != nil {
			return nil, err
		}
	}
	var out []*object.Commit
	if err := r.walk(tip, func(c *object.Commit) bool {
		if stop.Contains(c.Hash) {
			return false
		}
		out = append(out, c)
		return true
	}); err != nil {
		return nil, err
	}
	return out, nil
}

// walk visits the first-parent chain from h, newest first, until visit returns false.
func (r *repo) walk(h plumbing.Hash, visit func(*object.Commit) bool) error {
	for !h.IsZero() {
		c, err := r.git.CommitObject(h)
		if err != nil {
			return err
		}
		if !visit(c) || len(c.ParentHashes) == 0 {
			return nil
		}
		h = c.ParentHashes[0]
	}
	return nil
}

// mergeBase returns the newest commit on the first-parent chain of a that the chain of
// b holds, or the zero hash when they share none.
func (r *repo) mergeBase(a, b plumbing.Hash) (plumbing.Hash, error) {
	onB := set.New[plumbing.Hash]()
	if err := r.walk(b, func(c *object.Commit) bool {
		onB.Add(c.Hash)
		return true
	}); err != nil {
		return plumbing.ZeroHash, err
	}
	base := plumbing.ZeroHash
	if err := r.walk(a, func(c *object.Commit) bool {
		if onB.Contains(c.Hash) {
			base = c.Hash
			return false
		}
		return true
	}); err != nil {
		return plumbing.ZeroHash, err
	}
	return base, nil
}

// resolve returns the commit with the full or short hash rev.
func (r *repo) resolve(rev string) (plumbing.Hash, error) {
	h, err := r.git.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return plumbing.ZeroHash, errors.Wrapf(err, "resolve %s", rev)
	}
	return *h, nil
}

// remote holds what a fetch or a push needs to reach the remote.
type remote struct {
	url   string
	token string
}

func (rm remote) auth() transport.AuthMethod {
	if rm.token == "" {
		return nil
	}
	return &githttp.BasicAuth{Username: "x-access-token", Password: rm.token}
}

// setRemote points the origin remote at url, or removes it when url is empty.
func (r *repo) setRemote(url string) error {
	existing, err := r.git.Remote(remoteName)
	if err != nil && !errors.Is(err, git.ErrRemoteNotFound) {
		return err
	}
	if existing != nil {
		if urls := existing.Config().URLs; len(urls) == 1 && urls[0] == url {
			return nil
		}
		if err = r.git.DeleteRemote(remoteName); err != nil {
			return err
		}
	}
	if url == "" {
		return nil
	}
	_, err = r.git.CreateRemote(&gitconfig.RemoteConfig{
		Name: remoteName,
		URLs: []string{url},
		Fetch: []gitconfig.RefSpec{
			gitconfig.RefSpec("+refs/heads/*:refs/remotes/" + remoteName + "/*"),
		},
	})
	return err
}

// fetch updates the remote branches and range tags. An empty remote is no error.
func (r *repo) fetch(ctx context.Context, rm remote) error {
	if err := r.fetchRefs(
		ctx,
		rm,
		true,
		"+refs/heads/*:refs/remotes/"+remoteName+"/*",
	); err != nil {
		return err
	}
	// A prune would delete each snapshot tag that still waits for its push.
	return r.fetchRefs(
		ctx,
		rm,
		false,
		"+refs/tags/"+tagPrefix+"*:refs/tags/"+tagPrefix+"*",
	)
}

func (r *repo) fetchRefs(
	ctx context.Context,
	rm remote,
	prune bool,
	spec gitconfig.RefSpec,
) error {
	err := r.git.FetchContext(ctx, &git.FetchOptions{
		RemoteName: remoteName,
		RefSpecs:   []gitconfig.RefSpec{spec},
		Auth:       rm.auth(),
		Tags:       git.NoTags,
		Prune:      prune,
	})
	if errors.Is(err, git.NoErrAlreadyUpToDate) ||
		errors.Is(err, transport.ErrEmptyRemoteRepository) {
		return nil
	}
	return explain(err)
}

// push sends the local branch name to the remote and records the remote branch.
func (r *repo) push(ctx context.Context, rm remote, name string) error {
	h, ok, err := r.ref(branchRef(name))
	if err != nil || !ok {
		return err
	}
	spec := gitconfig.RefSpec(string(branchRef(name)) + ":" + string(branchRef(name)))
	err = r.git.PushContext(ctx, &git.PushOptions{
		RemoteName: remoteName,
		RefSpecs:   []gitconfig.RefSpec{spec},
		Auth:       rm.auth(),
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return explain(err)
	}
	return r.setRef(remoteRef(name), h)
}

// pushTags sends every range tag to the remote.
func (r *repo) pushTags(ctx context.Context, rm remote) error {
	err := r.git.PushContext(ctx, &git.PushOptions{
		RemoteName: remoteName,
		RefSpecs: []gitconfig.RefSpec{
			gitconfig.RefSpec(
				"refs/tags/" + tagPrefix + "*:refs/tags/" + tagPrefix + "*",
			),
		},
		Auth: rm.auth(),
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return explain(err)
	}
	return nil
}

// explain turns a refusal from the remote into a sentence that says what to fix. The
// remote often sends no reason of its own.
func explain(err error) error {
	switch {
	case errors.Is(err, transport.ErrAuthenticationRequired):
		return errors.New(
			"The remote needs a valid access token. Check that the token is correct " +
				"and has not expired.",
		)
	case errors.Is(err, transport.ErrAuthorizationFailed):
		return errors.New(
			"The access token cannot write to the repository. Give it read and " +
				"write access to the repository contents.",
		)
	case errors.Is(err, transport.ErrRepositoryNotFound):
		return errors.New(
			"No repository was found at this URL. Check the URL, and check that " +
				"the access token can see the repository.",
		)
	}
	return err
}

// branches returns the names of the local and remote branches, sorted, without review
// branches.
func (r *repo) branches() ([]string, error) {
	refs, err := r.git.References()
	if err != nil {
		return nil, err
	}
	var names []string
	remotePrefix := "refs/remotes/" + remoteName + "/"
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		n := string(ref.Name())
		var name string
		switch {
		case ref.Name().IsBranch():
			name = ref.Name().Short()
		case strings.HasPrefix(n, remotePrefix):
			name = strings.TrimPrefix(n, remotePrefix)
		default:
			return nil
		}
		if name != "HEAD" && !strings.HasPrefix(name, reviewPrefix) {
			names = append(names, name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}
