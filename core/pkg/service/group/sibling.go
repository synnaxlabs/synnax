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
	"slices"
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
// Sibling of a record.
func NewMemberIndex[K gorp.Key, E gorp.Entry[K]](
	scope resource.Type,
	sibling func(E) Sibling,
) *gorp.UniqueIndex[K, E, Sibling] {
	return gorp.NewUniqueIndex(
		"sibling",
		siblingSpace(scope),
		func(e E) (Sibling, bool) { return sibling(e), true },
	)
}

// SiblingError returns err as a validation error that names the kind of the holder of
// the name when err is a clash of sibling names, and returns err unchanged otherwise.
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
	return validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"a %s named '%s' already exists %s",
		clash.Holder.Table,
		s.Name,
		where,
	), "name")
}

// siblingIndexes returns the index of the names of the groups of each namespaced scope.
func siblingIndexes() []gorp.Index[Key, Group] {
	var scopes []resource.Type
	for _, r := range memberRules {
		if r.namespaced && !slices.Contains(scopes, r.scope) {
			scopes = append(scopes, r.scope)
		}
	}
	slices.Sort(scopes)
	indexes := make([]gorp.Index[Key, Group], len(scopes))
	for i, scope := range scopes {
		indexes[i] = newSiblingIndex(scope)
	}
	return indexes
}

// newSiblingIndex returns the index that keeps the names of the groups of the
// namespaced scope unique among siblings. A group of that scope sits at the top or in
// a group, since only a pinned scope takes another parent.
func newSiblingIndex(scope resource.Type) *gorp.UniqueIndex[Key, Group, Sibling] {
	return gorp.NewUniqueIndex(
		"sibling_"+string(scope),
		siblingSpace(scope),
		func(g Group) (Sibling, bool) {
			if g.Scope != scope {
				return Sibling{}, false
			}
			s := Sibling{Name: g.Name}
			if g.Parent != nil {
				s.Parent = uuid.MustParse(g.Parent.Key)
			}
			return s, true
		},
	)
}

// isNamespaced reports whether the names in the groups of scope are unique.
func isNamespaced(scope resource.Type) bool {
	for _, r := range memberRules {
		if r.scope == scope {
			return r.namespaced
		}
	}
	return false
}

// checkName puts the name of g in NFC and checks it against the name rule
// (validate.IsPathSegment) when the scope of g is namespaced.
func checkName(g *Group) error {
	if !isNamespaced(g.Scope) {
		return nil
	}
	g.Name = validate.NormalizeName(g.Name)
	if g.Name == "" || validate.IsPathSegment(g.Name) {
		return nil
	}
	return validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"group name '%s' must hold a visible character, and no '.' or invisible "+
			"character",
		g.Name,
	), "name")
}
