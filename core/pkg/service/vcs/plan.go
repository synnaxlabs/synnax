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
	"path"
	"slices"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// planned is a plan with what Apply needs to run it.
type planned struct {
	plan  Plan
	live  *tree
	steps map[resource.ID]step
	// finish moves the repository after the apply commits. incoming holds the conflicts
	// that took the target.
	finish func(ctx context.Context, incoming set.Set[resource.ID]) error
}

// Plan returns what applying target changes on the live configuration.
func (s *Service) Plan(ctx context.Context, target Target) (Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.resolve(ctx, target)
	if err != nil {
		return Plan{}, err
	}
	return p.plan, nil
}

// Apply applies target to the live configuration in one transaction run by write, and
// returns what undoes it. Each conflict needs a choice. A pull then moves the tracked
// configuration to the remote, replays the commits that wait for a push, and pushes
// them. A load tracks the loaded configuration, and its undo tracks the previous one
// again. A restore leaves its changes uncommitted. It returns ErrUnresolvedConflict
// when a conflict has no choice, and ErrMoved when the target moved since the plan.
// stop checks each running task that the apply deletes before it stops the task.
func (s *Service) Apply(
	ctx context.Context,
	target Target,
	choices []Choice,
	write Write,
	stop Check,
) (Inverse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.resolve(ctx, target)
	if err != nil {
		return Inverse{}, err
	}
	chosen := make(map[string]ChoiceKind, len(choices))
	for _, c := range choices {
		if c.Choice != ChoiceMine && c.Choice != ChoiceIncoming {
			return Inverse{}, errors.Wrapf(
				validate.ErrValidation,
				"choice: %q is not mine or incoming",
				c.Choice,
			)
		}
		chosen[c.Resource] = c.Choice
	}
	incoming := set.New[resource.ID]()
	steps := make([]step, 0, len(p.plan.Items))
	for _, it := range p.plan.Items {
		id, err := resource.ParseID(it.Resource)
		if err != nil {
			return Inverse{}, err
		}
		if it.Conflict {
			c, ok := chosen[it.Resource]
			if !ok {
				return Inverse{}, errors.Wrapf(ErrUnresolvedConflict, "%s", it.Name)
			}
			if c == ChoiceMine {
				continue
			}
			incoming.Add(id)
		}
		steps = append(steps, p.steps[id])
	}
	inv, err := s.applySteps(ctx, steps, p.live, write, stop)
	if err != nil {
		return Inverse{}, err
	}
	if target.Kind == TargetLoad {
		inv.Configuration = s.conn.Configuration
	}
	if p.finish != nil {
		if err = p.finish(ctx, incoming); err != nil {
			// The repository did not follow the live configuration, so it goes back.
			return Inverse{}, errors.Combine(err, s.revert(ctx, inv, write, stop))
		}
	}
	_, err = s.recompute(ctx)
	return inv, err
}

func (s *Service) resolve(ctx context.Context, target Target) (*planned, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	live, err := s.working(ctx)
	if err != nil {
		return nil, err
	}
	p := &planned{
		plan:  Plan{Target: target, Items: []PlanItem{}},
		live:  live,
		steps: map[resource.ID]step{},
	}
	switch target.Kind {
	case TargetPull:
		err = s.resolvePull(p)
	case TargetRestore:
		err = s.resolveRestore(p)
	case TargetLoad:
		err = s.resolveLoad(p)
	default:
		err = errors.Wrapf(
			validate.ErrValidation,
			"target: %q is not pull, restore, or load",
			target.Kind,
		)
	}
	if err != nil {
		return nil, err
	}
	sortItems(p.plan.Items)
	return p, s.markRunning(ctx, p.plan.Items, func(id resource.ID) (bool, error) {
		return sameProgram(p.live, p.steps[id].from, id)
	})
}

// add puts the item that moves the resource with id from live to dest in p. It adds
// nothing when they hold the same resource, or when dest deletes a channel.
func (p *planned) add(id resource.ID, dest *tree, conflict bool) {
	if p.live.same(dest, id) {
		return
	}
	it, inLive := p.live.items[id]
	next, inDest := dest.items[id]
	item := PlanItem{Resource: id.String(), Type: string(id.Type), Conflict: conflict}
	st := step{id: id, from: dest}
	switch {
	case !inLive:
		item.Kind, item.Name = PlanCreate, next.name
	case !inDest:
		if id.Type == resource.TypeChannel {
			return
		}
		item.Kind, item.Name, st.from = PlanDelete, it.name, nil
	default:
		item.Kind, item.Name = PlanChange, next.name
	}
	p.plan.Items = append(p.plan.Items, item)
	p.steps[id] = st
}

// pin sets the commit of the target to h, or returns ErrMoved when the target names
// another commit.
func (p *planned) pin(h plumbing.Hash) error {
	if c := p.plan.Target.Commit; c != "" && c != h.String() {
		return ErrMoved
	}
	p.plan.Target.Commit = h.String()
	return nil
}

func sortItems(items []PlanItem) {
	slices.SortFunc(items, func(a, b PlanItem) int {
		return cmpStrings(a.Type, b.Type, a.Name, b.Name, a.Resource, b.Resource)
	})
}

func (s *Service) resolvePull(p *planned) error {
	conf := s.conn.Configuration
	origin, ok, err := s.repo.ref(remoteRef(conf))
	if err != nil || !ok {
		return err
	}
	if err = p.pin(origin); err != nil {
		return err
	}
	head, _, err := s.head()
	if err != nil {
		return err
	}
	base, err := s.repo.mergeBase(head, origin)
	if err != nil {
		return err
	}
	baseTree, err := s.repo.readTree(base)
	if err != nil {
		return err
	}
	originTree, err := s.repo.readTree(origin)
	if err != nil {
		return err
	}
	mine := set.New[string]()
	for _, c := range p.live.diff(baseTree) {
		mine.Add(c.Resource)
	}
	for _, c := range originTree.diff(baseTree) {
		id, err := resource.ParseID(c.Resource)
		if err != nil {
			return err
		}
		p.add(id, originTree, mine.Contains(c.Resource))
	}
	p.finish = func(ctx context.Context, incoming set.Set[resource.ID]) error {
		tip, err := s.replay(head, origin, incoming)
		if err != nil {
			return err
		}
		if err = s.repo.setRef(branchRef(conf), tip); err != nil {
			return err
		}
		s.push(ctx)
		return nil
	}
	return nil
}

// replay commits the changes of each commit from base to head that waits for a push
// onto onto, oldest first, and returns the new tip. It leaves out the resources in skip
// and every commit that changes nothing. A waiting resource that would take the file of
// an incoming one moves to a numbered name.
func (s *Service) replay(
	head, onto plumbing.Hash,
	skip set.Set[resource.ID],
) (plumbing.Hash, error) {
	waiting, err := s.repo.between(head, onto)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	slices.Reverse(waiting)
	tip := onto
	m := moves{}
	for _, c := range waiting {
		changes, err := s.repo.changesOf(c.Hash)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		var selected []resource.ID
		for _, ch := range changes {
			id, err := resource.ParseID(ch.Resource)
			if err != nil {
				return plumbing.ZeroHash, err
			}
			if !skip.Contains(id) {
				selected = append(selected, id)
			}
		}
		tipTree, err := s.repo.readTree(tip)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		commitTree, err := s.repo.readTree(c.Hash)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		fs, err := patch(
			tipTree,
			relocate(tipTree, commitTree, selected, m),
			selected,
		)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		same, err := s.repo.sameTree(tip, fs)
		if err != nil {
			return plumbing.ZeroHash, err
		}
		if same {
			continue
		}
		if tip, err = s.repo.commitAs(fs, c.Message, c.Author, tip); err != nil {
			return plumbing.ZeroHash, err
		}
	}
	return tip, nil
}

// moves maps the paths of waiting commits to the free paths that replay gave them. A
// key that ends in a slash moves a folder and the files in it.
type moves map[string]string

// to returns the path that p moved to, or p.
func (m moves) to(p string) string {
	if to, ok := m[p]; ok {
		return to
	}
	from := ""
	for k := range m {
		if strings.HasSuffix(k, "/") && strings.HasPrefix(p, k) && len(k) > len(from) {
			from = k
		}
	}
	if from == "" {
		return p
	}
	return m[from] + strings.TrimPrefix(p, from)
}

// relocate returns the selected resources of work at their paths in m. It first adds a
// move to m for each one that would take a file of a resource that base keeps.
func relocate(base, work *tree, selected []resource.ID, m moves) *tree {
	chosen := set.New(selected...)
	occupied := set.New[string]()
	for id, it := range base.items {
		if !chosen.Contains(id) {
			for _, p := range it.paths {
				occupied.Add(filename.Fold(p))
			}
		}
	}
	var items []*item
	for _, id := range selected {
		if it, ok := work.items[id]; ok {
			items = append(items, it)
		}
	}
	// A folder moves before the files in it.
	slices.SortFunc(items, func(a, b *item) int {
		_, aFolder := folderOf(a.main())
		_, bFolder := folderOf(b.main())
		if aFolder != bFolder {
			return lo.Ternary(aFolder, -1, 1)
		}
		return cmp.Or(
			cmp.Compare(strings.Count(a.main(), "/"), strings.Count(b.main(), "/")),
			cmp.Compare(a.main(), b.main()),
		)
	})
	out := newTree()
	out.raw[readmePath] = work.raw[readmePath]
	out.raw[manifestPath] = work.raw[manifestPath]
	for _, it := range items {
		if slices.ContainsFunc(it.paths, func(p string) bool {
			return occupied.Contains(filename.Fold(m.to(p)))
		}) {
			m.move(it, occupied)
		}
		fs := files{}
		for _, p := range it.paths {
			to := m.to(p)
			fs[to] = work.raw[p]
			occupied.Add(filename.Fold(to))
		}
		out.add(it.id, it.name, fs)
	}
	return out
}

// move adds a move to m that gives it a numbered name that no path of occupied holds.
// A project, group, or Arc directory moves with its folder.
func (m moves) move(it *item, occupied set.Set[string]) {
	main := m.to(it.main())
	if dir, ok := folderOf(main); ok {
		parent, name := path.Split(dir)
		claimed := claim(entries(occupied, parent), name, "", []string{""})
		m[path.Dir(it.main())+"/"] = parent + claimed + "/"
		return
	}
	parent, name := path.Split(main)
	ext := extOf(it.id, main)
	exts := make([]string, len(it.paths))
	for i, p := range it.paths {
		exts[i] = extOf(it.id, p)
	}
	stem := strings.TrimSuffix(claim(entries(occupied, parent), name, ext, exts), ext)
	for _, p := range it.paths {
		m[p] = parent + stem + extOf(it.id, p)
	}
}

// folderOf returns the folder of the resource whose own file sits at p, when the
// resource is a folder.
func folderOf(p string) (string, bool) {
	switch path.Base(p) {
	case projectFile, groupFile, directoryFile:
		return path.Dir(p), true
	}
	return "", false
}

// extOf returns the extension of the file at p of a resource with id, such as
// .schematic.json for a schematic.
func extOf(id resource.ID, p string) string {
	if slices.Contains(memberTypes, id.Type) {
		return "." + string(id.Type) + jsonExt
	}
	return path.Ext(p)
}

// entries returns the folded names of the entries of the folder dir, which ends in a
// slash, that the folded paths of occupied hold.
func entries(occupied set.Set[string], dir string) set.Set[string] {
	prefix := filename.Fold(dir)
	out := set.New[string]()
	for p := range occupied {
		if rest, ok := strings.CutPrefix(p, prefix); ok {
			first, _, _ := strings.Cut(rest, "/")
			out.Add(first)
		}
	}
	return out
}

func (s *Service) resolveRestore(p *planned) error {
	h, err := s.repo.resolve(p.plan.Target.Commit)
	if err != nil {
		return errors.Wrap(query.ErrNotFound, err.Error())
	}
	dest, err := s.repo.readTree(h)
	if err != nil {
		return err
	}
	ids := dest.ids(p.live)
	if len(p.plan.Target.Resources) > 0 {
		ids = make([]resource.ID, 0, len(p.plan.Target.Resources))
		for _, r := range p.plan.Target.Resources {
			id, err := resource.ParseID(r)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		p.add(id, dest, false)
	}
	return nil
}

func (s *Service) resolveLoad(p *planned) error {
	name := p.plan.Target.Configuration
	if err := validateName(name); err != nil {
		return err
	}
	tip, err := s.tip(name)
	if err != nil {
		return err
	}
	if err = p.pin(tip); err != nil {
		return err
	}
	dest, err := s.repo.readTree(tip)
	if err != nil {
		return err
	}
	_, headTree, err := s.head()
	if err != nil {
		return err
	}
	uncommitted := set.New[resource.ID]()
	for _, c := range p.live.diff(headTree) {
		id, err := resource.ParseID(c.Resource)
		if err != nil {
			return err
		}
		uncommitted.Add(id)
	}
	for _, id := range dest.ids(p.live) {
		p.add(id, dest, uncommitted.Contains(id))
	}
	p.finish = func(context.Context, set.Set[resource.ID]) error {
		if _, ok, err := s.repo.ref(branchRef(name)); err != nil || ok {
			if err != nil {
				return err
			}
			return s.track(name)
		}
		if err = s.repo.setRef(branchRef(name), tip); err != nil {
			return err
		}
		return s.track(name)
	}
	return nil
}

// markRunning sets Running and Restarts on each task and Arc item. A running task
// restarts when the plan changes it, and a running Arc when same reports that its
// program changed.
func (s *Service) markRunning(
	ctx context.Context,
	items []PlanItem,
	same func(id resource.ID) (bool, error),
) error {
	for i := range items {
		it := &items[i]
		if it.Kind == PlanCreate {
			continue
		}
		id, err := resource.ParseID(it.Resource)
		if err != nil {
			return err
		}
		var keys []task.Key
		switch id.Type {
		case resource.TypeTask:
			key, err := resource.ParseUUIDKey(id)
			if err != nil {
				return err
			}
			keys = []task.Key{key}
		case resource.TypeArc:
			tasks, err := s.arcTasks(ctx, id)
			if err != nil {
				return err
			}
			keys = lo.Map(tasks, func(t task.Task, _ int) task.Key { return t.Key })
		default:
			continue
		}
		running, err := s.running(ctx, keys)
		if err != nil {
			return err
		}
		it.Running = len(running) > 0
		it.Restarts = it.Running && it.Kind == PlanChange
		if it.Restarts && id.Type == resource.TypeArc {
			unchanged, err := same(id)
			if err != nil {
				return err
			}
			it.Restarts = !unchanged
		}
	}
	return nil
}

// arcTasks returns the tasks that run the Arc with id.
func (s *Service) arcTasks(ctx context.Context, id resource.ID) ([]task.Task, error) {
	key, err := resource.ParseUUIDKey(id)
	if err != nil {
		return nil, err
	}
	var tasks []task.Task
	if err = s.cfg.Task.NewRetrieve().
		Where(s.cfg.MatchArcTasks(key)).
		Entries(&tasks).
		Exec(ctx, s.cfg.DB); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	return tasks, nil
}

// running returns the tasks of keys that report running.
func (s *Service) running(ctx context.Context, keys []task.Key) ([]task.Key, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	owners := lo.Map(
		keys,
		func(k task.Key, _ int) resource.ID { return task.ResourceID(k) },
	)
	var statuses []task.Status
	if err := s.cfg.Status.NewRetrieve[task.StatusDetails]().
		Where(status.MatchOwners[task.StatusDetails](owners...)).
		Entries(&statuses).
		Exec(ctx, s.cfg.DB); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	var out []task.Key
	for _, st := range statuses {
		if st.Details.Running && st.Owner != nil {
			key, err := resource.ParseUUIDKey(*st.Owner)
			if err != nil {
				return nil, err
			}
			out = append(out, key)
		}
	}
	return out, nil
}

// stopRemoved stops the running tasks that steps delete, with the tasks of each Arc
// they delete, so the Driver frees their channels. It returns the tasks it stopped.
func (s *Service) stopRemoved(
	ctx context.Context,
	steps []step,
	live *tree,
	stop Check,
) ([]task.Task, error) {
	var keys []task.Key
	for _, st := range steps {
		if st.from != nil {
			continue
		}
		if _, ok := live.items[st.id]; !ok {
			continue
		}
		switch st.id.Type {
		case resource.TypeTask:
			key, err := resource.ParseUUIDKey(st.id)
			if err != nil {
				return nil, err
			}
			keys = append(keys, key)
		case resource.TypeArc:
			tasks, err := s.arcTasks(ctx, st.id)
			if err != nil {
				return nil, err
			}
			for _, t := range tasks {
				keys = append(keys, t.Key)
			}
		}
	}
	running, err := s.running(ctx, keys)
	if err != nil || len(running) == 0 {
		return nil, err
	}
	ids := make([]resource.ID, len(running))
	for i, k := range running {
		ids[i] = task.ResourceID(k)
	}
	if err = stop(ctx, ids); err != nil {
		return nil, err
	}
	var stopped []task.Task
	if err = s.cfg.Task.NewRetrieve().
		Where(task.MatchKeys(running...)).
		Entries(&stopped).
		Exec(ctx, s.cfg.DB); err != nil {
		return nil, err
	}
	if err = s.cfg.TaskStopper.Stop(ctx, running); err != nil {
		return nil, err
	}
	return stopped, nil
}

// arcTaskHashes returns the config hash of each task of the Arcs in items, so a restart
// can wait for the Core to compile the new source.
func (s *Service) arcTaskHashes(
	ctx context.Context,
	items []PlanItem,
) (map[task.Key]string, error) {
	hashes := map[task.Key]string{}
	for _, it := range items {
		if it.Type != string(resource.TypeArc) {
			continue
		}
		id, err := resource.ParseID(it.Resource)
		if err != nil {
			return nil, err
		}
		tasks, err := s.arcTasks(ctx, id)
		if err != nil {
			return nil, err
		}
		for _, t := range tasks {
			hashes[t.Key] = t.ConfigHash
		}
	}
	return hashes, nil
}

// restart restarts the tasks of items once each has its new config. An Arc task gets
// its config from a compile after the commit, so restart waits for its hash to change
// from the one in before, up to the restart timeout.
func (s *Service) restart(
	ctx context.Context,
	items []PlanItem,
	before map[task.Key]string,
) error {
	var tasks []task.Task
	for _, it := range items {
		id, err := resource.ParseID(it.Resource)
		if err != nil {
			return err
		}
		switch id.Type {
		case resource.TypeTask:
			key, err := resource.ParseUUIDKey(id)
			if err != nil {
				return err
			}
			var t task.Task
			if err = s.cfg.Task.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&t).
				Exec(ctx, s.cfg.DB); err != nil {
				return err
			}
			tasks = append(tasks, t)
		case resource.TypeArc:
			arcTasks, err := s.awaitArcTasks(ctx, id, before)
			if err != nil {
				return err
			}
			tasks = append(tasks, arcTasks...)
		}
	}
	if len(tasks) == 0 {
		return nil
	}
	return s.cfg.TaskStopper.Restart(ctx, tasks)
}

func (s *Service) awaitArcTasks(
	ctx context.Context,
	id resource.ID,
	before map[task.Key]string,
) ([]task.Task, error) {
	deadline := time.Now().Add(s.cfg.RestartTimeout)
	for {
		tasks, err := s.arcTasks(ctx, id)
		if err != nil {
			return nil, err
		}
		if !slices.ContainsFunc(tasks, func(t task.Task) bool {
			return t.ConfigHash == before[t.Key]
		}) || time.Now().After(deadline) {
			return tasks, nil
		}
		t := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		case <-t.C:
		}
	}
}
