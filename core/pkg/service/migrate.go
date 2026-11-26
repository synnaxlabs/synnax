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
	"unicode/utf8"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	resourcev0 "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
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

// newRenameReport returns a migration that writes a warning status for each rename that
// the name passes staged, renames the storage of each renamed channel in ts, and then
// deletes the staged renames.
func newRenameReport(
	res *resource.Service,
	statuses *status.Service,
	ts *cesium.DB,
) gorp.Migration {
	return migrate.New(
		"v63_report_renames",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			renames, err := resourcev0.StagedRenames(tx)
			if err != nil || len(renames) == 0 {
				return err
			}
			ids := make([]resource.ID, len(renames))
			for i, r := range renames {
				ids[i] = r.ID
			}
			found, err := res.Retrieve(ctx, tx, ids...)
			if err != nil {
				return err
			}
			names := make(map[resource.ID]string, len(found))
			for _, r := range found {
				names[r.ID] = r.Name
			}
			var (
				w      = statuses.NewWriter(tx)
				stored = make(map[cesium.ChannelKey]string)
			)
			for _, r := range renames {
				name, ok := names[r.ID]
				if !ok {
					return errors.Newf("staged rename of %s names no record", r.ID)
				}
				s := renameStatus(r, name)
				if err = w.Set(ctx, &s); err != nil {
					return err
				}
				if r.ID.Type != resource.TypeChannel {
					continue
				}
				key, err := channel.ParseKey(r.ID.Key)
				if err != nil {
					return err
				}
				// A free channel has no storage.
				if !key.Free() {
					stored[cesium.ChannelKey(key)] = name
				}
			}
			if err = ts.RenameChannels(ctx, stored); err != nil {
				return err
			}
			return resourcev0.ClearRenames(ctx, tx)
		},
	)
}

// renameStatus returns the warning status that reports the rename r of a resource that
// now holds name.
func renameStatus(r resourcev0.Rename, name string) status.Status[any] {
	noun := "channel"
	if r.ID.Type == resource.TypeGroup {
		noun = "folder"
	}
	return status.Status[any]{
		Name:    name,
		Variant: status.VariantWarning,
		Message: "Renamed " + noun,
		Description: fmt.Sprintf(
			"The upgrade renamed %s %q to %q, because %s. Update each reference to "+
				"the old name outside Synnax.",
			noun,
			r.From,
			name,
			renameReason(r.From, noun),
		),
		Time: telem.Now(),
	}
}

// renameReason returns why the name passes renamed a noun that held the name from.
func renameReason(from, noun string) string {
	switch {
	case !utf8.ValidString(from):
		return "the old name held bytes that are not valid text"
	case strings.Contains(from, "."):
		return "a '.' now separates the folders of a path, so a name cannot hold one"
	case from == "":
		return "the old name was empty"
	case noun == "folder":
		return "another folder beside it held the same name"
	}
	return "another channel in the same folder held the same name"
}

// rangeAliasDropMigration deletes the range aliases that an upgraded store holds.
var rangeAliasDropMigration = migrate.Drop[string]("v62_drop_range_aliases", "Alias")

// taskNames returns the task names that the migrations of PagerDuty, Kafka, HTTP, and
// MQTT read.
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
