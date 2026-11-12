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
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// legacyWorkspaceType is the type projects had before the rename. The project table
// converts workspace edges only after this migration runs, so it reads them as project
// edges.
const legacyWorkspaceType = resource.Type("workspace")

// builtinScopes maps the name of each built-in root group to the scope of the groups
// directly under it.
var builtinScopes = map[string]resource.Type{
	"Channels":          resource.TypeChannel,
	"Devices":           resource.TypeDevice,
	"Projects":          resource.TypeProject,
	"Workspaces":        resource.TypeProject,
	"Schematic Symbols": resource.TypeSchematicSymbol,
}

// memberScopes maps each type that can sit in a group to the scope of the groups that
// hold it.
var memberScopes = map[resource.Type]resource.Type{
	resource.TypeChannel:         resource.TypeChannel,
	resource.TypeRack:            resource.TypeDevice,
	resource.TypeDevice:          resource.TypeDevice,
	resource.TypeProject:         resource.TypeProject,
	legacyWorkspaceType:          resource.TypeProject,
	resource.TypeSchematic:       resource.TypeProject,
	resource.TypeLineplot:        resource.TypeProject,
	resource.TypeLog:             resource.TypeProject,
	resource.TypeTable:           resource.TypeProject,
	resource.TypePanel:           resource.TypeProject,
	resource.TypeSchematicSymbol: resource.TypeSchematicSymbol,
}

// legacyRangesRoot names the built-in root of the legacy range groups. The range
// migration reads the edges under it, so ParentMigration keeps them.
const legacyRangesRoot = "Ranges"

// ParentMigration derives each group's parent and scope from the parent edge that
// points at it, then deletes every parent edge into a group outside the legacy range
// groups. A group under the ontology root is a built-in root: it gets no parent and the
// builtin scope, as do the legacy range groups. A group under a project, a rack, or a
// root in builtinScopes gets the scope they give. Any other group at the top of a chain
// gets no parent and the one scope that the members under it imply. The migration logs
// each group with no such scope and gives it the builtin scope. It also logs each
// membership outside the built-in roots that the member migrations drop.
var ParentMigration = migrate.New("v61_group_parent_and_scope", migrateParents)

// placement is the scope that ParentMigration gives a group.
type placement struct {
	scope resource.Type
	// placed is false when the migration found no scope for the group.
	placed bool
}

func migrateParents(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
	var groups []Group
	if err := gorp.NewRetrieve[Key, Group]().
		Entries(&groups).
		Exec(ctx, tx); err != nil {
		return err
	}
	edges, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.Type == ontology.RelationshipTypeParentOf &&
				(r.From.Type == resource.TypeGroup || r.To.Type == resource.TypeGroup)
		},
	)
	if err != nil {
		return err
	}
	byKey := make(map[string]*Group, len(groups))
	for i := range groups {
		byKey[groups[i].Key.String()] = &groups[i]
	}
	var (
		parents  = make(map[string]resource.ID, len(edges))
		children = make(map[string][]resource.ID, len(edges))
		into     []ontology.Relationship
	)
	for _, e := range edges {
		if e.From.Type == resource.TypeGroup {
			children[e.From.Key] = append(children[e.From.Key], e.To)
		}
		if e.To.Type != resource.TypeGroup {
			continue
		}
		into = append(into, e)
		if _, ok := parents[e.To.Key]; !ok {
			parent := e.From
			if parent.Type == legacyWorkspaceType {
				parent.Type = resource.TypeProject
			}
			parents[e.To.Key] = parent
		}
	}
	implied := func(key string) placement {
		scopes, visited := set.New[resource.Type](), set.New[string]()
		var walk func(key string)
		walk = func(key string) {
			visited.Add(key)
			for _, c := range children[key] {
				if c.Type != resource.TypeGroup {
					if s, ok := memberScopes[c.Type]; ok {
						scopes.Add(s)
					}
				} else if parents[c.Key] == groupID(key) && !visited.Contains(c.Key) {
					walk(c.Key)
				}
			}
		}
		walk(key)
		if len(scopes) != 1 {
			return placement{scope: resource.TypeBuiltin}
		}
		return placement{scope: scopes.Slice()[0], placed: true}
	}
	placements := make(map[string]placement, len(groups))
	var place func(key string, visited set.Set[string]) placement
	place = func(key string, visited set.Set[string]) placement {
		if p, ok := placements[key]; ok {
			return p
		}
		if visited.Contains(key) {
			return placement{scope: resource.TypeBuiltin}
		}
		visited.Add(key)
		p := func() placement {
			parent, ok := parents[key]
			switch {
			case !ok:
				return implied(key)
			case parent == ontology.RootID:
				return placement{scope: resource.TypeBuiltin, placed: true}
			case parent.Type == resource.TypeProject:
				return placement{scope: resource.TypeProject, placed: true}
			case parent.Type == resource.TypeRack:
				return placement{scope: resource.TypeDevice, placed: true}
			case parent.Type != resource.TypeGroup:
				return implied(key)
			}
			p, ok := byKey[parent.Key]
			if !ok {
				return implied(key)
			}
			if parents[parent.Key] != ontology.RootID {
				return place(parent.Key, visited)
			}
			if s, ok := builtinScopes[p.Name]; ok {
				return placement{scope: s, placed: true}
			}
			if p.Name == legacyRangesRoot {
				return placement{scope: resource.TypeBuiltin, placed: true}
			}
			return implied(key)
		}()
		placements[key] = p
		return p
	}
	for i := range groups {
		g := &groups[i]
		key := g.Key.String()
		p := place(key, set.New[string]())
		g.Scope, g.Parent = p.scope, nil
		if !p.placed {
			ins.L.Warn(
				"found no scope for group, so its members will sit in no group",
				zap.Stringer("key", g.Key),
				zap.String("name", g.Name),
			)
			continue
		}
		if g.Scope == resource.TypeBuiltin {
			continue
		}
		parent := parents[key]
		switch parent.Type {
		case resource.TypeProject, resource.TypeRack:
			g.Parent = &parent
		case resource.TypeGroup:
			if _, ok := byKey[parent.Key]; ok &&
				parents[parent.Key] != ontology.RootID {
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
		if parent == ontology.RootID {
			return byKey[key] != nil && byKey[key].Name == legacyRangesRoot
		}
		return parent.Type == resource.TypeGroup && underRanges(parent.Key, visited)
	}
	kept := set.New[resource.ID]()
	for _, e := range edges {
		if e.From.Type != resource.TypeGroup || e.To.Type == resource.TypeGroup {
			continue
		}
		g, ok := byKey[e.From.Key]
		if !ok ||
			parents[e.From.Key] == ontology.RootID ||
			underRanges(e.From.Key, set.New[string]()) {
			continue
		}
		// The member migrations keep only the first group of a member.
		if s, ok := memberScopes[e.To.Type]; ok && s == g.Scope &&
			!kept.Contains(e.To) {
			kept.Add(e.To)
			continue
		}
		ins.L.Warn(
			"dropped a member from a group that cannot hold it",
			zap.Stringer("group", g.Key),
			zap.String("group_name", g.Name),
			zap.Stringer("member", e.To),
		)
	}
	converted := slices.DeleteFunc(into, func(e ontology.Relationship) bool {
		return underRanges(e.To.Key, set.New[string]())
	})
	return ontology.DeleteRelationships(ctx, tx, converted...)
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
	// Holds reports whether a group with top parent top can hold e. Nil holds every
	// group.
	Holds func(e *E, top *resource.ID) bool
}

// NewMembersMigration returns a migration that sets the group of each member from the
// parent edge that points at it from a group whose scope holds Type. It then deletes
// every parent edge from a group to a member.
func NewMembersMigration[K gorp.Key, E gorp.Entry[K]](
	cfg MembersConfig[K, E],
) gorp.Migration {
	return migrate.New(
		cfg.Name,
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			return migrateMembers(ctx, tx, cfg, ins)
		},
	)
}

func migrateMembers[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	cfg MembersConfig[K, E],
	ins alamos.Instrumentation,
) error {
	edges, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.Type == ontology.RelationshipTypeParentOf &&
				r.From.Type == resource.TypeGroup &&
				r.To.Type == cfg.Type
		},
	)
	if err != nil || len(edges) == 0 {
		return err
	}
	var groups []Group
	if err = gorp.NewRetrieve[Key, Group]().Entries(&groups).Exec(ctx, tx); err != nil {
		return err
	}
	byKey := make(map[string]Group, len(groups))
	for _, g := range groups {
		byKey[g.Key.String()] = g
	}
	memberGroups := make(map[string]Group, len(edges))
	for _, e := range edges {
		g, ok := byKey[e.From.Key]
		if !ok || g.Scope != memberScopes[cfg.Type] {
			continue
		}
		if _, ok = memberGroups[e.To.Key]; !ok {
			memberGroups[e.To.Key] = g
		}
	}
	var members []E
	if err = gorp.NewRetrieve[K, E]().
		Where(gorp.Match(func(_ gorp.Context, e *E) (bool, error) {
			_, ok := memberGroups[gorp.FormatKey((*e).GorpKey())]
			return ok, nil
		})).
		Entries(&members).
		Exec(ctx, tx); err != nil {
		return err
	}
	var owners []ontology.Relationship
	for i := range members {
		id := gorp.FormatKey(members[i].GorpKey())
		g := memberGroups[id]
		var top *resource.ID
		if p, ok := TopParent(g, byKey); ok {
			top = &p
		}
		if cfg.Holds != nil && !cfg.Holds(&members[i], top) {
			ins.L.Warn(
				"dropped a member from a group outside its pinned parent",
				zap.Stringer("group", g.Key),
				zap.String("group_name", g.Name),
				zap.Stringer("member", resource.ID{Type: cfg.Type, Key: id}),
			)
			continue
		}
		cfg.SetGroup(&members[i], g.Key)
		if !cfg.ProjectOwned {
			continue
		}
		if top != nil && top.Type == resource.TypeProject {
			owners = append(owners, ontology.Relationship{
				From: *top,
				Type: ontology.RelationshipTypeParentOf,
				To:   resource.ID{Type: cfg.Type, Key: id},
			})
		}
	}
	if err = gorp.WrapWriter[K, E](tx).Set(ctx, members...); err != nil {
		return err
	}
	if err = ontology.DeleteRelationships(ctx, tx, edges...); err != nil {
		return err
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Set(ctx, owners...)
}

// TopParent returns the parent of the group at the top of g's chain, if it has one.
func TopParent(g Group, byKey map[string]Group) (resource.ID, bool) {
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

func groupID(key string) resource.ID {
	return resource.ID{Type: resource.TypeGroup, Key: key}
}
