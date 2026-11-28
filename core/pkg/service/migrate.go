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
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/ranges"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
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
// (channelv1.TextRename), read with cfg as a v0.62 Core read it. It runs before
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
			byKey, err := channelv1.TextChannels(ctx, tx, channels)
			if err != nil {
				return err
			}
			renames, err := channelv1.TextRename(tx, byKey)
			if err != nil || renames == nil {
				return err
			}
			w := channelv1.Rewriter{
				Renames: renames,
				Config:  cfg,
				Ambient: slices.Concat(
					stl.NewSymbols(),
					arcstatus.NewSymbols(),
					ranges.NewSymbols(),
				),
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
				renamed, err := renameSource(ctx, tx, w, a.ResourceID(), &b.Source)
				if err != nil {
					return err
				}
				if renamed {
					a.SetBase(b)
					changed = append(changed, a)
				}
			}
			return gorp.WrapWriter[arc.Key, arc.Arc](tx).Set(ctx, changed...)
		},
	)
}

// renameSource rewrites each stored text of src with w, stages each reference that the
// rewrite kept on an old name as a text site of holder, and reports whether it changed
// src.
func renameSource(
	ctx context.Context,
	tx gorp.Tx,
	w channelv1.Rewriter,
	holder resource.ID,
	src *arc.Source,
) (bool, error) {
	rewrite := func(source channelv1.Source) (string, error) {
		res, err := w.Rewrite(ctx, source)
		if err != nil {
			return "", err
		}
		return res.Text, channelv1.StageKept(ctx, tx, holder, source.Function, res)
	}
	changed := false
	switch v := src.Variant.(type) {
	case arc.TextSource:
		raw := v.Text.Materialize().Raw
		out, err := rewrite(channelv1.Source{Text: raw, Form: channelv1.FormProgram})
		if err != nil {
			return false, err
		}
		if out != raw {
			v.Text = text.Text{Doc: text.Create(out)}
			src.Variant, changed = v, true
		}
	case arc.GraphSource:
		keys := make([]string, len(v.Graph.Functions))
		for i, f := range v.Graph.Functions {
			keys[i] = f.Key
		}
		for i, f := range v.Graph.Functions {
			out, err := rewrite(channelv1.Source{
				Text:      f.Body.Raw,
				Form:      channelv1.FormBody,
				Function:  f.Key,
				Inputs:    f.Inputs,
				Outputs:   f.Outputs,
				Functions: keys,
			})
			if err != nil {
				return false, err
			}
			if out != f.Body.Raw {
				v.Graph.Functions[i].Body.Raw = out
				src.Variant, changed = v, true
			}
		}
	}
	return changed, nil
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
			if err = reportTextSites(ctx, tx, res, w); err != nil {
				return err
			}
			if err = resourcev0.ClearTextSites(ctx, tx); err != nil {
				return err
			}
			return resourcev0.ClearRenames(ctx, tx)
		},
	)
}

// reportTextSites writes a warning status for each reference in stored text that the
// text passes left on the old name of a renamed channel (resourcev0.StagedTextSites).
func reportTextSites(
	ctx context.Context,
	tx gorp.Tx,
	res *resource.Service,
	w status.Writer,
) error {
	sites, err := resourcev0.StagedTextSites(tx)
	if err != nil || len(sites) == 0 {
		return err
	}
	ids := make([]resource.ID, len(sites))
	for i, site := range sites {
		ids[i] = site.Holder
	}
	found, err := res.Retrieve(ctx, tx, ids...)
	if err != nil {
		return err
	}
	holders := make(map[resource.ID]string, len(found))
	for _, r := range found {
		holders[r.ID] = r.Name
	}
	for _, site := range sites {
		holder, ok := holders[site.Holder]
		if !ok {
			return errors.Newf("staged text site of %s names no record", site.Holder)
		}
		s := textSiteStatus(site, holder)
		if err = w.Set(ctx, &s); err != nil {
			return err
		}
	}
	return nil
}

// textSiteStatus returns the warning status that reports site in the text of the
// resource named holder.
func textSiteStatus(site resourcev0.TextSite, holder string) status.Status[any] {
	noun := "calculated channel"
	if site.Holder.Type == resource.TypeArc {
		noun = "Arc"
	}
	if site.Function != "" {
		noun = fmt.Sprintf("function %q of %s", site.Function, noun)
	}
	why, fix := "the text does not parse, or its names do not bind",
		"Edit the text so that it reads the channel."
	switch {
	case site.Captured:
		why = "a symbol that the text declares holds one of these names there"
	case site.Folder != "":
		why = fmt.Sprintf(
			"the folder %q on its path cannot be read in Arc yet",
			site.Folder,
		)
		fix = "Rename that folder so that Arc can read it, then edit the text so " +
			"that it reads the channel."
	}
	return status.Status[any]{
		Name:    holder,
		Variant: status.VariantWarning,
		Message: "Kept an old channel name in text",
		Description: fmt.Sprintf(
			"The upgrade renamed channel %q to %q, but the text of %s %q still reads "+
				"the old name at line %d, column %d, because %s. %s",
			site.Name,
			site.NewName,
			noun,
			holder,
			site.Line,
			site.Column+1,
			why,
			fix,
		),
		Time: telem.Now(),
	}
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
			channelv1.MetricsFolder,
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
