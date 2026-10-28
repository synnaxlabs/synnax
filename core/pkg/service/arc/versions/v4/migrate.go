// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
)

// MigrateArc lifts a v3 Arc to v4. Every Arc stored before v4 is an automation.
func MigrateArc(ctx context.Context, old v3.Arc) (Arc, error) {
	a, err := autoMigrateArc(ctx, old)
	if err != nil {
		return Arc{}, err
	}
	a.ApplyDefaults()
	return a, nil
}

// Migration lifts stored Arcs from v3 to v4.
var Migration = migrate.NewEntry("v61_add_type", MigrateArc)

// MigrationConfig is the configuration for NewGroupMigration.
type MigrationConfig struct {
	// Ontology holds the parent relationships of stored Arcs.
	Ontology *ontology.Ontology
	// Group is the Arcs group.
	Group group.Group
}

// NewGroupMigration returns the migration that parents every stored Arc without a
// group under the Arcs group.
func NewGroupMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"v61_arcs_group",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) (err error) {
			iter, err := gorp.WrapReader[Key, Arc](tx).OpenIterator(gorp.IterOptions{})
			if err != nil {
				return err
			}
			defer func() { err = errors.Combine(err, iter.Close()) }()
			var orphans []ontology.ID
			for iter.First(); iter.Valid(); iter.Next() {
				id := iter.Value(ctx).OntologyID()
				if err = iter.Error(); err != nil {
					return err
				}
				var parents []ontology.Resource
				// The Arc ontology service is not registered yet, so field data must
				// stay excluded.
				if err = cfg.Ontology.NewRetrieve().
					WhereIDs(id).
					TraverseTo(ontology.ParentsTraverser).
					WhereTypes(ontology.ResourceTypeGroup).
					ExcludeFieldData(true).
					Entries(&parents).
					Exec(ctx, tx); err != nil {
					return err
				}
				if len(parents) == 0 {
					orphans = append(orphans, id)
				}
			}
			if len(orphans) == 0 {
				return nil
			}
			return cfg.Ontology.NewWriter(tx).DefineRelationships(
				ctx,
				cfg.Group.OntologyID(),
				ontology.RelationshipTypeParentOf,
				orphans...,
			)
		},
	)
}
