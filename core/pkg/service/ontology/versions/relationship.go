// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import v0 "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"

// RelationshipType uniquely identifies the type of a relationship between two
// resources.
type RelationshipType = v0.RelationshipType

// RelationshipTypeParentOf indicates that a resource is the parent of another resource.
const RelationshipTypeParentOf = v0.RelationshipTypeParentOf

// Relationship represents a relationship between two resources in the ontology.
type Relationship = v0.Relationship

// RelationshipKeySep separates the From, Type, and To fields in an encoded
// relationship gorp key.
const RelationshipKeySep = v0.RelationshipKeySep
