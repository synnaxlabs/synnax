// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/validate"
)

// ValidateOwner checks the owner of a visualization, which belongs to a project. It
// returns a validation error on the project field when proj is nil.
func ValidateOwner(proj *Key) error {
	if proj != nil {
		return nil
	}
	return validate.PathedError(
		errors.Wrap(validate.ErrValidation, "must belong to a project"),
		"project",
	)
}

// GroupProject returns the project that a visualization of type t in the group with key
// belongs to: the parent at the top of the group's chain. It returns nil for a nil key,
// and a validation error on the group field when the group cannot hold t.
func GroupProject(
	ctx context.Context,
	tx gorp.Tx,
	key *group.Key,
	t resource.Type,
) (*Key, error) {
	if key == nil {
		return nil, nil
	}
	if err := group.CheckMember(ctx, tx, key, t, nil); err != nil {
		return nil, err
	}
	top, err := group.TopParent(ctx, tx, *key)
	if err != nil {
		return nil, validate.PathedError(err, "group")
	}
	return TopParentProject(top, t)
}

// ValidateGroup returns a validation error on the group field when g is not in project
// p. A nil g is valid.
func ValidateGroup(
	ctx context.Context,
	tx gorp.Tx,
	p Key,
	g *group.Key,
	t resource.Type,
) error {
	proj, err := GroupProject(ctx, tx, g, t)
	if err != nil || proj == nil || *proj == p {
		return err
	}
	return validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"group %s is not in project %s",
		*g,
		p,
	), "group")
}

// Placement is the group and project of a visualization.
type Placement struct {
	// Group is the group that holds the visualization, or nil.
	Group *group.Key
	// Project is the project that owns the visualization, or nil when a range owns it.
	Project *Key
}

// Place returns the placement of a visualization of type t after an edit moves it from
// prev to next, and the actions that the move adds to the edit, built with set and
// remove. A new group moves the visualization into the project that holds the group,
// which adds a set of the project when it changes. A new project with the same group
// moves the visualization out of its group, which adds a remove of the group. Place
// returns a validation error on the group field when the new group cannot hold t.
func Place[A any](
	ctx context.Context,
	tx gorp.Tx,
	t resource.Type,
	prev, next Placement,
	set func(patch.Set) A,
	remove func(patch.Remove) A,
) (Placement, []A, error) {
	if !groupsEqual(prev.Group, next.Group) {
		p, err := GroupProject(ctx, tx, next.Group, t)
		if err != nil {
			return Placement{}, nil, err
		}
		if p == nil || lo.FromPtr(next.Project) == *p {
			return next, nil, nil
		}
		next.Project = p
		return next, []A{set(patch.Set{Path: []string{"project"}, Value: *p})}, nil
	}
	if next.Group == nil || lo.FromPtr(prev.Project) == lo.FromPtr(next.Project) {
		return next, nil, nil
	}
	next.Group = nil
	return next, []A{remove(patch.Remove{Key: "group"})}, nil
}

func groupsEqual(a, b *group.Key) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// TopParentProject returns the project that a visualization of type t belongs to when
// it sits in a group whose top parent is top. It returns a validation error on the
// group field when top is not a project.
func TopParentProject(top *resource.ID, t resource.Type) (*Key, error) {
	if top == nil || top.Type != resource.TypeProject {
		return nil, validate.PathedError(errors.Wrapf(
			validate.ErrValidation,
			"a %s sits only in a group under a project",
			t,
		), "group")
	}
	k, err := KeyFromResourceID(*top)
	if err != nil {
		return nil, err
	}
	return &k, nil
}
