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
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
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

// newGroupDropReport returns a migration that writes a warning status for each member
// that the group migrations removed from a group. It must run after the member
// migrations and before ontologyDropMigration and the built-in group deletion.
func newGroupDropReport(
	res *resource.Service,
	groups *group.Service,
	statuses *status.Service,
) gorp.Migration {
	return migrate.New(
		"v61_report_dropped_group_members",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			return reportGroupDrops(ctx, tx, res, groups, statuses)
		},
	)
}

func reportGroupDrops(
	ctx context.Context,
	tx gorp.Tx,
	res *resource.Service,
	groups *group.Service,
	statuses *status.Service,
) error {
	staged, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.Type == groupv1.RelationshipTypeDropped ||
				r.Type == groupv1.RelationshipTypeKept
		},
	)
	if err != nil || len(staged) == 0 {
		return err
	}
	var (
		dropped []ontology.Relationship
		kept    = make(map[resource.ID]resource.ID)
		ids     []resource.ID
		keys    []group.Key
	)
	for _, r := range staged {
		if r.Type == groupv1.RelationshipTypeKept {
			kept[r.To] = r.From
		} else {
			dropped = append(dropped, r)
		}
		if res.Provides(r.To.Type) {
			ids = append(ids, r.To)
		}
		key, err := uuid.Parse(r.From.Key)
		if err != nil {
			return err
		}
		keys = append(keys, key)
	}
	found, err := res.Retrieve(ctx, tx, ids...)
	if err != nil {
		return err
	}
	names := make(map[resource.ID]string, len(found))
	for _, r := range found {
		names[r.ID] = r.Name
	}
	var gs []group.Group
	if err = groups.NewRetrieve().
		Where(group.MatchKeys(keys...)).
		Entries(&gs).
		Exec(ctx, tx); err != nil {
		return err
	}
	byID := make(map[resource.ID]group.Group, len(gs))
	for _, g := range gs {
		byID[g.ResourceID()] = g
	}
	w := statuses.NewWriter(tx)
	for _, r := range dropped {
		name, ok := names[r.To]
		if !ok {
			continue
		}
		g, ok := byID[r.From]
		k, isKept := kept[r.To]
		noun := strings.ReplaceAll(string(r.To.Type), "_", " ")
		var description string
		switch {
		case !ok:
			description = fmt.Sprintf(
				"The upgrade deleted a group that held %[1]s %[2]q, so the %[1]s is "+
					"no longer in it.",
				noun, name,
			)
		case g.Scope == resource.TypeBuiltin:
			description = fmt.Sprintf(
				"The upgrade deleted group %[3]q, so %[1]s %[2]q is no longer in it.",
				noun, name, g.Name,
			)
		case isKept:
			description = fmt.Sprintf(
				"A %[1]s can now be in only one group, so the upgrade kept %[1]s %[2]q "+
					"in group %[3]q, the first of its groups by name, and removed it "+
					"from group %[4]q. Drag the %[1]s into %[4]q to move it back.",
				noun,
				name,
				byID[k].Name,
				g.Name,
			)
		default:
			description = fmt.Sprintf(
				"The upgrade removed %[1]s %[2]q from group %[3]q, because that "+
					"group cannot hold a %[1]s.",
				noun, name, g.Name,
			)
		}
		s := status.Status[any]{
			Name:        g.Name,
			Variant:     status.VariantWarning,
			Message:     fmt.Sprintf("Removed %s from group", noun),
			Description: description,
			Time:        telem.Now(),
		}
		if err = w.Set(ctx, &s); err != nil {
			return err
		}
	}
	return nil
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
