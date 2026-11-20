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
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
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

// newGroupRemovalReport returns a migration that writes a warning status for each
// member that the group migrations removed from a group. It must run before
// ontologyDropMigration.
func newGroupRemovalReport(
	res *resource.Service,
	statuses *status.Service,
) gorp.Migration {
	return migrate.New(
		"v61_report_group_removals",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			removals, err := groupv1.RetrieveRemovals(ctx, tx)
			if err != nil || len(removals) == 0 {
				return err
			}
			var ids []resource.ID
			for _, r := range removals {
				if res.Provides(r.Member.Type) {
					ids = append(ids, r.Member)
				}
			}
			found, err := res.Retrieve(ctx, tx, ids...)
			if err != nil {
				return err
			}
			names := make(map[resource.ID]string, len(found))
			for _, r := range found {
				names[r.ID] = r.Name
			}
			w := statuses.NewWriter(tx)
			for _, r := range removals {
				name, ok := names[r.Member]
				if !ok {
					name = r.Member.Key
				}
				noun := strings.ReplaceAll(string(r.Member.Type), "_", " ")
				s := status.Status[any]{
					Name:        r.GroupName,
					Variant:     status.VariantWarning,
					Message:     fmt.Sprintf("Removed %s from group", noun),
					Description: r.Describe(noun, name),
					Time:        telem.Now(),
				}
				if err = w.Set(ctx, &s); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// rangeAliasDropMigration deletes the range aliases that an upgraded store holds.
var rangeAliasDropMigration = migrate.Drop[string]("v62_drop_range_aliases", "Alias")

// taskNames returns the task names that the migrations of PagerDuty and Kafka read.
func taskNames(
	svc *task.Service,
) func(context.Context, gorp.Tx, []task.Key) (map[task.Key]string, error) {
	return func(
		ctx context.Context,
		tx gorp.Tx,
		keys []task.Key,
	) (map[task.Key]string, error) {
		var tasks []task.Task
		if err := svc.NewRetrieve().
			Where(task.MatchKeys(keys...)).
			Entries(&tasks).
			Exec(ctx, tx); err != nil {
			return nil, err
		}
		return lo.SliceToMap(tasks, func(t task.Task) (task.Key, string) {
			return t.Key, t.Name
		}), nil
	}
}

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
