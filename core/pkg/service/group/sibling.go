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
// top when Parent is the zero key. A scope that is a namespace keeps each Sibling
// unique
// among its records and its groups.
type Sibling struct {
	Parent Key
	Name   string
}

// SiblingSpace returns the name of the space of the indexes that keep the siblings of
// scope unique.
func SiblingSpace(scope resource.Type) string { return "namespace." + string(scope) }

// SiblingError returns err as a validation error that names the holder of the name
// when err is a clash of sibling names, and returns err unchanged otherwise.
func SiblingError(err error) error {
	var clash gorp.UniqueError
	if !errors.As(err, &clash) {
		return err
	}
	s, ok := clash.Value.(Sibling)
	if !ok {
		return err
	}
	return validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"%s %s already holds the name '%s' in this group",
		clash.Holder.Table,
		clash.Holder.Key,
		s.Name,
	), "name")
}

// newSiblingIndex returns the index that keeps the names of the groups of scope unique
// among siblings. A group of a namespace scope sits in a group of that scope or at the
// top, so a parent of another type holds no sibling.
func newSiblingIndex(scope resource.Type) *gorp.UniqueIndex[Key, Group, Sibling] {
	return gorp.NewUniqueIndex(
		"sibling_"+string(scope),
		SiblingSpace(scope),
		func(g *Group) (Sibling, bool) {
			if g.Scope != scope {
				return Sibling{}, false
			}
			s := Sibling{Name: g.Name}
			if g.Parent == nil {
				return s, true
			}
			if g.Parent.Type != resource.TypeGroup {
				return Sibling{}, false
			}
			parent, err := uuid.Parse(g.Parent.Key)
			if err != nil {
				return Sibling{}, false
			}
			s.Parent = parent
			return s, true
		},
	)
}

// isNamespace reports whether the names in the groups of scope are unique.
func isNamespace(scope resource.Type) bool {
	for _, r := range memberRules {
		if r.scope == scope && r.namespace {
			return true
		}
	}
	return false
}

// checkName puts the name of g in NFC and checks it against the name rule
// (validate.IsPathSegment) when the scope of g is a namespace.
func checkName(g *Group) error {
	if !isNamespace(g.Scope) {
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
