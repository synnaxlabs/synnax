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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arcv4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	resourcev0 "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// sibling is a name in a channel folder, or at the top when parent is the zero key.
type sibling struct {
	parent group.Key
	name   string
}

// nameClashMigration renames one side of each channel and channel folder that hold the
// same name in one folder, after NFC. The channel takes the first free suffix when a
// name pass renamed it and not the folder, or when the folder is the top-level metrics
// folder (groupv1.MetricsFolder); else the folder does. It stages each rename, so it
// runs after the name passes and before the text passes.
var nameClashMigration = migrate.New(
	"v63_name_clashes",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		staged, err := resourcev0.StagedRenames(tx)
		if err != nil {
			return err
		}
		renamed := make(set.Set[resource.ID], len(staged))
		for _, r := range staged {
			renamed.Add(r.ID)
		}
		var channels []channel.Channel
		if err = gorp.NewRetrieve[channel.Key, channel.Channel]().
			Entries(&channels).
			Exec(ctx, tx); err != nil {
			return err
		}
		folders, err := channelFolders(ctx, tx)
		if err != nil {
			return err
		}
		var (
			names    = make(set.Set[string], len(channels))
			siblings = make(set.Set[sibling], len(channels)+len(folders))
			holders  = make(map[sibling]int, len(channels))
		)
		for i, c := range channels {
			at := sibling{name: resourcev0.NormalizeName(c.Name)}
			if c.Group != nil {
				at.parent = *c.Group
			}
			names.Add(c.Name)
			siblings.Add(at)
			holders[at] = i
		}
		at := make([]sibling, len(folders))
		for i, f := range folders {
			at[i].name = resourcev0.NormalizeName(f.Name)
			if f.Parent != nil {
				if at[i].parent, err = group.KeyFromResourceID(*f.Parent); err != nil {
					return err
				}
			}
			siblings.Add(at[i])
		}
		for i, f := range folders {
			parent := at[i].parent
			h, ok := holders[at[i]]
			if !ok {
				continue
			}
			c := channels[h]
			builtIn := parent == (group.Key{}) && f.Name == groupv1.MetricsFolder
			if builtIn ||
				renamed.Contains(c.ResourceID()) && !renamed.Contains(f.ResourceID()) {
				name := resourcev0.FreeName(c.Name, func(name string) bool {
					return names.Contains(name) ||
						siblings.Contains(sibling{parent: parent, name: name})
				})
				names.Add(name)
				siblings.Add(sibling{parent: parent, name: name})
				if err = resourcev0.StageRename(
					ctx,
					tx,
					c.ResourceID(),
					c.Name,
				); err != nil {
					return err
				}
				c.Name = name
				if err = gorp.WrapWriter[channel.Key, channel.Channel](tx).
					Set(ctx, c); err != nil {
					return err
				}
				continue
			}
			name := resourcev0.FreeName(f.Name, func(name string) bool {
				return siblings.Contains(sibling{parent: parent, name: name})
			})
			siblings.Add(sibling{parent: parent, name: name})
			if err = resourcev0.StageRename(
				ctx,
				tx,
				f.ResourceID(),
				f.Name,
			); err != nil {
				return err
			}
			f.Name = name
			if err = gorp.WrapWriter[group.Key, group.Group](
				tx,
			).Set(ctx, f); err != nil {
				return err
			}
		}
		return nil
	},
)

// channelFolders returns every channel folder, by key.
func channelFolders(ctx context.Context, tx gorp.Tx) ([]group.Group, error) {
	var folders []group.Group
	if err := gorp.NewRetrieve[group.Key, group.Group]().
		Where(gorp.Match(func(_ gorp.Context, g *group.Group) (bool, error) {
			return g.Scope == resource.TypeChannel, nil
		})).
		Entries(&folders).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	slices.SortFunc(folders, func(a, b group.Group) int { return a.Key.Compare(b.Key) })
	return folders, nil
}

// textRename returns the rename that the text passes apply to each channel name in
// stored text. A dashed name that a channel holds becomes quoted. A name that a renamed
// channel held, and that no channel holds now, becomes the new name of that channel,
// unless several renamed channels held it.
func textRename(ctx context.Context, tx gorp.Tx) (func(string) (string, bool), error) {
	var channels []channel.Channel
	if err := gorp.NewRetrieve[channel.Key, channel.Channel]().
		Entries(&channels).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	renames, err := resourcev0.StagedRenames(tx)
	if err != nil {
		return nil, err
	}
	var (
		held      = make(set.Set[string], len(channels))
		current   = make(map[resource.ID]string, len(channels))
		moved     = make(map[string]string, len(renames))
		ambiguous = make(set.Set[string])
	)
	for _, c := range channels {
		held.Add(c.Name)
		current[c.ResourceID()] = c.Name
	}
	for _, r := range renames {
		if r.ID.Type != resource.TypeChannel || held.Contains(r.From) {
			continue
		}
		to := current[r.ID]
		if prev, ok := moved[r.From]; ok && prev != to {
			ambiguous.Add(r.From)
		}
		moved[r.From] = to
	}
	return func(name string) (string, bool) {
		if held.Contains(name) {
			return name, strings.IndexByte(name, '-') >= 0
		}
		if ambiguous.Contains(name) {
			return "", false
		}
		to, ok := moved[name]
		return to, ok
	}, nil
}

// calculationTextMigration writes each channel name in the expression of each
// calculation in its new form (textRename). It runs after the calculations move to
// their own records and before anything compiles them.
var calculationTextMigration = migrate.New(
	"v63_rename_calculation_channels",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		rename, err := textRename(ctx, tx)
		if err != nil {
			return err
		}
		var calcs []calculation.Calculation
		if err = gorp.NewRetrieve[calculation.Key, calculation.Calculation]().
			Entries(&calcs).
			Exec(ctx, tx); err != nil {
			return err
		}
		var changed []calculation.Calculation
		for _, c := range calcs {
			if e := arcv4.RenameChannels(c.Expression, rename); e != c.Expression {
				c.Expression = e
				changed = append(changed, c)
			}
		}
		return gorp.WrapWriter[calculation.Key, calculation.Calculation](tx).
			Set(ctx, changed...)
	},
)

// arcTextMigration writes each channel name in the text of each Arc and in the function
// bodies of each graph in its new form (textRename). It runs before anything compiles
// an Arc.
var arcTextMigration = migrate.New(
	"v63_rename_arc_channels",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		rename, err := textRename(ctx, tx)
		if err != nil {
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
			if renameSource(&b.Source, rename) {
				a.SetBase(b)
				changed = append(changed, a)
			}
		}
		return gorp.WrapWriter[arc.Key, arc.Arc](tx).Set(ctx, changed...)
	},
)

// renameSource applies rename to each channel name in src, and reports whether it
// changed src.
func renameSource(src *arc.Source, rename func(string) (string, bool)) bool {
	changed := false
	switch v := src.Variant.(type) {
	case arc.TextSource:
		raw := v.Text.Materialize().Raw
		if renamed := arcv4.RenameChannels(raw, rename); renamed != raw {
			v.Text = text.Text{Doc: text.Create(renamed)}
			src.Variant, changed = v, true
		}
	case arc.GraphSource:
		for i, f := range v.Graph.Functions {
			renamed := arcv4.RenameChannels(f.Body.Raw, rename)
			if renamed != f.Body.Raw {
				v.Graph.Functions[i].Body.Raw = renamed
				src.Variant, changed = v, true
			}
		}
	}
	return changed
}

// newRenameReport returns the migration that writes a warning status for each rename
// that the name and clash passes staged, and then deletes the staged renames. It runs
// after the text passes.
func newRenameReport(statuses *status.Service) gorp.Migration {
	return migrate.New(
		"v63_report_renames",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			renames, err := resourcev0.StagedRenames(tx)
			if err != nil || len(renames) == 0 {
				return err
			}
			names, err := resourceNames(ctx, tx)
			if err != nil {
				return err
			}
			w := statuses.NewWriter(tx)
			for _, r := range renames {
				noun := "channel"
				if r.ID.Type == resource.TypeGroup {
					noun = "folder"
				}
				name := names[r.ID]
				s := status.Status[any]{
					Name:    name,
					Variant: status.VariantWarning,
					Message: "Renamed " + noun,
					Description: fmt.Sprintf(
						"The upgrade renamed %s %q to %q. A name must now be unique "+
							"in its folder and hold no '.' or invisible character.",
						noun,
						r.From,
						name,
					),
					Time: telem.Now(),
				}
				if err = w.Set(ctx, &s); err != nil {
					return err
				}
			}
			return resourcev0.ClearRenames(ctx, tx)
		},
	)
}

// resourceNames returns the name of each channel and channel folder by resource ID.
func resourceNames(ctx context.Context, tx gorp.Tx) (map[resource.ID]string, error) {
	var channels []channel.Channel
	if err := gorp.NewRetrieve[channel.Key, channel.Channel]().
		Entries(&channels).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	folders, err := channelFolders(ctx, tx)
	if err != nil {
		return nil, err
	}
	names := make(map[resource.ID]string, len(channels)+len(folders))
	for _, c := range channels {
		names[c.ResourceID()] = c.Name
	}
	for _, f := range folders {
		names[f.ResourceID()] = f.Name
	}
	return names, nil
}
