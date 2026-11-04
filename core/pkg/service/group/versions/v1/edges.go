// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// builtinScopes maps the name of each built-in root group to the scope of the groups
// directly under it. A root not in the map gives its groups the builtin scope.
var builtinScopes = map[string]resource.Type{
	"Channels":          resource.TypeChannel,
	"Devices":           resource.TypeDevice,
	"Projects":          resource.TypeProject,
	"Schematic Symbols": resource.TypeSchematicSymbol,
}

var rootID = resource.ID{Type: resource.TypeBuiltin, Key: "root"}

// legacyRangesRoot names the built-in root of the legacy range groups. The range
// migration reads the edges under it, so ParentMigration keeps them.
const legacyRangesRoot = "Ranges"

// ParentMigration derives each group's parent and scope from the parent edge that
// points at it, then deletes every parent edge into a group outside the legacy range
// groups. A group under the ontology root is a built-in root: it gets no parent and the
// builtin scope. A group with no parent edge, or under a resource that cannot hold a
// group, gets the same.
var ParentMigration = migrate.New("v61_group_parent_and_scope", migrateParents)

func migrateParents(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	groups, err := collect[Key, Group](ctx, tx, func(Group) bool { return true })
	if err != nil {
		return err
	}
	edges, err := parentEdges(ctx, tx, resource.TypeGroup)
	if err != nil {
		return err
	}
	byKey := make(map[string]*Group, len(groups))
	for i := range groups {
		byKey[groups[i].Key.String()] = &groups[i]
	}
	parents := make(map[string]resource.ID, len(edges))
	for _, e := range edges {
		if _, ok := parents[e.To.Key]; !ok {
			parents[e.To.Key] = e.From
		}
	}
	var scopeOf func(key string, visited set.Set[string]) resource.Type
	scopeOf = func(key string, visited set.Set[string]) resource.Type {
		parent, ok := parents[key]
		if !ok || visited.Contains(key) {
			return resource.TypeBuiltin
		}
		visited.Add(key)
		switch parent.Type {
		case resource.TypeProject:
			return resource.TypeProject
		case resource.TypeRack:
			return resource.TypeDevice
		case resource.TypeGroup:
			p, ok := byKey[parent.Key]
			if !ok {
				return resource.TypeBuiltin
			}
			if parents[parent.Key] == rootID {
				if s, ok := builtinScopes[p.Name]; ok {
					return s
				}
				return resource.TypeBuiltin
			}
			return scopeOf(parent.Key, visited)
		}
		return resource.TypeBuiltin
	}
	for i := range groups {
		g := &groups[i]
		key := g.Key.String()
		g.Scope = scopeOf(key, set.New[string]())
		g.Parent = nil
		parent, ok := parents[key]
		if !ok || g.Scope == resource.TypeBuiltin {
			continue
		}
		switch parent.Type {
		case resource.TypeProject, resource.TypeRack:
			g.Parent = &parent
		case resource.TypeGroup:
			if parents[parent.Key] != rootID {
				g.Parent = &parent
			}
		}
	}
	if err = gorp.WrapWriter[Key, Group](tx).Set(ctx, groups...); err != nil {
		return err
	}
	var underRanges func(key string, visited set.Set[string]) bool
	underRanges = func(key string, visited set.Set[string]) bool {
		parent, ok := parents[key]
		if !ok || visited.Contains(key) {
			return false
		}
		visited.Add(key)
		if parent == rootID {
			return byKey[key].Name == legacyRangesRoot
		}
		return parent.Type == resource.TypeGroup && underRanges(parent.Key, visited)
	}
	converted := slices.DeleteFunc(edges, func(e ontology.Relationship) bool {
		return underRanges(e.To.Key, set.New[string]())
	})
	return deleteEdges(ctx, tx, converted)
}

// MembersConfig configures a migration that moves the members of type Type out of the
// ontology and into their group fields.
type MembersConfig[K gorp.Key, E gorp.Entry[K]] struct {
	// Name is the migration's key.
	Name string
	// Type is the resource type of the members.
	Type resource.Type
	// SetGroup sets the group of e.
	SetGroup func(e *E, group Key)
	// ProjectOwned adds a parent edge to each grouped member from the project at the
	// top of its group's chain.
	ProjectOwned bool
}

// NewMembersMigration returns a migration that sets the group of each member from the
// parent edge that points at it from a group, unless the group is a built-in root. It
// then deletes every parent edge from a group to a member.
func NewMembersMigration[K gorp.Key, E gorp.Entry[K]](
	cfg MembersConfig[K, E],
) gorp.Migration {
	return migrate.New(
		cfg.Name,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			return migrateMembers(ctx, tx, cfg)
		},
	)
}

func migrateMembers[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	cfg MembersConfig[K, E],
) error {
	edges, err := parentEdges(ctx, tx, cfg.Type)
	if err != nil {
		return err
	}
	edges = slices.DeleteFunc(edges, func(e ontology.Relationship) bool {
		return e.From.Type != resource.TypeGroup
	})
	if len(edges) == 0 {
		return nil
	}
	groups, err := collect[Key, Group](ctx, tx, func(Group) bool { return true })
	if err != nil {
		return err
	}
	byKey := make(map[string]Group, len(groups))
	for _, g := range groups {
		byKey[g.Key.String()] = g
	}
	memberGroups := make(map[string]Group, len(edges))
	for _, e := range edges {
		g, ok := byKey[e.From.Key]
		if !ok || g.Scope == resource.TypeBuiltin {
			continue
		}
		if _, ok = memberGroups[e.To.Key]; !ok {
			memberGroups[e.To.Key] = g
		}
	}
	members, err := collect[K, E](ctx, tx, func(e E) bool {
		_, ok := memberGroups[gorp.FormatKey(e.GorpKey())]
		return ok
	})
	if err != nil {
		return err
	}
	var owners []ontology.Relationship
	for i := range members {
		id := gorp.FormatKey(members[i].GorpKey())
		g := memberGroups[id]
		cfg.SetGroup(&members[i], g.Key)
		if !cfg.ProjectOwned {
			continue
		}
		if p, ok := topParent(g, byKey); ok && p.Type == resource.TypeProject {
			owners = append(owners, ontology.Relationship{
				From: p,
				Type: ontology.RelationshipTypeParentOf,
				To:   resource.ID{Type: cfg.Type, Key: id},
			})
		}
	}
	if err = gorp.WrapWriter[K, E](tx).Set(ctx, members...); err != nil {
		return err
	}
	if err = deleteEdges(ctx, tx, edges); err != nil {
		return err
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Set(ctx, owners...)
}

// topParent returns the parent of the group at the top of g's chain, if it has one.
func topParent(g Group, byKey map[string]Group) (resource.ID, bool) {
	visited := set.New[string]()
	for g.Parent != nil {
		if g.Parent.Type != resource.TypeGroup {
			return *g.Parent, true
		}
		if visited.Contains(g.Parent.Key) {
			return resource.ID{}, false
		}
		visited.Add(g.Parent.Key)
		next, ok := byKey[g.Parent.Key]
		if !ok {
			return resource.ID{}, false
		}
		g = next
	}
	return resource.ID{}, false
}

// parentEdges returns every parent edge that points at a resource of type t.
func parentEdges(
	ctx context.Context,
	tx gorp.Tx,
	t resource.Type,
) ([]ontology.Relationship, error) {
	return collect[string, ontology.Relationship](
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.Type == ontology.RelationshipTypeParentOf && r.To.Type == t
		},
	)
}

func deleteEdges(ctx context.Context, tx gorp.Tx, edges []ontology.Relationship) error {
	keys := make([]string, len(edges))
	for i, e := range edges {
		keys[i] = e.GorpKey()
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Delete(ctx, keys...)
}

func collect[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	keep func(E) bool,
) (out []E, err error) {
	iter, err := gorp.WrapReader[K, E](tx).OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if e := iter.Value(ctx); e != nil && keep(*e) {
			out = append(out, *e)
		}
	}
	return out, iter.Error()
}
