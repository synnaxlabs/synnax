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
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// memberRule says which groups can hold a record of one type.
type memberRule struct {
	// scope is the scope of the groups that hold the type.
	scope resource.Type
	// pinned is true when the top parent of the group must be the one CheckMember gets.
	pinned bool
}

// memberRules holds the rule of each type that can sit in a group. A rack or project
// sits only in a group with no top parent, so no group can sit under its own member.
var memberRules = map[resource.Type]memberRule{
	resource.TypeChannel:         {scope: resource.TypeChannel},
	resource.TypeRack:            {scope: resource.TypeDevice, pinned: true},
	resource.TypeDevice:          {scope: resource.TypeDevice, pinned: true},
	resource.TypeProject:         {scope: resource.TypeProject, pinned: true},
	resource.TypeSchematic:       {scope: resource.TypeProject},
	resource.TypeLineplot:        {scope: resource.TypeProject},
	resource.TypeLog:             {scope: resource.TypeProject},
	resource.TypeTable:           {scope: resource.TypeProject},
	resource.TypePanel:           {scope: resource.TypeProject},
	resource.TypeSchematicSymbol: {scope: resource.TypeSchematicSymbol},
}

// CheckMember returns a validation error when a record of type t cannot sit in the
// group with key, as seen by tx. The group's scope must hold t. holder is the record
// that holds a device: its parent device, or else its rack. A device sits only in a
// group under its rack, and a device held by a device sits in no group. A rack or
// project sits only in a group with no top parent. Pass a nil holder for every type but
// a device. A nil key is no group, which every record can have.
func CheckMember(
	ctx context.Context,
	tx gorp.Tx,
	key *Key,
	t resource.Type,
	holder *resource.ID,
) error {
	if key == nil {
		return nil
	}
	if holder != nil && holder.Type == t {
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation,
			"a %s held by %s %s cannot sit in a group",
			t,
			holder.Type,
			holder.Key,
		), "group")
	}
	rule, ok := memberRules[t]
	if !ok {
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation, "a %s cannot sit in a group", t,
		), "group")
	}
	g, err := retrieveGroup(ctx, tx, *key)
	if err != nil {
		return validate.PathedError(err, "group")
	}
	if g.Scope != rule.scope {
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation,
			"a %s cannot sit in a %s group",
			t,
			g.Scope,
		), "group")
	}
	if !rule.pinned {
		return nil
	}
	actual, err := topParent(ctx, tx, g)
	if err != nil || sameID(actual, holder) {
		return err
	}
	if holder != nil {
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation,
			"group %s does not sit under %s %s",
			g.Name,
			holder.Type,
			holder.Key,
		), "group")
	}
	return validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"a %s cannot sit in a group under %s %s",
		t,
		actual.Type,
		actual.Key,
	), "group")
}

// pinnedMember returns the first member of a pinned type that sits in the group with
// key or in a group under it, or nil when there is none.
func (w Writer) pinnedMember(ctx context.Context, key Key) (*resource.ID, error) {
	deps, err := resource.Dependents(ctx, w.tx, w.table, key)
	if err != nil {
		return nil, err
	}
	for _, id := range deps {
		if id.Type != resource.TypeGroup {
			if memberRules[id.Type].pinned {
				return &id, nil
			}
			continue
		}
		child, err := KeyFromResourceID(id)
		if err != nil {
			return nil, err
		}
		if found, err := w.pinnedMember(ctx, child); found != nil || err != nil {
			return found, err
		}
	}
	return nil, nil
}

// checkMove returns a validation error when moving g under parent changes the top
// parent of a group that holds a member of a pinned type.
func (w Writer) checkMove(ctx context.Context, g Group, parent *resource.ID) error {
	before, err := topParent(ctx, w.tx, g)
	if err != nil {
		return err
	}
	after := parent
	if parent != nil && parent.Type == resource.TypeGroup {
		key, err := KeyFromResourceID(*parent)
		if err != nil {
			return validate.PathedError(err, "parent")
		}
		p, err := retrieveGroup(ctx, w.tx, key)
		if err != nil {
			return validate.PathedError(err, "parent")
		}
		if after, err = topParent(ctx, w.tx, p); err != nil {
			return err
		}
	}
	if sameID(before, after) {
		return nil
	}
	member, err := w.pinnedMember(ctx, g.Key)
	if member == nil || err != nil {
		return err
	}
	return validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"cannot move group %s under a new top parent because it holds %s %s",
		g.Name,
		member.Type,
		member.Key,
	), "parent")
}

// retrieveGroup returns the group with key, or a validation error when it does not
// exist.
func retrieveGroup(ctx context.Context, tx gorp.Tx, key Key) (Group, error) {
	var g Group
	err := gorp.NewRetrieve[Key, Group]().
		Where(gorp.MatchKeys[Key, Group](key)).
		Entry(&g).
		Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		return Group{}, errors.Wrapf(
			validate.ErrValidation,
			"group %s does not exist",
			key,
		)
	}
	return g, err
}

// topParent returns the parent of the group at the top of g's chain, or nil when that
// group has no parent.
func topParent(ctx context.Context, tx gorp.Tx, g Group) (*resource.ID, error) {
	for g.Parent != nil && g.Parent.Type == resource.TypeGroup {
		key, err := KeyFromResourceID(*g.Parent)
		if err != nil {
			return nil, err
		}
		if g, err = retrieveGroup(ctx, tx, key); err != nil {
			return nil, err
		}
	}
	return g.Parent, nil
}

func sameID(a, b *resource.ID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// CreateObjects returns the groups that a create of records needs update access on.
// stored holds the records that exist before the create. A new record joins the group
// it carries. A create keeps an existing record in its stored group, so it adds
// nothing.
func CreateObjects[K comparable, E interface{ GorpKey() K }](
	records []E,
	stored []E,
	groupOf func(E) *Key,
) []resource.ID {
	exists := make(set.Set[K], len(stored))
	for _, e := range stored {
		exists.Add(e.GorpKey())
	}
	var joined []*Key
	for _, e := range records {
		if !exists.Contains(e.GorpKey()) {
			joined = append(joined, groupOf(e))
		}
	}
	return MoveObjects(nil, nil, joined...)
}

// MoveObjects returns the resources that moving the records with ids into the groups in
// to needs update access on: the records, each group in from that holds one of them,
// and each group in to. A nil group adds nothing.
func MoveObjects(ids []resource.ID, from []*Key, to ...*Key) []resource.ID {
	objects := slices.Clone(ids)
	added := set.New[Key]()
	for _, k := range slices.Concat(from, to) {
		if k != nil && !added.Contains(*k) {
			added.Add(*k)
			objects = append(objects, ResourceID(*k))
		}
	}
	return objects
}
