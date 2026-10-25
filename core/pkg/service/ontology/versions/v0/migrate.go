// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/gorp/migrate"

// ResourceMigration re-encodes stored resources from MessagePack to Orc.
var ResourceMigration = migrate.Codec[string, Resource]("msgpack_to_orc")

// RelationshipMigration re-encodes stored relationships from MessagePack to Orc.
var RelationshipMigration = migrate.Codec[string, Relationship]("msgpack_to_orc")

// ResourceNormalizeKeys re-keys Resource rows stored under the pre-v0.54 key format.
var ResourceNormalizeKeys = migrate.NormalizeKeys[string, Resource]("Resource")

// RelationshipNormalizeKeys re-keys Relationship rows stored under the pre-v0.54 key
// format.
var RelationshipNormalizeKeys = migrate.NormalizeKeys[
	string,
	Relationship,
]("Relationship")
