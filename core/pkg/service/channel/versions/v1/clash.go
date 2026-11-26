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
	"fmt"
	"slices"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	group "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// NewClashMigration returns the migration that renames one side of each channel and
// channel folder that hold one name in NFC in one folder, and stages each rename
// (resource.StageRename). The folder yields, unless a name pass renamed the channel and
// not the folder, or the channel is top-level and holds group.MetricsFolder. Of several
// top-level metrics folders, the one that holds the metrics index channel of host keeps
// the name, or else the first by key. A folder takes the first suffix (_2, _3) that no
// sibling holds, and a channel the first that also no other channel holds. It runs
// after the name passes.
func NewClashMigration(host node.Key) gorp.Migration {
	// A v0.62 Core made its metrics index channel in its metrics folder.
	index := fmt.Sprintf("sy_node_%s_metrics_time", host)
	return migrate.New(
		"v63_name_clashes",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			c, err := openClash(ctx, tx)
			if err != nil {
				return err
			}
			return c.resolve(ctx, index)
		},
	)
}

// clash holds the channels and channel folders of one run of the clash pass, and the
// names that they hold in NFC.
type clash struct {
	tx       gorp.Tx
	from     map[resource.ID]string
	channels []Channel
	folders  []group.Group
	// at is the place of each folder, by index in folders.
	at       []sibling
	names    set.Set[string]
	siblings set.Set[sibling]
	holders  map[sibling]int
}

func openClash(ctx context.Context, tx gorp.Tx) (*clash, error) {
	staged, err := resource.StagedRenames(tx)
	if err != nil {
		return nil, err
	}
	c := &clash{tx: tx, from: make(map[resource.ID]string, len(staged))}
	for _, r := range staged {
		c.from[r.ID] = r.From
	}
	if err = gorp.NewRetrieve[Key, Channel]().
		Entries(&c.channels).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	if err = gorp.NewRetrieve[group.Key, group.Group]().
		Where(gorp.Match(func(_ gorp.Context, g *group.Group) (bool, error) {
			return g.Scope == resource.TypeChannel, nil
		})).
		Entries(&c.folders).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	slices.SortFunc(
		c.folders,
		func(a, b group.Group) int { return a.Key.Compare(b.Key) },
	)
	c.names = make(set.Set[string], len(c.channels))
	c.siblings = make(set.Set[sibling], len(c.channels)+len(c.folders))
	c.holders = make(map[sibling]int, len(c.channels))
	for i, ch := range c.channels {
		at := sibling{name: resource.NormalizeName(ch.Name)}
		if ch.Group != nil {
			at.group = *ch.Group
		}
		c.names.Add(at.name)
		c.siblings.Add(at)
		c.holders[at] = i
	}
	c.at = make([]sibling, len(c.folders))
	for i, f := range c.folders {
		if c.at[i].group, err = parentKey(f); err != nil {
			return nil, err
		}
		c.at[i].name = resource.NormalizeName(f.Name)
		c.siblings.Add(c.at[i])
	}
	return c, nil
}

// parentKey returns the key of the folder that holds f, or the zero key when f is
// top-level.
func parentKey(f group.Group) (group.Key, error) {
	if f.Parent == nil {
		return group.Key{}, nil
	}
	if f.Parent.Type != resource.TypeGroup {
		return group.Key{}, errors.Newf(
			"channel folder %s sits in %s",
			f.Key,
			*f.Parent,
		)
	}
	return uuid.Parse(f.Parent.Key)
}

func (c *clash) resolve(ctx context.Context, index string) error {
	metrics := sibling{name: group.MetricsFolder}
	kept := c.metricsFolder(index)
	for i := range c.folders {
		if i != kept && c.at[i] == metrics {
			if err := c.moveFolder(ctx, i); err != nil {
				return err
			}
		}
	}
	for i, f := range c.folders {
		h, ok := c.holders[c.at[i]]
		if !ok {
			continue
		}
		ch := c.channels[h]
		_, channelRenamed := c.from[ch.ResourceID()]
		_, folderRenamed := c.from[f.ResourceID()]
		var err error
		if i == kept || channelRenamed && !folderRenamed {
			err = c.moveChannel(ctx, ch, c.at[i].group)
		} else {
			err = c.moveFolder(ctx, i)
		}
		if err != nil {
			return err
		}
	}
	// The metrics service makes the folder later when the store holds none.
	if h, ok := c.holders[metrics]; ok && kept < 0 {
		return c.moveChannel(ctx, c.channels[h], group.Key{})
	}
	return nil
}

// metricsFolder returns the index of the top-level metrics folder that keeps its name:
// the one that holds the channel named index, or else the first. It returns -1 when no
// top-level folder holds the name.
func (c *clash) metricsFolder(index string) int {
	var holder *group.Key
	for _, ch := range c.channels {
		if ch.Name == index {
			holder = ch.Group
		}
	}
	kept := -1
	for i, f := range c.folders {
		if c.at[i] == (sibling{name: group.MetricsFolder}) &&
			(kept < 0 || holder != nil && *holder == f.Key) {
			kept = i
		}
	}
	return kept
}

// base returns the name that the suffix of a moved item starts from: the rewrite of its
// old name when a name pass renamed it, empty when that rewrite is empty, or else name.
func (c *clash) base(id resource.ID, name, empty string) string {
	old, ok := c.from[id]
	if !ok {
		return resource.NormalizeName(name)
	}
	if rewritten := resource.SanitizeName(old); rewritten != "" {
		return rewritten
	}
	return empty
}

func (c *clash) moveChannel(ctx context.Context, ch Channel, parent group.Key) error {
	first := c.base(ch.ResourceID(), ch.Name, "channel_"+ch.Key().String())
	name := resource.FreeName(first, func(name string) bool {
		return c.names.Contains(name) ||
			c.siblings.Contains(sibling{group: parent, name: name})
	})
	c.names.Add(name)
	c.siblings.Add(sibling{group: parent, name: name})
	if err := resource.StageRename(
		ctx,
		c.tx,
		ch.ResourceID(),
		ch.Name,
		resource.ReasonClash,
	); err != nil {
		return err
	}
	ch.Name = name
	return gorp.WrapWriter[Key, Channel](c.tx).Set(ctx, ch)
}

func (c *clash) moveFolder(ctx context.Context, i int) error {
	f, parent := c.folders[i], c.at[i].group
	name := resource.FreeName(
		c.base(f.ResourceID(), f.Name, "folder"),
		func(name string) bool {
			return c.siblings.Contains(sibling{group: parent, name: name})
		},
	)
	c.siblings.Add(sibling{group: parent, name: name})
	c.at[i].name = name
	if err := resource.StageRename(
		ctx,
		c.tx,
		f.ResourceID(),
		f.Name,
		resource.ReasonClash,
	); err != nil {
		return err
	}
	f.Name = name
	return gorp.WrapWriter[group.Key, group.Group](c.tx).Set(ctx, f)
}
