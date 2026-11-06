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
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// ValidateOwner checks the owner of a visualization, which belongs to exactly one
// project or range. It returns a validation error on the project field when proj and
// rng are both set or both nil.
func ValidateOwner(proj *Key, rng *ranger.Key) error {
	if (proj == nil) != (rng == nil) {
		return nil
	}
	return validate.PathedError(errors.Wrap(
		validate.ErrValidation,
		"must belong to exactly one of a project and a range",
	), "project")
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
	return &k, err
}

// OwnerID returns the ID of the project or range that owns a visualization, or nil when
// both are nil.
func OwnerID(proj *Key, rng *ranger.Key) *resource.ID {
	switch {
	case proj != nil:
		return new(ResourceID(*proj))
	case rng != nil:
		return new(ranger.ResourceID(*rng))
	default:
		return nil
	}
}

// CreateObjects returns the owners that a create of visualizations needs update access
// on. stored holds the visualizations that exist before the create. A new
// visualization joins the owner it carries. A create keeps an existing visualization
// with its stored owner, so it adds nothing.
func CreateObjects[K comparable, E interface{ GorpKey() K }](
	records []E,
	stored []E,
	ownerOf func(E) *resource.ID,
) []resource.ID {
	exists := make(set.Set[K], len(stored))
	for _, e := range stored {
		exists.Add(e.GorpKey())
	}
	var joined []*resource.ID
	for _, e := range records {
		if !exists.Contains(e.GorpKey()) {
			joined = append(joined, ownerOf(e))
		}
	}
	return MoveObjects(nil, nil, joined...)
}

// MoveObjects returns the resources that moving the visualizations with ids to the
// owners in to needs update access on: the visualizations, each owner in from that
// holds one of them, and each owner in to. A nil owner adds nothing.
func MoveObjects(ids []resource.ID, from []*resource.ID, to ...*resource.ID) []resource.ID {
	objects := slices.Clone(ids)
	added := set.New[resource.ID]()
	for _, id := range slices.Concat(from, to) {
		if id != nil && !added.Contains(*id) {
			added.Add(*id)
			objects = append(objects, *id)
		}
	}
	return objects
}
