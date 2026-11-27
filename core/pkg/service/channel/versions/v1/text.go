// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"slices"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/set"
)

// NewCalculationTextMigration returns the migration that writes the new name of each
// renamed channel into the expression of each calculated channel (TextRename), read
// with cfg as a v0.62 Core read it. It runs after the name and clash passes, and
// before anything compiles a calculation.
func NewCalculationTextMigration(cfg parser.Config) gorp.Migration {
	return migrate.New(
		"v63_rename_calculation_channels",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var channels []Channel
			if err := gorp.NewRetrieve[Key, Channel]().
				Entries(&channels).
				Exec(ctx, tx); err != nil {
				return err
			}
			byKey, err := TextChannels(ctx, tx, channels)
			if err != nil {
				return err
			}
			renames, err := TextRename(tx, byKey)
			if err != nil || renames == nil {
				return err
			}
			w := Rewriter{Renames: renames, Config: cfg, Ambient: stl.NewSymbols()}
			var changed []Channel
			for _, c := range channels {
				if c.Expression == "" {
					continue
				}
				res, err := w.Rewrite(
					ctx,
					Source{Text: c.Expression, Form: FormCalculation},
				)
				if err != nil {
					return err
				}
				if err = StageKept(ctx, tx, c.ResourceID(), "", res); err != nil {
					return err
				}
				if res.Text != c.Expression {
					c.Expression = res.Text
					changed = append(changed, c)
				}
			}
			return gorp.WrapWriter[Key, Channel](tx).Set(ctx, changed...)
		},
	)
}

// TextRename returns how text that a v0.62 Core stored reads each channel name, or nil
// when the upgrade staged no rename. channels holds each channel as it is now, by key.
// A name in text reads the channel that a v0.62 Core bound it to: of the channels that
// held it before the upgrade (each staged rename put back), the one with the highest
// key.
func TextRename(r kv.Reader, channels map[Key]TextChannel) (*Renames, error) {
	renames, err := resource.StagedRenames(r)
	if err != nil || len(renames) == 0 {
		return nil, err
	}
	from := make(map[Key]string, len(renames))
	for _, rn := range renames {
		if rn.ID.Type != resource.TypeChannel {
			continue
		}
		local, err := strconv.ParseUint(rn.ID.Key, 10, 32)
		if err != nil {
			return nil, err
		}
		key := Key(local)
		if _, ok := channels[key]; !ok {
			return nil, errors.Newf("staged rename of %s names no record", rn.ID)
		}
		from[key] = rn.From
	}
	t := &Renames{
		bound:    make(map[string]binding, len(channels)),
		channels: channels,
		after:    make(map[string]Key, len(from)),
	}
	for key, ch := range channels {
		before, renamed := from[key]
		if !renamed {
			before = ch.Name
		} else if k, ok := t.after[ch.Name]; !ok || key > k {
			t.after[ch.Name] = key
		}
		if b, ok := t.bound[before]; !ok || key > b.key {
			t.bound[before] = binding{key: key, renamed: renamed}
		}
	}
	return t, nil
}

// TextChannels returns each of channels as the text passes read it, by key, with the
// names of its folders as tx holds them.
func TextChannels(
	ctx context.Context,
	tx gorp.Tx,
	channels []Channel,
) (map[Key]TextChannel, error) {
	var folders []groupv1.Group
	if err := gorp.NewRetrieve[groupv1.Key, groupv1.Group]().
		Where(gorp.Match(func(_ gorp.Context, g *groupv1.Group) (bool, error) {
			return g.Scope == resource.TypeChannel, nil
		})).
		Entries(&folders).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	byKey := make(map[groupv1.Key]groupv1.Group, len(folders))
	for _, f := range folders {
		byKey[f.Key] = f
	}
	out := make(map[Key]TextChannel, len(channels))
	for _, c := range channels {
		names, err := folderNames(byKey, c.Group)
		if err != nil {
			return nil, err
		}
		out[c.Key()] = TextChannel{Name: c.Name, DataType: c.DataType, Folders: names}
	}
	return out, nil
}

// folderNames returns the names of the folder of key and of each folder above it, from
// the top level down. A folder that no record holds ends the walk, as the dangling
// reference step that runs later moves its channels to the top level. So does a folder
// that the walk already passed, since a v0.62 Core stored a parent with no cycle check.
func folderNames(
	folders map[groupv1.Key]groupv1.Group,
	key *groupv1.Key,
) ([]string, error) {
	var (
		names  []string
		passed = make(set.Set[groupv1.Key])
	)
	for key != nil && !passed.Contains(*key) {
		passed.Add(*key)
		f, ok := folders[*key]
		if !ok {
			break
		}
		names = append(names, f.Name)
		if key = nil; f.Parent != nil && f.Parent.Type == resource.TypeGroup {
			parent, err := uuid.Parse(f.Parent.Key)
			if err != nil {
				return nil, err
			}
			key = &parent
		}
	}
	slices.Reverse(names)
	return names, nil
}
