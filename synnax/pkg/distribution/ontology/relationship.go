// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"github.com/synnaxlabs/x/gorp"
)

// RelationshipType is a string that uniquely identifies the type of a relationship
// between two resources. For example, a relationship of type "member" could indicate
// that a particular resource is a member of another resource. When defining relationship
// types, use the synnax [Relationship.From] is the [Relationship.Type] of [Relationship.To]
// pattern. For example, if a relationship of type "member" indicates that a particular
// the variable should be named MemberOf (i.e. Start is a MemberOf To).
type RelationshipType string

const (
	// ParentOf indicates that a resource is the parent of another resource. When
	// examining a Relationship of type ParentOf, the Start field will be the parent
	// and the to field will be the child i.e. (Start is the ParentOf To).
	ParentOf RelationshipType = "parent"
)

// Relationship is a struct that represents a relationship between two resources in the
// ontology. A relationship is defined by a type, a from and a to field. This means that
// two resources can have multiple relationships of different types between them.
type Relationship struct {
	// From, To are the IDs of the related resources.
	From, To ID
	// Type is the type of relationship between the two resources. For more information
	// on relationship types, see the [RelationshipType] documentation.
	Type RelationshipType
}

var _ gorp.Entry[string] = Relationship{}

// GorpKey implements the gorp.Entry interface.
func (r Relationship) GorpKey() string {
	return r.From.String() + ":" + string(r.Type) + ":" + r.To.String()

}

// SetOptions implements the gorp.Entry interface.
func (r Relationship) SetOptions() []interface{} { return nil }
