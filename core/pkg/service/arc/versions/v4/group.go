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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// GroupName is the name of the root group that holds every Arc.
const GroupName = "Arcs"

// MigrationConfig is the configuration for NewGroupMigration.
type MigrationConfig struct {
	// Ontology is the cluster ontology; existing Arcs are parented in it.
	Ontology *ontology.Ontology
	// Group is the group service that holds the Arcs group.
	Group *group.Service
}

// NewGroupMigration returns the migration that parents every stored Arc under the Arcs
// group, creating the group when it does not exist.
func NewGroupMigration(cfg MigrationConfig) migrate.Migration {
	return gorp.NewMigration(
		"arc_group_1",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) (err error) {
			g, err := cfg.Group.CreateOrRetrieve(ctx, GroupName, ontology.RootID)
			if err != nil {
				return err
			}
			iter, err := gorp.WrapReader[Key, Arc](tx).OpenIterator(gorp.IterOptions{})
			if err != nil {
				return err
			}
			defer func() { err = errors.Combine(err, iter.Close()) }()
			var ids []ontology.ID
			for iter.First(); iter.Valid(); iter.Next() {
				if err = iter.Error(); err != nil {
					return err
				}
				ids = append(ids, iter.Value(ctx).OntologyID())
			}
			if len(ids) == 0 {
				return nil
			}
			return cfg.Ontology.NewWriter(tx).DefineRelationships(
				ctx,
				g.OntologyID(),
				ontology.RelationshipTypeParentOf,
				ids...,
			)
		},
	)
}
