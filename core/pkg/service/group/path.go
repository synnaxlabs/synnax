// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"context"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/validate"
)

// Path is the chain of groups from the top down to a group.
type Path []Group

// String returns the names of the groups on p joined by validate.PathSeparator.
func (p Path) String() string {
	var b strings.Builder
	for i, g := range p {
		if i > 0 {
			b.WriteByte(validate.PathSeparator)
		}
		b.WriteString(g.Name)
	}
	return b.String()
}

// Paths returns the path of each group with keys, as seen by snap. A path stops at a
// group whose parent is not a group. It reads the groups one level at a time, and
// returns query.ErrNotFound when a group does not exist.
func Paths(ctx context.Context, snap gorp.Snapshot, keys ...Key) (map[Key]Path, error) {
	var (
		groups  = make(map[Key]Group, len(keys))
		parents = make(map[Key]Key, len(keys))
		level   = keys
	)
	for len(level) > 0 {
		var read []Group
		if err := gorp.NewRetrieve[Key, Group]().
			Where(gorp.MatchKeys[Key, Group](level...)).
			Entries(&read).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		level = nil
		for _, g := range read {
			groups[g.Key] = g
			if g.Parent == nil || g.Parent.Type != resource.TypeGroup {
				continue
			}
			parent, err := KeyFromResourceID(*g.Parent)
			if err != nil {
				return nil, err
			}
			parents[g.Key] = parent
			if _, read := groups[parent]; !read {
				level = append(level, parent)
			}
		}
	}
	paths := make(map[Key]Path, len(groups))
	var pathOf func(key Key) Path
	pathOf = func(key Key) Path {
		if p, ok := paths[key]; ok {
			return p
		}
		var p Path
		if parent, ok := parents[key]; ok {
			above := pathOf(parent)
			p = append(above[:len(above):len(above)], groups[key])
		} else {
			p = Path{groups[key]}
		}
		paths[key] = p
		return p
	}
	out := make(map[Key]Path, len(keys))
	for _, k := range keys {
		out[k] = pathOf(k)
	}
	return out, nil
}
