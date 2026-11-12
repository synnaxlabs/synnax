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
	"context"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// channelFile is the file form of a channel. The Core derives the rest of a channel.
type channelFile struct {
	Key        channel.Key         `json:"key"`
	Name       string              `json:"name"`
	DataType   telem.DataType      `json:"data_type"`
	IsIndex    bool                `json:"is_index"`
	Index      channel.Key         `json:"index"`
	Virtual    bool                `json:"virtual"`
	Expression string              `json:"expression"`
	Operations []channel.Operation `json:"operations"`
}

// taskFile is the file form of a task: its stored config and the rack it runs on.
type taskFile struct {
	Key    task.Key            `json:"key"`
	Name   string              `json:"name"`
	Type   string              `json:"type"`
	Rack   rack.Key            `json:"rack"`
	Config msgpack.EncodedJSON `json:"config"`
}

// arcFile is the file form of an Arc. The source text of a text Arc sits in a .arc file
// beside it.
type arcFile struct {
	Key  arc.Key  `json:"key"`
	Name string   `json:"name"`
	Type arc.Type `json:"type"`
	Mode arc.Mode `json:"mode"`
	// Directory is nil for an Arc in the root directory.
	Directory *directory.Key `json:"directory,omitzero"`
	Graph     *graph.Graph   `json:"graph,omitzero"`
}

// directoryFile is the file form of an Arc directory.
type directoryRecord struct {
	Key  directory.Key `json:"key"`
	Name string        `json:"name"`
	// Parent is nil for a directory in the root directory.
	Parent *directory.Key `json:"parent,omitzero"`
}

// node is a folder or a file of the repository while export lays it out.
type node struct {
	parent *node
	name   string
	key    string
	// ext is the extension of a file node. A folder has none.
	ext string
	// record is the file that holds the folder's own resource, if any.
	record string
	// claimed is the path element that the layout gave the node.
	claimed string
	folder  bool
	files   files
	id      resource.ID
}

// layout places resources in folders and gives every node a unique path element.
type layout struct {
	nodes    []*node
	children map[*node][]*node
}

func newLayout() *layout { return &layout{children: map[*node][]*node{}} }

func (l *layout) folder(parent *node, name, key, record string) *node {
	n := &node{parent: parent, name: name, key: key, folder: true, record: record}
	if parent != nil {
		l.children[parent] = append(l.children[parent], n)
	}
	return n
}

func (l *layout) file(parent *node, id resource.ID, name, ext string, fs files) {
	n := &node{parent: parent, name: name, key: id.Key, ext: ext, files: fs, id: id}
	l.children[parent] = append(l.children[parent], n)
	l.nodes = append(l.nodes, n)
}

// resolve claims a path element for every child of root, in a walk sorted by name then
// key. Folders and files share the claims, and a file claims the name of each of its
// files, so no two entries of a folder share a name.
func (l *layout) resolve(root *node) error {
	kids := l.children[root]
	slices.SortFunc(kids, func(a, b *node) int {
		return cmpStrings(a.name, b.name, a.key, b.key)
	})
	taken := set.New[string]()
	for _, r := range []string{groupFile, projectFile, directoryFile} {
		taken.Add(filename.Fold(r))
	}
	for _, k := range kids {
		name, err := filename.Sanitize(k.name, k.ext)
		if err != nil {
			return err
		}
		exts := []string{k.ext}
		for ext := range k.files {
			if ext != k.ext {
				exts = append(exts, ext)
			}
		}
		k.claimed = claim(taken, name, k.ext, exts)
		if k.folder {
			if err = l.resolve(k); err != nil {
				return err
			}
		}
	}
	return nil
}

// claim returns name, or name with the first numeric suffix that frees it, such that
// the name with each of exts in place of ext is free. It takes all of those names.
func claim(taken set.Set[string], name, ext string, exts []string) string {
	free := func(n string) bool {
		base := strings.TrimSuffix(n, ext)
		return !slices.ContainsFunc(exts, func(e string) bool {
			return taken.Contains(filename.Fold(base + e))
		})
	}
	claimed := name
	for i := 1; !free(claimed); i++ {
		claimed = filename.WithSuffix(name, " ("+strconv.Itoa(i)+")", ext)
	}
	base := strings.TrimSuffix(claimed, ext)
	for _, e := range exts {
		taken.Add(filename.Fold(base + e))
	}
	return claimed
}

func (n *node) path() string {
	if n.parent == nil {
		return n.claimed
	}
	return path.Join(n.parent.path(), n.claimed)
}

// export reads the live configuration from snap as a tree of files.
func (s *Service) export(ctx context.Context, snap gorp.Snapshot) (*tree, error) {
	l := newLayout()
	roots := map[string]*node{}
	for _, r := range []string{
		channelsRoot, tasksRoot, arcRoot, symbolsRoot, groupsRoot, projectsRoot,
	} {
		roots[r] = &node{name: r, claimed: r, folder: true}
	}
	t := newTree()
	records := map[*node]recordFile{}
	if err := s.exportChannels(ctx, snap, l, roots[channelsRoot]); err != nil {
		return nil, err
	}
	if err := s.exportTasks(ctx, snap, l, roots[tasksRoot]); err != nil {
		return nil, err
	}
	if err := s.exportArcs(ctx, snap, l, roots[arcRoot], records); err != nil {
		return nil, err
	}
	if err := s.exportProjects(ctx, snap, l, roots, records); err != nil {
		return nil, err
	}
	for _, r := range roots {
		if err := l.resolve(r); err != nil {
			return nil, err
		}
	}
	for n, rec := range records {
		t.add(rec.id, rec.name, files{path.Join(n.path(), n.record): rec.body})
	}
	for _, n := range l.nodes {
		dir := n.parent.path()
		base := strings.TrimSuffix(n.claimed, n.ext)
		fs := files{}
		for ext, b := range n.files {
			fs[path.Join(dir, base+ext)] = b
		}
		t.add(n.id, n.name, fs)
	}
	return t, nil
}

// recordFile is the file that holds the resource of a folder.
type recordFile struct {
	id   resource.ID
	name string
	body []byte
}

func (s *Service) exportChannels(
	ctx context.Context,
	snap gorp.Snapshot,
	l *layout,
	root *node,
) error {
	var chs []channel.Channel
	q := s.cfg.Channel.NewRetrieve().Where(channel.MatchInternal(false))
	// The Core makes an index for each calculated channel outside its group, so an
	// index that a Core channel uses stays out too.
	coreIndexes := set.New[channel.Key]()
	if len(s.cfg.CoreGroups) > 0 {
		var core []channel.Channel
		if err := s.cfg.Channel.NewRetrieve().
			Where(channel.MatchGroups(s.cfg.CoreGroups...)).
			Entries(&core).
			Exec(ctx, snap); err != nil {
			return err
		}
		for _, ch := range core {
			coreIndexes.Add(ch.Index())
		}
		q = q.Where(channel.Not(channel.MatchGroups(s.cfg.CoreGroups...)))
	}
	if err := q.Entries(&chs).Exec(ctx, snap); err != nil {
		return err
	}
	for _, ch := range chs {
		if coreIndexes.Contains(ch.Key()) {
			continue
		}
		f := channelFile{
			Key:        ch.Key(),
			Name:       ch.Name,
			DataType:   ch.DataType,
			IsIndex:    ch.IsIndex,
			Index:      ch.Index(),
			Virtual:    ch.Virtual,
			Expression: ch.Expression,
			Operations: ch.Operations,
		}
		b, err := encode(f)
		if err != nil {
			return err
		}
		l.file(root, ch.ResourceID(), ch.Name, jsonExt, files{jsonExt: b})
	}
	return nil
}

// versionedTask reports whether a task belongs in a configuration. An Arc task is part
// of its Arc, and an internal task is part of the Core.
func versionedTask(t task.Task) bool {
	return !t.Internal && !t.Snapshot && t.Type != arc.TaskType
}

func (s *Service) exportTasks(
	ctx context.Context,
	snap gorp.Snapshot,
	l *layout,
	root *node,
) error {
	var tasks []task.Task
	if err := s.cfg.Task.NewRetrieve().
		Where(task.MatchInternal(false)).
		Where(task.MatchSnapshot(false)).
		Entries(&tasks).
		Exec(ctx, snap); err != nil {
		return err
	}
	for _, t := range tasks {
		if !versionedTask(t) {
			continue
		}
		b, err := encode(taskFile{
			Key:    t.Key,
			Name:   t.Name,
			Type:   t.Type,
			Rack:   t.Rack,
			Config: t.Config,
		})
		if err != nil {
			return err
		}
		l.file(root, task.ResourceID(t.Key), t.Name, jsonExt, files{jsonExt: b})
	}
	return nil
}

func (s *Service) exportArcs(
	ctx context.Context,
	snap gorp.Snapshot,
	l *layout,
	root *node,
	records map[*node]recordFile,
) error {
	var dirs []directory.Directory
	if err := s.cfg.Arc.NewDirectoryRetrieve().
		Entries(&dirs).
		Exec(ctx, snap); err != nil {
		return err
	}
	var rootKey directory.Key
	byKey := map[directory.Key]directory.Directory{}
	for _, d := range dirs {
		if d.Parent == nil {
			rootKey = d.Key
		}
		byKey[d.Key] = d
	}
	folders := map[directory.Key]*node{rootKey: root}
	var folderOf func(k directory.Key) (*node, error)
	folderOf = func(k directory.Key) (*node, error) {
		if n, ok := folders[k]; ok {
			return n, nil
		}
		d := byKey[k]
		parent, err := folderOf(*d.Parent)
		if err != nil {
			return nil, err
		}
		n := l.folder(parent, d.Name, d.Key.String(), directoryFile)
		folders[k] = n
		b, err := encode(directoryRecord{
			Key:    d.Key,
			Name:   d.Name,
			Parent: rootless(d.Parent, rootKey),
		})
		if err != nil {
			return nil, err
		}
		records[n] = recordFile{id: directory.ResourceID(d.Key), name: d.Name, body: b}
		return n, nil
	}
	for _, d := range dirs {
		if _, err := folderOf(d.Key); err != nil {
			return err
		}
	}
	var arcs []arc.Arc
	if err := s.cfg.Arc.NewRetrieve().Entries(&arcs).Exec(ctx, snap); err != nil {
		return err
	}
	for _, a := range arcs {
		b := a.Base()
		dirKey := rootKey
		if b.Directory != nil {
			dirKey = *b.Directory
		}
		parent, err := folderOf(dirKey)
		if err != nil {
			return err
		}
		f := arcFile{
			Key:       b.Key,
			Name:      b.Name,
			Type:      a.Type(),
			Mode:      b.Source.Mode(),
			Directory: rootless(b.Directory, rootKey),
		}
		fs := files{}
		switch src := b.Source.Variant.(type) {
		case arc.TextSource:
			fs[arcExt] = []byte(src.Text.Materialize().Raw)
		case arc.GraphSource:
			f.Graph = &src.Graph
		}
		if fs[jsonExt], err = encode(f); err != nil {
			return err
		}
		l.file(parent, a.ResourceID(), b.Name, jsonExt, fs)
	}
	return nil
}

// rootless returns nil for a reference to the root directory.
func rootless(k *directory.Key, root directory.Key) *directory.Key {
	if k == nil || *k == root {
		return nil
	}
	return k
}

func (s *Service) exportProjects(
	ctx context.Context,
	snap gorp.Snapshot,
	l *layout,
	roots map[string]*node,
	records map[*node]recordFile,
) error {
	var projects []project.Project
	if err := s.cfg.Project.NewRetrieve().
		Entries(&projects).
		Exec(ctx, snap); err != nil {
		return err
	}
	var groups []group.Group
	if err := s.cfg.Group.NewRetrieve().Entries(&groups).Exec(ctx, snap); err != nil {
		return err
	}
	folders := map[resource.ID]*node{}
	for _, p := range projects {
		n := l.folder(roots[projectsRoot], p.Name, p.Key.String(), projectFile)
		b, err := encode(p)
		if err != nil {
			return err
		}
		folders[p.ResourceID()] = n
		records[n] = recordFile{id: p.ResourceID(), name: p.Name, body: b}
	}
	groupsByID := map[resource.ID]group.Group{}
	for _, g := range groups {
		groupsByID[g.ResourceID()] = g
	}
	var folderOf func(id resource.ID) (*node, bool, error)
	folderOf = func(id resource.ID) (*node, bool, error) {
		if n, ok := folders[id]; ok {
			return n, true, nil
		}
		g, ok := groupsByID[id]
		if !ok {
			return nil, false, nil
		}
		var parent *node
		switch {
		case g.Parent != nil:
			var err error
			if parent, ok, err = folderOf(*g.Parent); !ok || err != nil {
				return nil, false, err
			}
		case g.Scope == resource.TypeSchematicSymbol:
			parent = roots[symbolsRoot]
		case g.Scope == resource.TypeProject:
			parent = roots[groupsRoot]
		default:
			return nil, false, nil
		}
		n := l.folder(parent, g.Name, g.Key.String(), groupFile)
		b, err := encode(g)
		if err != nil {
			return nil, false, err
		}
		folders[id] = n
		records[n] = recordFile{id: g.ResourceID(), name: g.Name, body: b}
		return n, true, nil
	}
	for _, g := range groups {
		if _, _, err := folderOf(g.ResourceID()); err != nil {
			return err
		}
	}
	member := func(
		id resource.ID,
		name string,
		grp *group.Key,
		proj *project.Key,
		doc any,
	) error {
		var parent *node
		if grp != nil {
			var err error
			if parent, _, err = folderOf(group.ResourceID(*grp)); err != nil {
				return err
			}
		}
		if parent == nil && proj != nil {
			parent = folders[project.ResourceID(*proj)]
		}
		if parent == nil {
			return nil
		}
		b, err := encode(doc)
		if err != nil {
			return err
		}
		ext := "." + string(id.Type) + jsonExt
		l.file(parent, id, name, ext, files{ext: b})
		return nil
	}
	var symbols []symbol.Symbol
	if err := s.cfg.Schematic.Symbol.NewRetrieve().
		Entries(&symbols).
		Exec(ctx, snap); err != nil {
		return err
	}
	for _, sym := range symbols {
		parent := roots[symbolsRoot]
		if sym.Group != nil {
			if n, ok, err := folderOf(group.ResourceID(*sym.Group)); err != nil {
				return err
			} else if ok {
				parent = n
			}
		}
		b, err := encode(sym)
		if err != nil {
			return err
		}
		l.file(parent, symbol.ResourceID(sym.Key), sym.Name, jsonExt, files{jsonExt: b})
	}
	var schematics []schematic.Schematic
	if err := s.cfg.Schematic.NewRetrieve().
		Entries(&schematics).
		Exec(ctx, snap); err != nil {
		return err
	}
	for _, d := range schematics {
		if d.Range != nil || d.Snapshot {
			continue
		}
		if err := member(d.ResourceID(), d.Name, d.Group, d.Project, d); err != nil {
			return err
		}
	}
	var plots []lineplot.LinePlot
	if err := s.cfg.LinePlot.NewRetrieve().Entries(&plots).Exec(ctx, snap); err != nil {
		return err
	}
	for _, d := range plots {
		if d.Range != nil {
			continue
		}
		if err := member(d.ResourceID(), d.Name, d.Group, d.Project, d); err != nil {
			return err
		}
	}
	var logs []log.Log
	if err := s.cfg.Log.NewRetrieve().Entries(&logs).Exec(ctx, snap); err != nil {
		return err
	}
	for _, d := range logs {
		if d.Range != nil {
			continue
		}
		if err := member(d.ResourceID(), d.Name, d.Group, d.Project, d); err != nil {
			return err
		}
	}
	var tables []table.Table
	if err := s.cfg.Table.NewRetrieve().Entries(&tables).Exec(ctx, snap); err != nil {
		return err
	}
	for _, d := range tables {
		if d.Range != nil {
			continue
		}
		if err := member(d.ResourceID(), d.Name, d.Group, d.Project, d); err != nil {
			return err
		}
	}
	var panels []panel.Panel
	if err := s.cfg.Panel.NewRetrieve().Entries(&panels).Exec(ctx, snap); err != nil {
		return err
	}
	for _, d := range panels {
		if d.Range != nil {
			continue
		}
		d.Root = withoutDiffTabs(d.Root)
		if err := member(d.ResourceID(), d.Name, d.Group, d.Project, d); err != nil {
			return err
		}
	}
	return nil
}

// diffViewType is the type of the Console view that shows a version control diff.
const diffViewType = "vcs_diff"

// withoutDiffTabs returns the panel tree without its diff tabs. A diff shows version
// control state, so opening one must not change the panel it opens in.
func withoutDiffTabs(n panel.Node) panel.Node {
	switch v := n.Variant.(type) {
	case panel.LeafNode:
		tabs := make([]panel.Tab, 0, len(v.Tabs))
		for _, t := range v.Tabs {
			if view, ok := t.Variant.(panel.ViewTab); ok && view.Type == diffViewType {
				continue
			}
			tabs = append(tabs, t)
		}
		v.Tabs = tabs
		return panel.Node{Variant: v}
	case panel.SplitNode:
		v.First = withoutDiffTabs(v.First)
		v.Last = withoutDiffTabs(v.Last)
		return panel.Node{Variant: v}
	default:
		return n
	}
}
