// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import "github.com/synnaxlabs/x/gorp"

// RelationshipTypeParentOf indicates that a resource is the parent of another resource.
// When examining a Relationship of type RelationshipTypeParentOf, the From field will
// be the parent and the To field will be the child i.e. (From is the parent of To).
const RelationshipTypeParentOf RelationshipType = "parent"

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

var _ gorp.Entry[string] = Relationship{}

// RelationshipKeySep separates the From, Type, and To fields in an encoded relationship
// gorp key. The four Writer delete helpers depend on this layout to short-circuit scans
// without decoding the entry.
const RelationshipKeySep = "->"

// GorpKey implements the gorp.Entry interface.
func (r Relationship) GorpKey() string {
	return r.From.String() +
		RelationshipKeySep +
		string(r.Type) +
		RelationshipKeySep +
		r.To.String()
}

// SetOptions implements the gorp.Entry interface.
func (Relationship) SetOptions() []any { return nil }
