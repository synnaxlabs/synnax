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

	"github.com/synnaxlabs/synnax/pkg/service/ontology/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// RelationshipType is a string that uniquely identifies the type of a relationship
// between two resources. For example, a relationship of type "member" could indicate
// that a particular resource is a member of another resource. When defining
// relationship types, use the synnax [Relationship.From] is the [Relationship.Type] of
// [Relationship.To] pattern. For example, if a relationship of type "member" indicates
// that a particular the variable should be named MemberOf (i.e. Start is a MemberOf
// To).
type RelationshipType = versions.RelationshipType

// RelationshipTypeParentOf indicates that a resource is the parent of another resource.
// When examining a Relationship of type RelationshipTypeParentOf, the From field will
// be the parent and the To field will be the child i.e. (From is the parent of To).
const RelationshipTypeParentOf = versions.RelationshipTypeParentOf

// Relationship is a struct that represents a relationship between two resources in the
// ontology. A relationship is defined by a type, a from and a to field. This means that
// two resources can have multiple relationships of different types between them. Think
// about the relationship like From->Type->To i.e. Dog->Parent->Puppy.
type Relationship = versions.Relationship

// relationshipKeySep separates the From, Type, and To fields in an encoded relationship
// gorp key. The four Writer delete helpers depend on this layout to short-circuit scans
// without decoding the entry.
const relationshipKeySep = versions.RelationshipKeySep

func ParseRelationship(key string) (Relationship, error) {
	split := strings.Split(key, "->")
	if len(split) != 3 {
		return Relationship{},
			errors.Wrapf(validate.ErrValidation, "invalid relationship key: %s", key)
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
