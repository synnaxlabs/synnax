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
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// RelationshipType is a string that uniquely identifies the type of a relationship
// between two resources. For example, a relationship of type "member" could indicate
// that a particular resource is a member of another resource. When defining
// relationship types, use the synnax [Relationship.From] is the [Relationship.Type] of
// [Relationship.To] pattern. For example, if a relationship of type "member" indicates
// that a particular the variable should be named MemberOf (i.e. Start is a MemberOf
// To).
type RelationshipType string

const (
	// ParentOf indicates that a resource is the parent of another resource. When
	// examining a Relationship of type ParentOf, the Start field will be the parent and
	// the to field will be the child i.e. (Start is the ParentOf To).
	ParentOf RelationshipType = "parent"
)

// Relationship is a struct that represents a relationship between two resources in the
// ontology. A relationship is defined by a type, a from and a to field. This means that
// two resources can have multiple relationships of different types between them. Think
// about the relationship like From->Type->To i.e. Dog->Parent->Puppy.
type Relationship struct {
	// From is the ID of the resource that the relationship starts from.
	From ID `json:"from" msgpack:"from"`
	// To is the ID of the resource that the relationship ends at.
	To ID `json:"to" msgpack:"to"`
	// Type is the type of relationship between the two resources. For more information
	// on relationship types, see the [RelationshipType] documentation.
	Type RelationshipType `json:"type" msgpack:"type"`
}

var _ gorp.Entry[[]byte] = Relationship{}

// GorpKey implements the gorp.Entry interface.
func (r Relationship) GorpKey() []byte {
	return []byte(r.From.String() + "->" + string(r.Type) + "->" + r.To.String())
}

// SetOptions implements the gorp.Entry interface.
func (r Relationship) SetOptions() []any { return nil }

func ParseRelationship(key []byte) (Relationship, error) {
	split := strings.Split(string(key), "->")
	if len(split) != 3 {
		return Relationship{}, errors.Wrapf(validate.Error, "invalid relationship key: %s", key)
	}
	var (
		r   Relationship
		err error
	)
	if r.From, err = ParseID(split[0]); err != nil {
		return Relationship{}, err
	}
	r.Type = RelationshipType(split[1])
	if r.To, err = ParseID(split[2]); err != nil {
		return Relationship{}, err
	}
	return r, nil
}
