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
	"cmp"
	"context"
	"math/rand/v2"
	"path"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// step is one resource that an apply puts or removes.
type step struct {
	id resource.ID
	// from holds the files to put. A nil from removes the resource.
	from *tree
}

// putOrder is the order in which an apply puts resources, so that a resource exists
// before another one names it.
var putOrder = []resource.Type{
	resource.TypeChannel,
	resource.TypeGroup,
	resource.TypeProject,
	resource.TypeArcDirectory,
	resource.TypeSchematicSymbol,
	resource.TypeArc,
	resource.TypeTask,
	resource.TypeSchematic,
	resource.TypeLineplot,
	resource.TypeLog,
	resource.TypeTable,
	resource.TypePanel,
}

// removeOrder is the order in which an apply removes resources: the resources that name
// others first, the folders that hold others last. An apply never removes a channel.
var removeOrder = []resource.Type{
	resource.TypePanel,
	resource.TypeSchematic,
	resource.TypeLineplot,
	resource.TypeLog,
	resource.TypeTable,
	resource.TypeSchematicSymbol,
	resource.TypeTask,
	resource.TypeArc,
	resource.TypeArcDirectory,
	resource.TypeGroup,
	resource.TypeProject,
}

// depth is the number of folders above the main file of the resource with id in t.
func depth(t *tree, id resource.ID) int {
	if it, ok := t.items[id]; ok {
		return strings.Count(it.main(), "/")
	}
	return 0
}

// order sorts steps into the order that an apply runs them in.
func order(steps []step, live *tree) []step {
	rank := func(s step) (int, int) {
		if s.from == nil {
			// A deeper folder goes first, since it sits in a shallower one.
			r := len(putOrder) + slices.Index(removeOrder, s.id.Type)
			return r, -depth(live, s.id)
		}
		r := slices.Index(putOrder, s.id.Type)
		if s.id.Type == resource.TypeGroup || s.id.Type == resource.TypeProject {
			// Groups and projects hold each other, so they go by depth.
			r = slices.Index(putOrder, resource.TypeGroup)
		}
		return r, depth(s.from, s.id)
	}
	sorted := slices.Clone(steps)
	slices.SortStableFunc(sorted, func(a, b step) int {
		ra, da := rank(a)
		rb, db := rank(b)
		if c := cmp.Compare(ra, rb); c != 0 {
			return c
		}
		if c := cmp.Compare(da, db); c != 0 {
			return c
		}
		// A group goes before a project at the same depth, since only the groups root
		// holds both.
		return cmp.Compare(
			slices.Index(putOrder, a.id.Type),
			slices.Index(putOrder, b.id.Type),
		)
	})
	return sorted
}

// applier runs the steps of an apply on one transaction.
type applier struct {
	s    *Service
	tx   gorp.Tx
	live *tree
	// root is the key of the Arc directory that holds every Arc.
	root directory.Key
	// dirs holds the stored Arc directories by key.
	dirs map[directory.Key]directory.Directory
	// channels maps the key in the file of each channel the apply put to its key on the
	// Core.
	channels map[channel.Key]channel.Key
}

func (s *Service) newApplier(
	ctx context.Context,
	tx gorp.Tx,
	live *tree,
) (*applier, error) {
	a := &applier{
		s:        s,
		tx:       tx,
		live:     live,
		dirs:     map[directory.Key]directory.Directory{},
		channels: map[channel.Key]channel.Key{},
	}
	var dirs []directory.Directory
	if err := s.cfg.Arc.NewDirectoryRetrieve().
		Entries(&dirs).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	for _, d := range dirs {
		if d.Parent == nil {
			a.root = d.Key
		}
		a.dirs[d.Key] = d
	}
	return a, nil
}

func (a *applier) run(ctx context.Context, steps []step) error {
	for _, st := range order(steps, a.live) {
		var err error
		if st.from == nil {
			err = a.remove(ctx, st.id)
		} else {
			err = a.put(ctx, st.id, st.from)
		}
		if err != nil {
			return errors.Wrapf(err, "apply %s", st.id)
		}
	}
	return nil
}

// body returns the content of the main file of the resource with id in t.
func body(t *tree, id resource.ID) []byte {
	return t.raw[t.items[id].main()]
}

// decodeAs decodes the main file of the resource with id in t.
func decodeAs[T any](t *tree, id resource.ID) (T, error) {
	var v T
	if err := decode(body(t, id), &v); err != nil {
		var zero T
		return zero, err
	}
	return v, nil
}

func (a *applier) exists(id resource.ID) bool {
	_, ok := a.live.items[id]
	return ok
}

func (a *applier) put(ctx context.Context, id resource.ID, t *tree) error {
	switch id.Type {
	case resource.TypeChannel:
		return a.putChannel(ctx, t, id)
	case resource.TypeTask:
		return a.putTask(ctx, t, id)
	case resource.TypeArc:
		return a.putArc(ctx, t, id)
	case resource.TypeArcDirectory:
		return a.putDirectory(ctx, t, id)
	case resource.TypeProject:
		return a.putProject(ctx, t, id)
	case resource.TypeGroup:
		return a.putGroup(ctx, t, id)
	case resource.TypeSchematicSymbol:
		v, err := decodeAs[symbol.Symbol](t, id)
		if err != nil {
			return err
		}
		w := a.s.cfg.Schematic.Symbol.NewWriter(a.tx)
		if a.exists(id) {
			return w.Replace(ctx, v)
		}
		return w.Create(ctx, &v)
	case resource.TypeSchematic:
		v, err := decodeAs[schematic.Schematic](t, id)
		if err != nil {
			return err
		}
		w := a.s.cfg.Schematic.NewWriter(a.tx)
		if a.exists(id) {
			return w.Replace(ctx, v)
		}
		return w.Create(ctx, &v)
	case resource.TypeLineplot:
		v, err := decodeAs[lineplot.LinePlot](t, id)
		if err != nil {
			return err
		}
		w := a.s.cfg.LinePlot.NewWriter(a.tx)
		if a.exists(id) {
			return w.Replace(ctx, v)
		}
		return w.Create(ctx, &v)
	case resource.TypeLog:
		v, err := decodeAs[log.Log](t, id)
		if err != nil {
			return err
		}
		w := a.s.cfg.Log.NewWriter(a.tx)
		if a.exists(id) {
			return w.Replace(ctx, v)
		}
		return w.Create(ctx, &v)
	case resource.TypeTable:
		v, err := decodeAs[table.Table](t, id)
		if err != nil {
			return err
		}
		w := a.s.cfg.Table.NewWriter(a.tx)
		if a.exists(id) {
			return w.Replace(ctx, v)
		}
		return w.Create(ctx, &v)
	case resource.TypePanel:
		v, err := decodeAs[panel.Panel](t, id)
		if err != nil {
			return err
		}
		w := a.s.cfg.Panel.NewWriter(a.tx)
		if a.exists(id) {
			return w.Replace(ctx, v)
		}
		return w.Create(ctx, &v)
	}
	panic(errors.Newf("vcs: no put for resource type %s", id.Type))
}

func (a *applier) remove(ctx context.Context, id resource.ID) error {
	if id.Type == resource.TypeChannel {
		return nil
	}
	key, err := resource.ParseUUIDKey(id)
	if err != nil {
		return err
	}
	switch id.Type {
	case resource.TypeTask:
		return a.s.cfg.Task.NewWriter(a.tx).Delete(ctx, key, false)
	case resource.TypeArc:
		return a.s.cfg.Arc.NewWriter(a.tx).Delete(ctx, key)
	case resource.TypeArcDirectory:
		return a.s.cfg.Arc.NewWriter(a.tx).DeleteDirectory(ctx, key)
	case resource.TypeProject:
		return a.s.cfg.Project.NewWriter(a.tx).Delete(ctx, key)
	case resource.TypeGroup:
		return a.s.cfg.Group.NewWriter(a.tx).Delete(ctx, key)
	case resource.TypeSchematicSymbol:
		return a.s.cfg.Schematic.Symbol.NewWriter(a.tx).Delete(ctx, key)
	case resource.TypeSchematic:
		return a.s.cfg.Schematic.NewWriter(a.tx).Delete(ctx, key)
	case resource.TypeLineplot:
		return a.s.cfg.LinePlot.NewWriter(a.tx).Delete(ctx, key)
	case resource.TypeLog:
		return a.s.cfg.Log.NewWriter(a.tx).Delete(ctx, key)
	case resource.TypeTable:
		return a.s.cfg.Table.NewWriter(a.tx).Delete(ctx, key)
	case resource.TypePanel:
		return a.s.cfg.Panel.NewWriter(a.tx).Delete(ctx, key)
	}
	panic(errors.Newf("vcs: no remove for resource type %s", id.Type))
}

// putChannel renames and edits the calculation of an existing channel. It returns a
// validation error for a channel that does not exist on the Core, since a new channel
// gets a new key and tasks and visualizations name channels by key.
func (a *applier) putChannel(ctx context.Context, t *tree, id resource.ID) error {
	f, err := decodeAs[channelFile](t, id)
	if err != nil {
		return err
	}
	if _, done := a.channels[f.Key]; done {
		return nil
	}
	w := a.s.cfg.Channel.NewWriter(a.tx)
	var existing channel.Channel
	err = a.s.cfg.Channel.NewRetrieve().
		Where(channel.MatchKeys(f.Key)).
		Entry(&existing).
		Exec(ctx, a.tx)
	if err == nil {
		a.channels[f.Key] = f.Key
		var acts []channel.Action
		if string(existing.Name) != f.Name {
			acts = append(acts, channel.NewSetAction(
				channel.SetPayload{Path: []string{"name"}, Value: f.Name},
			))
		}
		if existing.Expression != f.Expression {
			acts = append(acts, channel.NewSetAction(
				channel.SetPayload{Path: []string{"expression"}, Value: f.Expression},
			))
		}
		if !slices.Equal(existing.Operations, f.Operations) {
			acts = append(acts, channel.NewSetAction(
				channel.SetPayload{Path: []string{"operations"}, Value: f.Operations},
			))
		}
		if len(acts) == 0 {
			return nil
		}
		_, err = w.Apply(ctx, f.Key, acts...)
		return err
	}
	if !errors.Is(err, query.ErrNotFound) {
		return err
	}
	return errors.Mark(
		errors.Newf(
			"Channel %s does not exist on this Core. Version control cannot create "+
				"it, since tasks and visualizations name channels by key.",
			f.Name,
		),
		validate.ErrValidation,
	)
}

func (a *applier) putTask(ctx context.Context, t *tree, id resource.ID) error {
	f, err := decodeAs[taskFile](t, id)
	if err != nil {
		return err
	}
	w := a.s.cfg.Task.NewWriter(a.tx)
	if !a.exists(id) {
		return w.Create(ctx, &task.Task{
			Key:    f.Key,
			Name:   f.Name,
			Type:   f.Type,
			Rack:   f.Rack,
			Config: f.Config,
		})
	}
	_, err = w.Apply(
		ctx,
		f.Key,
		task.NewSetAction(task.SetPayload{Path: []string{"name"}, Value: f.Name}),
		task.NewSetAction(task.SetPayload{Path: []string{"rack"}, Value: f.Rack}),
		task.NewSetAction(task.SetPayload{Path: []string{"config"}, Value: f.Config}),
	)
	return err
}

var errGraphEdit = errors.Wrap(
	validate.ErrValidation,
	"version control cannot change the graph of an existing Arc",
)

func (a *applier) putArc(ctx context.Context, t *tree, id resource.ID) error {
	f, err := decodeAs[arcFile](t, id)
	if err != nil {
		return err
	}
	src := string(t.raw[arcSource(t, id)])
	w := a.s.cfg.Arc.NewWriter(a.tx)
	dir := lo.FromPtrOr(f.Directory, a.root)
	var existing arc.Arc
	err = a.s.cfg.Arc.NewRetrieve().
		Where(arc.MatchKeys(f.Key)).
		Entry(&existing).
		Exec(ctx, a.tx)
	if errors.Is(err, query.ErrNotFound) {
		base := arc.Base{Key: f.Key, Name: f.Name}
		if f.Mode == arc.ModeGraph && f.Graph != nil {
			base.Source = arc.Source{Variant: arc.GraphSource{Graph: *f.Graph}}
		} else {
			base.Source = arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: src}}}
		}
		next := arc.Arc{Variant: arc.AutomationArc{Base: base}}
		if f.Type == arc.TypeModule {
			next = arc.Arc{Variant: arc.ModuleArc{Base: base}}
		}
		if err = w.Create(ctx, &next); err != nil || dir == a.root {
			return err
		}
		return w.Apply(ctx, f.Key, arc.NewSetDirectoryAction(
			arc.SetDirectoryPayload{Directory: dir},
		))
	}
	if err != nil {
		return err
	}
	b := existing.Base()
	var acts []arc.Action
	if b.Name != f.Name {
		acts = append(acts, arc.NewSetAction(
			arc.SetPayload{Path: []string{"name"}, Value: f.Name},
		))
	}
	if lo.FromPtrOr(b.Directory, a.root) != dir {
		acts = append(acts, arc.NewSetDirectoryAction(
			arc.SetDirectoryPayload{Directory: dir},
		))
	}
	switch cur := b.Source.Variant.(type) {
	case arc.TextSource:
		if f.Mode != arc.ModeText {
			return errGraphEdit
		}
		acts = append(acts, textActions(cur.Text.Doc, src)...)
	case arc.GraphSource:
		same, err := sameGraph(cur, f)
		if err != nil {
			return err
		}
		if !same {
			if f.Mode != arc.ModeGraph || f.Graph == nil {
				return errGraphEdit
			}
			graphActs, err := graphActions(existing, *f.Graph)
			if err != nil {
				return err
			}
			acts = append(acts, graphActs...)
		}
	}
	if len(acts) == 0 {
		return nil
	}
	return w.Apply(ctx, f.Key, acts...)
}

// graphActions returns the actions that rebuild the graph of the Arc a as next. It
// returns errGraphEdit when the actions cannot give next, such as for new functions.
func graphActions(a arc.Arc, next graph.Graph) ([]arc.Action, error) {
	cur := a.Base().Source.Variant.(arc.GraphSource).Graph
	acts := make([]arc.Action, 0, len(cur.Nodes)+2*len(next.Nodes)+len(next.Edges))
	for _, n := range cur.Nodes {
		acts = append(acts, arc.NewRemoveNodeAction(arc.RemoveNodePayload{Key: n.Key}))
	}
	for _, n := range next.Nodes {
		acts = append(acts, arc.NewSetNodeAction(arc.SetNodePayload{Node: n}))
		if in, ok := next.Inputs[n.Key]; ok {
			acts = append(acts, arc.NewSetNodeInputsAction(
				arc.SetNodeInputsPayload{Key: n.Key, Inputs: in},
			))
		}
	}
	for _, e := range next.Edges {
		acts = append(acts, arc.NewAddEdgeAction(arc.AddEdgePayload{Edge: e}))
	}
	out, err := arc.Reduce(a, acts...)
	if err != nil {
		return nil, err
	}
	src, ok := out.Base().Source.Variant.(arc.GraphSource)
	if !ok {
		return nil, errGraphEdit
	}
	same, err := sameGraph(src, arcFile{Graph: &next})
	if err != nil || !same {
		return nil, errors.Combine(err, errGraphEdit)
	}
	return acts, nil
}

// sameProgram reports whether the Arc with id runs the same program in a and b: the
// same source, mode, and graph. A rename or a move leaves the program the same.
func sameProgram(a, b *tree, id resource.ID) (bool, error) {
	if !bytes.Equal(a.raw[arcSource(a, id)], b.raw[arcSource(b, id)]) {
		return false, nil
	}
	fa, err := decodeAs[arcFile](a, id)
	if err != nil {
		return false, err
	}
	fb, err := decodeAs[arcFile](b, id)
	if err != nil {
		return false, err
	}
	ga, err := encode(fa.Graph)
	if err != nil {
		return false, err
	}
	gb, err := encode(fb.Graph)
	if err != nil {
		return false, err
	}
	return fa.Mode == fb.Mode && bytes.Equal(ga, gb), nil
}

// arcSource returns the path of the source file of the Arc with id in t.
func arcSource(t *tree, id resource.ID) string {
	return strings.TrimSuffix(t.items[id].main(), jsonExt) + arcExt
}

func sameGraph(cur arc.GraphSource, f arcFile) (bool, error) {
	if f.Graph == nil {
		return false, nil
	}
	a, err := encode(cur.Graph)
	if err != nil {
		return false, err
	}
	b, err := encode(*f.Graph)
	if err != nil {
		return false, err
	}
	return bytes.Equal(a, b), nil
}

// textActions returns the character edits that turn doc into next: a delete of the
// changed middle, then an insert of the new one. Open editors merge them like the edits
// of another user.
func textActions(doc text.Document, next string) []arc.Action {
	t := crdt.New(freeReplica(doc))
	t.Load(doc.Inserts, doc.Deletes)
	prev, want := []rune(t.String()), []rune(next)
	prefix := 0
	for prefix < len(prev) && prefix < len(want) && prev[prefix] == want[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(prev)-prefix && suffix < len(want)-prefix &&
		prev[len(prev)-1-suffix] == want[len(want)-1-suffix] {
		suffix++
	}
	var acts []arc.Action
	for _, d := range t.Delete(prefix, len(prev)-prefix-suffix) {
		acts = append(acts, arc.NewDeleteCharAction(arc.DeleteCharPayload{ID: d.ID}))
	}
	for _, op := range t.Insert(prefix, string(want[prefix:len(want)-suffix])) {
		acts = append(acts, arc.NewInsertCharAction(arc.InsertCharPayload{
			ID:     op.ID,
			Origin: op.Origin,
			Side:   op.Side,
			Char:   op.Char,
		}))
	}
	return acts
}

// freeReplica returns a replica that no character of doc names. Replicas 0 and 1 are
// the root and the seed.
func freeReplica(doc text.Document) uint32 {
	used := set.New[uint32](0, 1)
	for _, ins := range doc.Inserts {
		used.Add(ins.ID.Replica)
	}
	for {
		if r := rand.Uint32(); !used.Contains(r) {
			return r
		}
	}
}

func (a *applier) putDirectory(ctx context.Context, t *tree, id resource.ID) error {
	d, err := decodeAs[directoryRecord](t, id)
	if err != nil {
		return err
	}
	parent := lo.FromPtrOr(d.Parent, a.root)
	w := a.s.cfg.Arc.NewWriter(a.tx)
	existing, ok := a.dirs[d.Key]
	if !ok {
		return w.CreateDirectory(
			ctx,
			&directory.Directory{Key: d.Key, Name: d.Name},
			parent,
		)
	}
	var acts []directory.Action
	if existing.Name != d.Name {
		acts = append(acts, directory.NewSetAction(
			directory.SetPayload{Path: []string{"name"}, Value: d.Name},
		))
	}
	if lo.FromPtrOr(existing.Parent, a.root) != parent {
		acts = append(acts, directory.NewSetAction(
			directory.SetPayload{Path: []string{"parent"}, Value: parent},
		))
	}
	if len(acts) == 0 {
		return nil
	}
	return w.ApplyDirectory(ctx, d.Key, acts...)
}

func (a *applier) putProject(ctx context.Context, t *tree, id resource.ID) error {
	p, err := decodeAs[project.Project](t, id)
	if err != nil {
		return err
	}
	w := a.s.cfg.Project.NewWriter(a.tx)
	if !a.exists(id) {
		return w.Create(ctx, &p)
	}
	prev, err := decodeAs[project.Project](a.live, id)
	if err != nil {
		return err
	}
	var acts []project.Action
	if prev.Name != p.Name {
		acts = append(acts, project.NewSetAction(
			project.SetPayload{Path: []string{"name"}, Value: p.Name},
		))
	}
	if lo.FromPtr(prev.Group) != lo.FromPtr(p.Group) {
		acts = append(acts, project.NewSetAction(
			project.SetPayload{Path: []string{"group"}, Value: p.Group},
		))
	}
	if len(acts) == 0 {
		return nil
	}
	return w.Apply(ctx, p.Key, acts...)
}

func (a *applier) putGroup(ctx context.Context, t *tree, id resource.ID) error {
	g, err := decodeAs[group.Group](t, id)
	if err != nil {
		return err
	}
	w := a.s.cfg.Group.NewWriter(a.tx)
	if !a.exists(id) {
		return w.Create(ctx, &g)
	}
	prev, err := decodeAs[group.Group](a.live, id)
	if err != nil {
		return err
	}
	var acts []group.Action
	if prev.Name != g.Name {
		acts = append(acts, group.NewSetAction(
			group.SetPayload{Path: []string{"name"}, Value: g.Name},
		))
	}
	if lo.FromPtr(prev.Parent) != lo.FromPtr(g.Parent) {
		acts = append(acts, group.NewSetParentAction(
			group.SetParentPayload{Parent: g.Parent},
		))
	}
	if len(acts) == 0 {
		return nil
	}
	return w.Apply(ctx, g.Key, acts...)
}

// inverse returns what undoes the steps on live: the live files of each resource the
// steps change or remove, and the resources they create. A channel is never removed.
func inverse(steps []step, live *tree) Inverse {
	inv := Inverse{Files: []File{}, Removed: []string{}}
	for _, st := range steps {
		if _, ok := live.items[st.id]; ok {
			inv.Files = append(inv.Files, live.filesOf(st.id)...)
		} else if st.id.Type != resource.TypeChannel {
			inv.Removed = append(inv.Removed, st.id.String())
		}
	}
	slices.SortFunc(
		inv.Files,
		func(a, b File) int { return strings.Compare(a.Path, b.Path) },
	)
	slices.Sort(inv.Removed)
	return inv
}

// treeOfFiles parses the files of an Inverse. A path must be clean and relative.
func treeOfFiles(fs []File) (*tree, error) {
	raw := files{}
	for _, f := range fs {
		if f.Path == "" || path.Clean(f.Path) != f.Path || path.IsAbs(f.Path) ||
			strings.HasPrefix(f.Path, "../") {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"invalid file path %q",
				f.Path,
			)
		}
		raw[f.Path] = []byte(f.Content)
	}
	return parseTree(raw)
}
