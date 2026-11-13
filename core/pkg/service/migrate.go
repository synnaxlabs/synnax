// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"go.uber.org/zap"
)

// ontologyDropMigration deletes every stored ontology resource and relationship, and
// logs each label edge it deletes. It must run after every migration that reads the
// ontology.
var ontologyDropMigration = migrate.New(
	"v61_drop_ontology",
	func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
		labels, err := ontology.RetrieveRelationships(
			ctx,
			tx,
			func(rel ontology.Relationship) bool {
				return rel.Type == ontology.RelationshipTypeLabeledBy
			},
		)
		if err != nil {
			return err
		}
		for _, rel := range labels {
			ins.L.Warn(
				"dropped a label from a resource that cannot hold labels",
				zap.Stringer("label", rel.To),
				zap.Stringer("resource", rel.From),
			)
		}
		return ontology.Drop(ctx, tx)
	},
)

// rangeAliasDropMigration deletes the range aliases that an upgraded store holds.
var rangeAliasDropMigration = migrate.Drop("v62_drop_range_aliases", "Alias")

// retiredChannelNames are the internal channels the Core announced ontology and range
// alias changes on.
var retiredChannelNames = []string{
	"sy_ontology_resource_set",
	"sy_ontology_resource_delete",
	"sy_ontology_relationship_set",
	"sy_ontology_relationship_delete",
	"sy_range_alias_set",
	"sy_range_alias_delete",
	"sy_range_alias_action_set",
}

// deleteRetiredChannels deletes the retired channels an upgraded store still holds. It
// does nothing when none exist.
func deleteRetiredChannels(
	ctx context.Context,
	db *gorp.DB,
	ch *channel.Service,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		return ch.NewWriter(tx).
			DeleteManyByNames(ctx, retiredChannelNames, true)
	})
}
