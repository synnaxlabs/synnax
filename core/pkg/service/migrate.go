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
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
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

// newArcTextMigration returns the migration that writes the new name of each renamed
// channel into the text of each Arc and into the function bodies of each graph
// (channelv1.TextRename), lexed with cfg as a v0.62 Core lexed it. It runs before
// newRenameReport deletes the staged renames, and before anything compiles an Arc.
func newArcTextMigration(cfg parser.Config) gorp.Migration {
	return migrate.New(
		"v63_rename_arc_channels",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var channels []channel.Channel
			if err := gorp.NewRetrieve[channel.Key, channel.Channel]().
				Entries(&channels).
				Exec(ctx, tx); err != nil {
				return err
			}
			names := make(map[channel.Key]string, len(channels))
			for _, c := range channels {
				names[c.Key()] = c.Name
			}
			rename, err := channelv1.TextRename(tx, names)
			if err != nil || rename == nil {
				return err
			}
			var arcs []arc.Arc
			if err = gorp.NewRetrieve[arc.Key, arc.Arc]().
				Entries(&arcs).
				Exec(ctx, tx); err != nil {
				return err
			}
			var changed []arc.Arc
			for _, a := range arcs {
				b := a.Base()
				if renameSource(&b.Source, cfg, rename) {
					a.SetBase(b)
					changed = append(changed, a)
				}
			}
			return gorp.WrapWriter[arc.Key, arc.Arc](tx).Set(ctx, changed...)
		},
	)
}

// renameSource applies rename to each channel name in src, and reports whether it
// changed src.
func renameSource(
	src *arc.Source,
	cfg parser.Config,
	rename func(string) (string, bool),
) bool {
	changed := false
	switch v := src.Variant.(type) {
	case arc.TextSource:
		raw := v.Text.Materialize().Raw
		if renamed := channelv1.RenameInText(raw, cfg, rename); renamed != raw {
			v.Text = text.Text{Doc: text.Create(renamed)}
			src.Variant, changed = v, true
		}
	case arc.GraphSource:
		for i, f := range v.Graph.Functions {
			renamed := channelv1.RenameInText(f.Body.Raw, cfg, rename)
			if renamed != f.Body.Raw {
				v.Graph.Functions[i].Body.Raw = renamed
				src.Variant, changed = v, true
			}
		}
	}
	return changed
}

// newRenameReport returns a migration that writes a warning status for each rename that
// the name and clash passes staged, renames the storage of each renamed channel through
// channels, and then deletes the staged renames.
func newRenameReport(
	res *resource.Service,
	statuses *status.Service,
	channels *channel.Service,
) gorp.Migration {
	return migrate.New(
		"v63_report_renames",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			renames, err := resourcev0.StagedRenames(tx)
			if err != nil || len(renames) == 0 {
				return err
			}
			names, err := renamedNames(ctx, tx, res, renames)
			if err != nil {
				return err
			}
			var (
				w      = statuses.NewWriter(tx)
				stored = make(map[channel.Key]string)
			)
			for _, r := range renames {
				s, err := renameStatus(r, names[r.ID])
				if err != nil {
					return err
				}
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
				stored[key] = names[r.ID]
			}
			if err = channels.RenameStorage(ctx, stored); err != nil {
				return err
			}
			return resourcev0.ClearRenames(ctx, tx)
		},
	)
}

// renamedNames returns the name that each renamed channel and folder holds now, by
// resource ID.
func renamedNames(
	ctx context.Context,
	tx gorp.Tx,
	res *resource.Service,
	renames []resourcev0.Rename,
) (map[resource.ID]string, error) {
	ids := make([]resource.ID, len(renames))
	for i, r := range renames {
		if r.ID.Type != resource.TypeChannel && r.ID.Type != resource.TypeGroup {
			return nil, errors.Newf(
				"staged rename of %s is not of a channel or a folder",
				r.ID,
			)
		}
		ids[i] = r.ID
	}
	found, err := res.Retrieve(ctx, tx, ids...)
	if err != nil {
		return nil, err
	}
	names := make(map[resource.ID]string, len(found))
	for _, r := range found {
		names[r.ID] = r.Name
	}
	for _, id := range ids {
		if _, ok := names[id]; !ok {
			return nil, errors.Newf("staged rename of %s names no record", id)
		}
	}
	return names, nil
}

// renameStatus returns the warning status that reports the rename r of a resource that
// now holds name.
func renameStatus(r resourcev0.Rename, name string) (status.Status[any], error) {
	noun := "channel"
	if r.ID.Type == resource.TypeGroup {
		noun = "folder"
	}
	reason, err := renameReason(r.Reason, noun)
	if err != nil {
		return status.Status[any]{}, errors.Wrapf(err, "staged rename of %s", r.ID)
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
			reason,
		),
		Time: telem.Now(),
	}, nil
}

// renameReason returns why the upgrade renamed a noun for reason.
func renameReason(reason resourcev0.Reason, noun string) (string, error) {
	switch reason {
	case resourcev0.ReasonInvalidBytes:
		return "the old name held bytes that are not valid text", nil
	case resourcev0.ReasonSeparator:
		return "a '.' now separates the folders of a path, so a name cannot hold one", nil
	case resourcev0.ReasonEmpty:
		return "the old name was empty", nil
	case resourcev0.ReasonRepeat:
		if noun == "folder" {
			return "another folder beside it held the same name", nil
		}
		return "another channel in the same folder held the same name", nil
	case resourcev0.ReasonClash:
		if noun == "folder" {
			return "a channel beside it held the same name", nil
		}
		return "a folder beside it held the same name", nil
	case resourcev0.ReasonMetricsFolder:
		return fmt.Sprintf(
			"the Core keeps the top-level name %q for its metrics folder",
			groupv1.MetricsFolder,
		), nil
	}
	return "", errors.Newf("unknown reason %d", reason)
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
