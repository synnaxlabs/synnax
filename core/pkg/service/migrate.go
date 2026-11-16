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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
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

// removalDescriptions writes the description of a removal status for each reason a
// migration removes a member from a group, given the member's type, the member's name,
// and the group's name.
var removalDescriptions = map[ontology.RelationshipType]func(
	noun, member, group string,
) string{
	groupv1.RemovedDuplicate: func(noun, member, group string) string {
		return fmt.Sprintf(
			"A %[1]s can now be in only one group, so the upgrade removed %[1]s %[2]q "+
				"from group %[3]q. It kept the %[1]s in the group that comes first by "+
				"name. Drag the %[1]s into %[3]q to move it back.",
			noun, member, group,
		)
	},
	groupv1.RemovedUnheld: func(noun, member, group string) string {
		return fmt.Sprintf(
			"The upgrade removed %[1]s %[2]q from group %[3]q, because that group "+
				"cannot hold a %[1]s.",
			noun, member, group,
		)
	},
	groupv1.RemovedPinned: func(noun, member, group string) string {
		return fmt.Sprintf(
			"The upgrade removed %[1]s %[2]q from group %[3]q, because a %[1]s cannot "+
				"be in a group inside a project or rack.",
			noun, member, group,
		)
	},
	groupv1.RemovedWithGroup: func(noun, member, group string) string {
		return fmt.Sprintf(
			"The upgrade deleted group %[3]q, so %[1]s %[2]q is no longer in it.",
			noun, member, group,
		)
	},
	groupv1.RemovedRackless: func(noun, member, group string) string {
		return fmt.Sprintf(
			"The upgrade removed %[1]s %[2]q from group %[3]q, because the rack of the "+
				"%[1]s does not exist.",
			noun,
			member,
			group,
		)
	},
}

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
					continue
				}
				describe, ok := removalDescriptions[r.Reason]
				if !ok {
					return errors.Newf("no description for group removal %q", r.Reason)
				}
				noun := strings.ReplaceAll(string(r.Member.Type), "_", " ")
				s := status.Status[any]{
					Name:        r.GroupName,
					Variant:     status.VariantWarning,
					Message:     fmt.Sprintf("Removed %s from group", noun),
					Description: describe(noun, name, r.GroupName),
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

// ontologyChannelNames are the internal channels the Core announced ontology changes
// on.
var ontologyChannelNames = []string{
	"sy_ontology_resource_set",
	"sy_ontology_resource_delete",
	"sy_ontology_relationship_set",
	"sy_ontology_relationship_delete",
}

// deleteOntologyChannels deletes the ontology channels an upgraded store still holds.
// It does nothing when none exist.
func deleteOntologyChannels(
	ctx context.Context,
	db *gorp.DB,
	ch *channel.Service,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		return ch.NewWriter(tx).
			DeleteManyByNames(ctx, ontologyChannelNames, true)
	})
}
