// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/x/types"
)

// resourceMigration re-encodes stored resources from MessagePack to Orc.
var resourceMigration = migrate.Codec[string, Resource]("msgpack_to_orc")

// relationshipMigration re-encodes stored relationships from MessagePack to Orc.
var relationshipMigration = migrate.Codec[string, Relationship]("msgpack_to_orc")

// ResourceNormalizeKeys re-keys Resource rows stored under the pre-v0.54 key format.
var ResourceNormalizeKeys = migrate.NormalizeKeys[string, Resource]("Resource")

// RelationshipNormalizeKeys re-keys Relationship rows stored under the pre-v0.54 key
// format.
var RelationshipNormalizeKeys = migrate.NormalizeKeys[
	string,
	Relationship,
]("Relationship")

// ResourceMigrations is the ordered migration chain for stored resources.
var ResourceMigrations = []gorp.Migration{ResourceNormalizeKeys, resourceMigration}

// RelationshipMigrations is the ordered migration chain for stored relationships.
var RelationshipMigrations = []gorp.Migration{
	RelationshipNormalizeKeys,
	relationshipMigration,
}

// Migrate runs the migration chains of the resource and relationship tables in db, so
// that later migrations read stored rows in the v0 format. It must run before any
// migration that reads them.
func Migrate(ctx context.Context, db *gorp.DB, ins alamos.Instrumentation) error {
	if err := gorp.Migrate(ctx, gorp.MigrateConfig{
		Instrumentation: ins,
		DB:              db,
		Namespace:       types.Name[Resource](),
		Migrations:      ResourceMigrations,
	}); err != nil {
		return err
	}
	return gorp.Migrate(ctx, gorp.MigrateConfig{
		Instrumentation: ins,
		DB:              db,
		Namespace:       types.Name[Relationship](),
		Migrations:      RelationshipMigrations,
	})
}
