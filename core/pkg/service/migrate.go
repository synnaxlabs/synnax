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
	group "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
)

// ontologyDropMigration deletes every stored ontology resource and relationship, and
// the built-in root groups that the group migration gave the builtin scope. It must run
// after every migration that reads the ontology.
var ontologyDropMigration = migrate.New(
	"v61_drop_ontology",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) (err error) {
		if err = ontology.Drop(ctx, tx); err != nil {
			return err
		}
		iter, err := gorp.WrapReader[group.Key, group.Group](tx).
			OpenIterator(gorp.IterOptions{})
		if err != nil {
			return err
		}
		var builtin []group.Key
		for iter.First(); iter.Valid(); iter.Next() {
			if g := iter.Value(ctx); g != nil && g.Scope == resource.TypeBuiltin {
				builtin = append(builtin, g.Key)
			}
		}
		if err = errors.Combine(iter.Error(), iter.Close()); err != nil {
			return err
		}
		return gorp.WrapWriter[group.Key, group.Group](tx).Delete(ctx, builtin...)
	},
)

// ontologyChannelNames are the internal channels the Core announced ontology changes
// on.
var ontologyChannelNames = []string{
	"sy_ontology_resource_set",
	"sy_ontology_resource_delete",
	"sy_ontology_relationship_set",
	"sy_ontology_relationship_delete",
}

// deleteOntologyChannels deletes the ontology channels an upgraded store still holds.
// It
// does nothing when none exist.
func deleteOntologyChannels(
	ctx context.Context,
	db *gorp.DB,
	ch *channel.Service,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		return ch.NewWriter(tx).DeleteManyByNames(ctx, ontologyChannelNames, true)
	})
}
