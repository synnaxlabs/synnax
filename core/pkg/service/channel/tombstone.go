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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/telem"
)

const (
	tombstoneTableName = "channel_tombstone"
	// tombstoneSpace keeps one tombstone per path apart from the names of live
	// channels, so a tombstone never blocks a create.
	tombstoneSpace = "channel.tombstone"
	// rewiresReference is the name of the reference of the rewires of a channel and of
	// a tombstone.
	rewiresReference = "rewires.from"
)

// MirrorTables returns the tables of the records that a channel change writes beside
// the channel: its tombstones. Pass them as bind.ServiceConfig.Mirrors.
func MirrorTables() []string { return []string{tombstoneTableName} }

// path is the folder and the name of a channel.
type path struct {
	group group.Key
	name  string
}

func tombstonePath(t *Tombstone) (path, bool) {
	p := path{name: t.Name}
	if t.Group != nil {
		p.group = *t.Group
	}
	return p, true
}

type tombstones struct {
	table  *gorp.Table[Key, Tombstone]
	paths  *gorp.UniqueIndex[Key, Tombstone, path]
	groups *gorp.KeyReference[Key, Tombstone, group.Key]
	// held is write, bound once so that a delete passes it with no allocation.
	held gorp.HeldFunc[Key, Channel]
}

func openTombstones(
	ctx context.Context,
	db *gorp.DB,
	ins alamos.Instrumentation,
) (*tombstones, error) {
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
	t.held = t.write
	var err error
	t.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Tombstone]{
		DB:              db,
		Name:            tombstoneTableName,
		Indexes:         []gorp.Index[Key, Tombstone]{t.paths},
		Instrumentation: ins,
		References: []gorp.Reference[Key, Tombstone]{
			t.groups,
			gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, Tombstone, Key]{
				Name:     rewiresReference,
				Table:    tableName,
				OnDelete: gorp.Weak,
				Kind:     gorp.Use,
				Targets:  func(t *Tombstone) []Key { return rewireTargets(t.Rewires) },
			}),
		},
	})
	return t, err
}

func rewireTargets(rewires []Rewire) []Key {
	var targets []Key
	for _, r := range rewires {
		if r.From != 0 {
			targets = append(targets, r.From)
		}
	}
	return targets
}

// hasTombstone reports whether a tombstone holds the path of name in the folder g.
func (w Writer) hasTombstone(
	ctx context.Context,
	name string,
	g *group.Key,
) (bool, error) {
	p, _ := tombstonePath(&Tombstone{Name: name, Group: g})
	return w.svc.tombstones.table.NewRetrieve().
		Where(w.svc.tombstones.paths.Filter(p)).
		Exists(ctx, w.tx)
}

// DeleteTombstones deletes the tombstones of the channels deleted from groups.
func (w Writer) DeleteTombstones(ctx context.Context, groups ...group.Key) error {
	return w.svc.tombstones.table.NewDelete().
		Where(w.svc.tombstones.groups.Filter(groups...)).
		Exec(ctx, w.tx)
}

// write writes a tombstone of c, which a delete removes, when a link other than a
// rewires entry holds c. The tombstone takes the rewires of c and a gap entry. It
// replaces a tombstone that waits at the path of c, keeping the entries of the waiting
// one that end before the new list starts.
func (t *tombstones) write(
	ctx context.Context,
	tx gorp.Tx,
	c Channel,
	held []gorp.Dependent,
) error {
	if !slices.ContainsFunc(held, func(d gorp.Dependent) bool { return !isRewire(d) }) {
		return nil
	}
	next := Tombstone{
		Key:     c.Key(),
		Name:    c.Name,
		Group:   c.Group,
		Rewires: withGap(c.Rewires, telem.Now()),
	}
	p, _ := tombstonePath(&next)
	var waiting []Tombstone
	if err := t.table.NewRetrieve().
		Where(t.paths.Filter(p)).
		Entries(&waiting).
		Exec(ctx, tx); err != nil {
		return err
	}
	if len(waiting) > 0 {
		keys := make([]Key, len(waiting))
		for i, w := range waiting {
			next.Rewires = after(w.Rewires, next.Rewires)
			keys[i] = w.Key
		}
		if err := t.table.NewDelete().
			Where(gorp.MatchKeys[Key, Tombstone](keys...)).
			Exec(ctx, tx); err != nil {
			return err
		}
	}
	return t.table.NewCreate().Entry(&next).Exec(ctx, tx)
}

func isRewire(d gorp.Dependent) bool {
	return d.Reference == rewiresReference &&
		(d.Holder.Table == tableName || d.Holder.Table == tombstoneTableName)
}

// withGap returns rewires with a gap entry that ends at now, or just after the last
// entry when that is later, so each at rises. A gap at the end of rewires grows to the
// new end, so gaps next to each other merge.
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

// after returns next behind the entries of prev that end before next starts, oldest
// first. The rest of prev drops. A gap that ends those entries merges into a gap that
// starts next.
func after(prev, next []Rewire) []Rewire {
	n := 0
	for n < len(prev) && prev[n].At < next[0].At {
		n++
	}
	if n > 0 && prev[n-1].From == 0 && next[0].From == 0 {
		n--
	}
	return append(slices.Clone(prev[:n]), next...)
}
