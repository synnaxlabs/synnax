// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/telem"
)

const (
	// tombstoneTableName is the name of the table of tombstones.
	tombstoneTableName = "channel_tombstone"
	// tombstoneSpace is the space of the index that keeps one tombstone per path.
	tombstoneSpace = "channel.tombstone"
	// rewiresReference is the name of the reference of a rewires list on a channel or
	// a tombstone.
	rewiresReference = "rewires.from"
)

// MirrorTables returns the tables of the records that a channel change writes beside
// the channel: its tombstones. Pass them as bind.ServiceConfig.Mirrors.
func MirrorTables() []string { return []string{tombstoneTableName} }

// path is the group and name of a channel or a tombstone.
type path struct {
	group group.Key
	name  string
}

func tombstonePath(t Tombstone) (path, bool) {
	var p path
	if t.Group != nil {
		p.group = *t.Group
	}
	p.name = t.Name
	return p, true
}

// tombstones is the table of tombstones, with the index of their paths and their
// reference to their group.
type tombstones struct {
	table  *gorp.Table[Key, Tombstone]
	paths  *gorp.UniqueIndex[Key, Tombstone, path]
	groups *gorp.KeyReference[Key, Tombstone, group.Key]
	// write is write, bound once so that a delete allocates nothing to pass it.
	write gorp.HeldFunc[Key, Channel]
}

// openTombstones opens the table of tombstones. Its references are written by hand:
// Oracle generates one retrievable record per Go package, and a package of its own
// would import channel in a cycle.
func openTombstones(ctx context.Context, db *gorp.DB) (*tombstones, error) {
	t := &tombstones{
		paths: gorp.NewUniqueIndex("path", tombstoneSpace, tombstonePath),
		groups: gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, Tombstone, group.Key]{
			Name:     "group",
			Table:    "group",
			OnDelete: gorp.Restrict,
			Kind:     gorp.Place,
			Targets: func(t *Tombstone) []group.Key {
				if t.Group == nil {
					return nil
				}
				return []group.Key{*t.Group}
			},
		}),
	}
	t.write = t.writeHeld
	var err error
	t.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Tombstone]{
		DB:      db,
		Name:    tombstoneTableName,
		Indexes: []gorp.Index[Key, Tombstone]{t.paths},
		References: []gorp.Reference[Key, Tombstone]{
			t.groups,
			gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, Tombstone, Key]{
				Name:     rewiresReference,
				Table:    tableName,
				OnDelete: gorp.Weak,
				Kind:     gorp.Use,
				// No direction: a tombstone reads no channel, so uses leave it out.
				Targets: func(t *Tombstone) []Key { return rewireTargets(t.Rewires) },
				Detach:  func(t *Tombstone, k Key) { detachRewire(t.Rewires, k) },
			}),
		},
	})
	return t, err
}

// rewireTargets returns the past holders in rewires, without gap entries.
func rewireTargets(rewires []Rewire) []Key {
	var targets []Key
	for _, r := range rewires {
		if r.From != 0 {
			targets = append(targets, r.From)
		}
	}
	return targets
}

func detachRewire(rewires []Rewire, k Key) {
	for i := range rewires {
		if rewires[i].From == k {
			rewires[i].From = 0
		}
	}
}

// withGap returns rewires with a gap entry that ends at now, the later of now and the
// last at plus 1 ns. A gap at the end of rewires takes the new end.
func withGap(rewires []Rewire, now telem.TimeStamp) []Rewire {
	out := slices.Clone(rewires)
	at := now
	if n := len(out); n > 0 {
		next := out[n-1].At
		if next < telem.TimeStampMax {
			next++
		}
		at = max(at, next)
		if out[n-1].From == 0 {
			out[n-1].At = at
			return out
		}
	}
	return append(out, Rewire{At: at})
}

// clearTombstones deletes, in the transaction of w, the tombstone at the path of each
// of channels, so a path never holds a channel and a tombstone at once.
func (w Writer) clearTombstones(ctx context.Context, channels ...Channel) error {
	var keys []Key
	for _, c := range channels {
		k, ok, err := w.tombstoneAt(ctx, c.Name, c.Group)
		if err != nil {
			return err
		}
		if ok {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return w.svc.tombstones.table.NewDelete().
		Where(gorp.MatchKeys[Key, Tombstone](keys...)).
		Exec(ctx, w.tx)
}

// tombstoneAt returns the key of the tombstone at the path of name in g, and false when
// none is there.
func (w Writer) tombstoneAt(
	ctx context.Context,
	name string,
	g *group.Key,
) (Key, bool, error) {
	p, _ := tombstonePath(Tombstone{Name: name, Group: g})
	return w.svc.tombstones.paths.Holder(ctx, w.tx, p)
}

// DeleteTombstones deletes the tombstones in the groups with keys and returns them. A
// tombstone refuses the delete of its group, so a group delete calls it first.
func (w Writer) DeleteTombstones(
	ctx context.Context,
	groups ...group.Key,
) ([]Tombstone, error) {
	keys, err := w.svc.tombstones.groups.Get(ctx, w.tx, groups...)
	if err != nil || len(keys) == 0 {
		return nil, err
	}
	dead := make([]Tombstone, 0, len(keys))
	if err = w.svc.tombstones.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Tombstone](keys...)).
		Entries(&dead).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	return dead, w.svc.tombstones.table.NewDelete().
		Where(gorp.MatchKeys[Key, Tombstone](keys...)).
		Exec(ctx, w.tx)
}

// writeHeld writes, in tx, the tombstone of c, which a delete removes, when a link
// other than a rewires entry holds it.
func (t *tombstones) writeHeld(
	ctx context.Context,
	tx gorp.Tx,
	c Channel,
	held []gorp.Dependent,
) error {
	if !slices.ContainsFunc(held, func(d gorp.Dependent) bool { return !isRewire(d) }) {
		return nil
	}
	ts := Tombstone{
		Key:     c.Key(),
		Name:    c.Name,
		Group:   c.Group,
		Rewires: withGap(c.Rewires, telem.Now()),
	}
	return t.table.NewCreate().Entry(&ts).Exec(ctx, tx)
}

// isRewire reports whether d is the link of a channel or tombstone to a past holder.
func isRewire(d gorp.Dependent) bool {
	return d.Reference == rewiresReference &&
		(d.Holder.Table == tableName || d.Holder.Table == tombstoneTableName)
}
