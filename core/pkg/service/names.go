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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arcv4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	resourcev0 "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// sibling is a name in a channel folder, or at the top when parent is the zero key.
type sibling struct {
	parent group.Key
	name   string
}

// nameClashMigration renames one side of each channel and channel folder that hold the
// same name in one folder, after NFC. The channel takes the first free suffix when a
// name pass renamed it and not the folder; else the folder does. It stages each rename,
// so it runs after the name passes and before the text passes.
var nameClashMigration = migrate.New(
	"v63_name_clashes",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		staged, err := resourcev0.StagedRenames(tx)
		if err != nil {
			return err
		}
		renamed := make(set.Set[resource.ID], len(staged))
		for _, r := range staged {
			renamed.Add(r.Resource)
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
			at := sibling{name: validate.NormalizeName(c.Name)}
			if c.Group != nil {
				at.parent = *c.Group
			}
			names.Add(c.Name)
			siblings.Add(at)
			holders[at] = i
		}
		at := make([]sibling, len(folders))
		for i, f := range folders {
			at[i].name = validate.NormalizeName(f.Name)
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
			if renamed.Contains(c.ResourceID()) && !renamed.Contains(f.ResourceID()) {
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

// channelNames returns the name of every channel.
func channelNames(ctx context.Context, tx gorp.Tx) (set.Set[string], error) {
	var channels []channel.Channel
	if err := gorp.NewRetrieve[channel.Key, channel.Channel]().
		Entries(&channels).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	names := make(set.Set[string], len(channels))
	for _, c := range channels {
		names.Add(c.Name)
	}
	return names, nil
}

// calculationTextMigration quotes each dashed channel name in the expression of each
// calculation (arcv4.QuoteDashedNames). It runs after the calculations move to their
// own records and before anything compiles them.
var calculationTextMigration = migrate.New(
	"v63_quote_calculation_names",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		held, err := channelNames(ctx, tx)
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
			if e := arcv4.QuoteDashedNames(
				c.Expression,
				held.Contains,
			); e != c.Expression {
				c.Expression = e
				changed = append(changed, c)
			}
		}
		return gorp.WrapWriter[calculation.Key, calculation.Calculation](tx).
			Set(ctx, changed...)
	},
)

// arcTextMigration quotes each dashed channel name in the text of each Arc and in the
// function bodies of each graph (arcv4.QuoteDashedNames). It runs before anything
// compiles an Arc.
var arcTextMigration = migrate.New(
	"v63_quote_arc_names",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		held, err := channelNames(ctx, tx)
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
			if quoteSource(&b.Source, held.Contains) {
				a.SetBase(b)
				changed = append(changed, a)
			}
		}
		return gorp.WrapWriter[arc.Key, arc.Arc](tx).Set(ctx, changed...)
	},
)

// quoteSource quotes each dashed name that held reports true for in src, and reports
// whether it changed src.
func quoteSource(src *arc.Source, held func(string) bool) bool {
	changed := false
	switch v := src.Variant.(type) {
	case arc.TextSource:
		raw := v.Text.Materialize().Raw
		if quoted := arcv4.QuoteDashedNames(raw, held); quoted != raw {
			v.Text = text.Text{Doc: text.Create(quoted)}
			src.Variant, changed = v, true
		}
	case arc.GraphSource:
		for i, f := range v.Graph.Functions {
			if quoted := arcv4.QuoteDashedNames(
				f.Body.Raw,
				held,
			); quoted != f.Body.Raw {
				v.Graph.Functions[i].Body.Raw = quoted
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
				if r.Resource.Type == resource.TypeGroup {
					noun = "folder"
				}
				name := names[r.Resource]
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
