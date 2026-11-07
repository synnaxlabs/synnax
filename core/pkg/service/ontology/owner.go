// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Ownership reserves parent_of relationships for one service. See [Ontology.Own].
type Ownership struct {
	// Children are the types whose parent relationships only the owner changes.
	Children []resource.Type
	// Parents are the types whose child relationships only the owner changes.
	Parents []resource.Type
}

// Owner opens the only writers that change the relationships of an [Ownership].
type Owner struct{ otg *Ontology }

// NewWriter opens a Writer on tx that may also change the owner's relationships.
func (o *Owner) NewWriter(tx gorp.Tx) Writer {
	w := o.otg.NewWriter(tx)
	w.owner = o
	return w
}

// ownedTypes maps each owned type to its owner.
type ownedTypes struct{ children, parents map[resource.Type]*Owner }

func newOwnedTypes() ownedTypes {
	return ownedTypes{
		children: make(map[resource.Type]*Owner),
		parents:  make(map[resource.Type]*Owner),
	}
}

// add returns false, and adds nothing, when a type of o is already owned.
func (ot ownedTypes) add(o Ownership, owner *Owner) bool {
	for _, t := range o.Children {
		if _, ok := ot.children[t]; ok {
			zap.S().DPanic("children already owned", zap.Stringer("type", t))
			return false
		}
	}
	for _, t := range o.Parents {
		if _, ok := ot.parents[t]; ok {
			zap.S().DPanic("parents already owned", zap.Stringer("type", t))
			return false
		}
	}
	for _, t := range o.Children {
		ot.children[t] = owner
	}
	for _, t := range o.Parents {
		ot.parents[t] = owner
	}
	return true
}

// check returns a validation error when an owner other than by owns rel.
func (ot ownedTypes) check(rel Relationship, by *Owner) error {
	if rel.Type != RelationshipTypeParentOf {
		return nil
	}
	owner, ok := ot.children[rel.To.Type]
	if !ok {
		owner, ok = ot.parents[rel.From.Type]
	}
	if !ok || owner == by {
		return nil
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"another service owns the relationship from %s to %s",
		rel.From,
		rel.To,
	)
}
