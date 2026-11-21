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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
)

// Path is the chain of groups from the top down to a group.
type Path []Group

// Ancestors returns every group above the groups with keys that is not one of them,
// each once, as seen by snap. The walk stops at a parent that is not a group.
func Ancestors(ctx context.Context, snap gorp.Snapshot, keys ...Key) ([]Group, error) {
	groups, _, err := chain(ctx, snap, keys)
	if err != nil {
		return nil, err
	}
	named := set.New(keys...)
	var out []Group
	for k, g := range groups {
		if !named.Contains(k) {
			out = append(out, g)
		}
	}
	return out, nil
}

// Paths returns the path of each group with keys, as seen by snap. A path stops at a
// group whose parent is not a group. It reads the groups one level at a time, and
// returns query.ErrNotFound when a group does not exist.
func Paths(ctx context.Context, snap gorp.Snapshot, keys ...Key) (map[Key]Path, error) {
	groups, parents, err := chain(ctx, snap, keys)
	if err != nil {
		return nil, err
	}
	paths := make(map[Key]Path, len(keys))
	for _, k := range keys {
		depth := 1
		for at, ok := parents[k]; ok; at, ok = parents[at] {
			depth++
		}
		p := make(Path, depth)
		at := k
		for i := depth - 1; i >= 0; i-- {
			p[i], at = groups[at], parents[at]
		}
		paths[k] = p
	}
	return paths, nil
}

// chain returns the groups with keys and every group above them, and the parent of
// each of those groups whose parent is a group, as seen by snap. It reads one level of
// groups at a time.
func chain(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []Key,
) (map[Key]Group, map[Key]Key, error) {
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
			return nil, nil, err
		}
		for _, g := range read {
			groups[g.Key] = g
		}
		next := make(set.Set[Key])
		for _, g := range read {
			if g.Parent == nil || g.Parent.Type != resource.TypeGroup {
				continue
			}
			parent, err := KeyFromResourceID(*g.Parent)
			if err != nil {
				return nil, nil, err
			}
			parents[g.Key] = parent
			if _, seen := groups[parent]; !seen {
				next.Add(parent)
			}
		}
		level = next.Slice()
	}
	return groups, parents, nil
}
