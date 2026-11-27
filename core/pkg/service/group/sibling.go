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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Sibling is the name of a record or a group in the group with key Parent, or at the
// top when Parent is the zero key. A namespaced scope keeps each Sibling unique among
// its records and its groups.
type Sibling struct {
	Parent Key
	Name   string
}

// siblingSpace returns the name of the unique space of the siblings of scope.
func siblingSpace(scope resource.Type) string { return "namespace." + string(scope) }

// NewMemberIndex returns the index that keeps the name of each record of the namespaced
// scope unique among the records and the groups in its group. sibling returns the
// Sibling of a record. Names compare as stored, so each writer of a record, a migration
// through gorp.WrapWriter included, must store its name in one normal form.
func NewMemberIndex[K gorp.Key, E gorp.Entry[K]](
	scope resource.Type,
	sibling func(*E) Sibling,
) *gorp.UniqueIndex[K, E, Sibling] {
	return gorp.NewUniqueIndex(
		"sibling",
		siblingSpace(scope),
		func(e *E) (Sibling, bool) { return sibling(e), true },
	)
}

// SiblingError returns err as a validation error that names the kind of the holder of
// the name when err is a clash of sibling names, and returns err unchanged otherwise.
// The gorp.UniqueError of the clash stays in the chain of the error.
func SiblingError(err error) error {
	var clash gorp.UniqueError
	if !errors.As(err, &clash) {
		return err
	}
	s, ok := clash.Value.(Sibling)
	if !ok {
		return err
	}
	where := "in the same group"
	if s.Parent == uuid.Nil() {
		where = "at the top level"
	}
	return validate.PathedError(validate.WithCause(
		err,
		"a %s named '%s' already exists %s",
		clash.Holder.Table,
		s.Name,
		where,
	), "name")
}

// siblingIndex is the index of the names of the groups of one namespaced scope.
type siblingIndex = *gorp.UniqueIndex[Key, Group, Sibling]

// newSiblingIndexes returns the index of the names of the groups of each namespaced
// scope.
func newSiblingIndexes() map[resource.Type]siblingIndex {
	indexes := make(map[resource.Type]siblingIndex)
	for _, r := range memberRules {
		if r.namespaced {
			indexes[r.scope] = newSiblingIndex(r.scope)
		}
	}
	return indexes
}

// newSiblingIndex returns the index that keeps the names of the groups of the
// namespaced scope unique among siblings, compared as stored: the rule of the scope
// (ServiceConfig.NameRules) puts each name in its stored form. A group of that scope
// sits at the top or in a group, since only a pinned scope takes another parent. The
// index exempts a group whose parent is not a group, as it has no place among siblings.
func newSiblingIndex(scope resource.Type) siblingIndex {
	return gorp.NewUniqueIndex(
		"sibling_"+string(scope),
		siblingSpace(scope),
		func(g *Group) (Sibling, bool) {
			if g.Scope != scope {
				return Sibling{}, false
			}
			s := Sibling{Name: g.Name}
			if g.Parent == nil {
				return s, true
			}
			parent, err := KeyFromResourceID(*g.Parent)
			if err != nil {
				return Sibling{}, false
			}
			s.Parent = parent
			return s, true
		},
	)
}
