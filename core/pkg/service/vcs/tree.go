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
	"encoding/hex"
	"encoding/json/jsontext"
	"hash/fnv"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
)

const (
	readmePath    = "README.md"
	manifestPath  = "synnax.json"
	channelsRoot  = "channels"
	tasksRoot     = "tasks"
	arcRoot       = "arc"
	symbolsRoot   = "symbols"
	groupsRoot    = "groups"
	projectsRoot  = "projects"
	groupFile     = "_group.json"
	projectFile   = "_project.json"
	directoryFile = "_directory.json"
	jsonExt       = ".json"
	arcExt        = ".arc"
)

// memberTypes are the types a project holds as files named <name>.<type>.json.
var memberTypes = []resource.Type{
	resource.TypeSchematic,
	resource.TypeLineplot,
	resource.TypeLog,
	resource.TypeTable,
	resource.TypePanel,
}

// files maps a repository path to its content.
type files map[string][]byte

// item is one resource and its files.
type item struct {
	id   resource.ID
	name string
	// paths holds the paths of the files of the resource, sorted.
	paths []string
}

// main returns the path of the JSON file of the resource.
func (it *item) main() string {
	for _, p := range it.paths {
		if strings.HasSuffix(p, jsonExt) {
			return p
		}
	}
	return it.paths[0]
}

// tree is a configuration as files, indexed by resource.
type tree struct {
	raw   files
	items map[resource.ID]*item
}

func newTree() *tree {
	return &tree{raw: files{}, items: map[resource.ID]*item{}}
}

// add puts the files of a resource in t.
func (t *tree) add(id resource.ID, name string, fs files) {
	it := &item{id: id, name: name}
	for p, b := range fs {
		t.raw[p] = b
		it.paths = append(it.paths, p)
	}
	slices.Sort(it.paths)
	t.items[id] = it
}

// filesOf returns the files of the resource with id, sorted by path, or nil when t does
// not hold it.
func (t *tree) filesOf(id resource.ID) []File {
	it, ok := t.items[id]
	if !ok {
		return nil
	}
	out := make([]File, len(it.paths))
	for i, p := range it.paths {
		out[i] = File{Path: p, Content: string(t.raw[p])}
	}
	return out
}

// same reports whether the resource with id has the same files in t and o.
func (t *tree) same(o *tree, id resource.ID) bool {
	a, aOK := t.items[id]
	b, bOK := o.items[id]
	if aOK != bOK {
		return false
	}
	if !aOK {
		return true
	}
	if !slices.Equal(a.paths, b.paths) {
		return false
	}
	for _, p := range a.paths {
		if !bytes.Equal(t.raw[p], o.raw[p]) {
			return false
		}
	}
	return true
}

// ids returns the IDs of the resources that t or o hold.
func (t *tree) ids(o *tree) []resource.ID {
	all := slices.Collect(maps.Keys(t.items))
	for id := range o.items {
		if _, ok := t.items[id]; !ok {
			all = append(all, id)
		}
	}
	return all
}

// diff returns the resources that differ from base to t, sorted by type, name, and key.
func (t *tree) diff(base *tree) []Change {
	changes := make([]Change, 0)
	for _, id := range t.ids(base) {
		if t.same(base, id) {
			continue
		}
		changes = append(changes, t.change(base, id))
	}
	sortChanges(changes)
	return changes
}

// change describes how the resource with id differs from base to t.
func (t *tree) change(base *tree, id resource.ID) Change {
	c := Change{Resource: id.String(), Type: string(id.Type)}
	it, inT := t.items[id]
	old, inBase := base.items[id]
	switch {
	case inT && !inBase:
		c.Kind, c.Name, c.Path = ChangeAdded, it.name, it.main()
	case !inT && inBase:
		c.Kind, c.Name, c.Path = ChangeDeleted, old.name, old.main()
	default:
		c.Kind, c.Name, c.Path = ChangeModified, it.name, it.main()
	}
	if inT {
		c.Digest = t.digest(it)
	}
	return c
}

// digest hashes the paths and contents of the files of it.
func (t *tree) digest(it *item) string {
	h := fnv.New64a()
	for _, p := range it.paths {
		h.Write([]byte(p))
		h.Write([]byte{0})
		h.Write(t.raw[p])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sortChanges(changes []Change) {
	slices.SortFunc(changes, func(a, b Change) int {
		return cmpStrings(a.Type, b.Type, a.Name, b.Name, a.Resource, b.Resource)
	})
}

// cmpStrings compares pairs of strings in order until one pair differs.
func cmpStrings(pairs ...string) int {
	for i := 0; i+1 < len(pairs); i += 2 {
		if c := strings.Compare(pairs[i], pairs[i+1]); c != 0 {
			return c
		}
	}
	return 0
}

// header is the part of every resource file that identifies the resource.
type header struct {
	Key  jsontext.Value `json:"key"`
	Name string         `json:"name"`
}

// parseTree indexes the files of a repository by resource. Files that the layout does
// not name, such as README.md, stay in raw but belong to no resource.
func parseTree(raw files) (*tree, error) {
	t := &tree{raw: raw, items: map[resource.ID]*item{}}
	byJSON := map[string]*item{}
	for p, b := range raw {
		typ, ok := typeOf(p)
		if !ok {
			continue
		}
		var h header
		if err := decode(b, &h); err != nil {
			return nil, errors.Wrapf(err, "read %s", p)
		}
		key, err := keyOf(h.Key)
		if err != nil {
			return nil, errors.Wrapf(err, "read %s", p)
		}
		id := resource.ID{Type: typ, Key: key}
		it := &item{id: id, name: h.Name, paths: []string{p}}
		t.items[id] = it
		byJSON[p] = it
	}
	for p := range raw {
		if !strings.HasPrefix(p, arcRoot+"/") || path.Ext(p) != arcExt {
			continue
		}
		if it, ok := byJSON[strings.TrimSuffix(p, arcExt)+jsonExt]; ok {
			it.paths = append(it.paths, p)
			slices.Sort(it.paths)
		}
	}
	return t, nil
}

// keyOf returns the resource key that a JSON key holds: a string as is, a number as its
// decimal text.
func keyOf(v jsontext.Value) (string, error) {
	switch v.Kind() {
	case '"':
		var s string
		if err := decode(v, &s); err != nil {
			return "", err
		}
		return s, nil
	case '0':
		return string(v), nil
	}
	return "", errors.New("file has no key")
}

// typeOf returns the type of the resource whose JSON file sits at p.
func typeOf(p string) (resource.Type, bool) {
	if path.Ext(p) != jsonExt {
		return "", false
	}
	dir, base := path.Split(p)
	root, _, _ := strings.Cut(dir, "/")
	switch root {
	case channelsRoot:
		return resource.TypeChannel, dir == channelsRoot+"/"
	case tasksRoot:
		return resource.TypeTask, dir == tasksRoot+"/"
	case arcRoot:
		if base == directoryFile {
			return resource.TypeArcDirectory, true
		}
		return resource.TypeArc, true
	case symbolsRoot:
		if base == groupFile {
			return resource.TypeGroup, true
		}
		return resource.TypeSchematicSymbol, true
	case groupsRoot, projectsRoot:
		switch base {
		case projectFile:
			return resource.TypeProject, root == projectsRoot
		case groupFile:
			return resource.TypeGroup, true
		}
		stem := strings.TrimSuffix(base, jsonExt)
		for _, t := range memberTypes {
			if strings.HasSuffix(stem, "."+string(t)) {
				return t, true
			}
		}
	}
	return "", false
}
