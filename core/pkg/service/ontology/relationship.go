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

// RelationshipTypeParentOf indicates that a resource is the parent of another resource.
// When examining a Relationship of type RelationshipTypeParentOf, the From field will
// be the parent and the To field will be the child i.e. (From is the parent of To).
const RelationshipTypeParentOf = versions.RelationshipTypeParentOf

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
