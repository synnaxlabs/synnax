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
	"bytes"
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
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

// IsTombstone reports whether id, as group.Service.Children lists a record, names a
// tombstone.
func IsTombstone(id resource.ID) bool { return id.Type == tombstoneTableName }

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

// takeLists gives each new channel of channels, one with no local key, the rewires of
// the tombstone at its path, or none: only a heal or a delete writes a list. It returns
// the dead keys of those tombstones by the position of their channel.
func (w Writer) takeLists(
	ctx context.Context,
	channels []Channel,
) (map[int]Key, error) {
	var deadAt map[int]Key
	for i := range channels {
		c := &channels[i]
		if c.LocalKey != 0 {
			continue
		}
		c.Rewires = nil
		dead, ok, err := w.takeList(ctx, c)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if deadAt == nil {
			deadAt = make(map[int]Key)
		}
		deadAt[i] = dead
	}
	return deadAt, nil
}

// takeList gives c the list of the tombstone at its path once the service heals. It
// returns the dead key of that tombstone, and false when it takes no list.
func (w Writer) takeList(ctx context.Context, c *Channel) (Key, bool, error) {
	if !w.svc.healing.Load() {
		return 0, false, nil
	}
	t, ok, err := w.tombstoneOf(ctx, c.Name, c.Group)
	if err != nil || !ok {
		return 0, false, err
	}
	c.Rewires = t.Rewires
	return t.Key, true, nil
}

// healed maps dead to live in to, which it makes when nil, and returns to.
func healed(to map[Key]Key, dead, live Key) map[Key]Key {
	if to == nil {
		to = make(map[Key]Key, 1)
	}
	to[dead] = live
	return to
}

// Heal makes each later create, rename, or move that takes the path of a tombstone
// heal, and heals each tombstone whose path a channel holds now. Until Heal, such a
// write leaves the tombstone. Call it once every table that holds channels has opened.
func (s *Service) Heal(ctx context.Context) error {
	s.healing.Store(true)
	return s.db.WithTx(ctx, func(tx gorp.Tx) error {
		var ts []Tombstone
		if err := s.tombstones.table.NewRetrieve().
			Entries(&ts).
			Exec(ctx, tx); err != nil {
			return err
		}
		keys := make([]Key, 0, len(ts))
		for _, t := range ts {
			k, ok, err := s.names.Holder(ctx, tx, t.Name)
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
		var channels []Channel
		if err := s.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Channel](keys...)).
			Entries(&channels).
			Exec(ctx, tx); err != nil {
			return err
		}
		return s.NewWriter(tx).heal(ctx, channels...)
	})
}

// heal gives each of channels, which are written, that holds the path of a tombstone
// the list of that tombstone, and then heals the links of those tombstones.
func (w Writer) heal(ctx context.Context, channels ...Channel) error {
	to := make(map[Key]Key, len(channels))
	for _, c := range channels {
		dead, ok, err := w.takeList(ctx, &c)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err = w.svc.table.NewCreate().Entry(&c).Exec(ctx, w.tx); err != nil {
			return err
		}
		to[dead] = c.Key()
	}
	return w.healLinks(ctx, to)
}

// healLinks moves the links that hold each dead key of to, and that Retargets accepts,
// onto the key it maps to, and deletes the tombstones of the dead keys.
func (w Writer) healLinks(ctx context.Context, to map[Key]Key) error {
	if len(to) == 0 {
		return nil
	}
	dead := make([]Key, 0, len(to))
	var holders []gorp.Dependent
	for k := range to {
		dead = append(dead, k)
		deps, err := w.svc.table.Dependents(ctx, w.tx, k)
		if err != nil {
			return err
		}
		for _, d := range deps {
			retargets, err := w.svc.cfg.Retargets(d)
			if err != nil {
				return err
			}
			if retargets {
				holders = append(holders, d)
			}
		}
	}
	if err := w.svc.table.RetargetHolders(ctx, w.tx, to, holders...); err != nil {
		return err
	}
	return w.svc.tombstones.table.NewDelete().
		Where(gorp.MatchKeys[Key, Tombstone](dead...)).
		Exec(ctx, w.tx)
}

// tombstoneOf returns the tombstone at the path of name in g, and false when none is
// there.
func (w Writer) tombstoneOf(
	ctx context.Context,
	name string,
	g *group.Key,
) (Tombstone, bool, error) {
	k, ok, err := w.tombstoneAt(ctx, name, g)
	if err != nil || !ok {
		return Tombstone{}, false, err
	}
	var t Tombstone
	err = w.svc.tombstones.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Tombstone](k)).
		Entry(&t).
		Exec(ctx, w.tx)
	return t, err == nil, err
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

// Tombstones returns the tombstones in the groups with keys, as snap sees them.
func (s *Service) Tombstones(
	ctx context.Context,
	snap gorp.Snapshot,
	groups ...group.Key,
) ([]Tombstone, error) {
	_, dead, err := s.tombstonesIn(ctx, snap, groups)
	return dead, err
}

// tombstonesIn returns the keys of the tombstones in groups and the tombstones, as snap
// sees them.
func (s *Service) tombstonesIn(
	ctx context.Context,
	snap gorp.Snapshot,
	groups []group.Key,
) ([]Key, []Tombstone, error) {
	keys, err := s.tombstones.groups.Get(ctx, snap, groups...)
	if err != nil || len(keys) == 0 {
		return nil, nil, err
	}
	dead := make([]Tombstone, 0, len(keys))
	if err = s.tombstones.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Tombstone](keys...)).
		Entries(&dead).
		Exec(ctx, snap); err != nil {
		return nil, nil, err
	}
	return keys, dead, nil
}

// DeleteTombstones deletes the tombstones in the groups with keys and returns them. A
// tombstone refuses the delete of its group, so a group delete calls it first.
func (w Writer) DeleteTombstones(
	ctx context.Context,
	groups ...group.Key,
) ([]Tombstone, error) {
	keys, dead, err := w.svc.tombstonesIn(ctx, w.tx, groups)
	if err != nil || len(keys) == 0 {
		return nil, err
	}
	return dead, w.svc.tombstones.table.NewDelete().
		Where(gorp.MatchKeys[Key, Tombstone](keys...)).
		Exec(ctx, w.tx)
}

// MoveTombstones moves the tombstones in each group that moves maps to the group that
// it maps to, or to no group for nil, and returns the tombstones that drop instead. A
// tombstone drops when its new path holds a channel or a tombstone, or a tombstone that
// moved before it. A channel in a group that moves maps counts at the group it maps to.
// Tombstones move in order of group key, then name. No group that moves maps to may be
// a key of moves.
func (w Writer) MoveTombstones(
	ctx context.Context,
	moves map[group.Key]*group.Key,
) ([]Tombstone, error) {
	moved, dropped, err := w.svc.tombstoneMoves(ctx, w.tx, moves)
	if err != nil {
		return nil, err
	}
	if len(dropped) > 0 {
		keys := make([]Key, len(dropped))
		for i, t := range dropped {
			keys[i] = t.Key
		}
		if err = w.svc.tombstones.table.NewDelete().
			Where(gorp.MatchKeys[Key, Tombstone](keys...)).
			Exec(ctx, w.tx); err != nil {
			return nil, err
		}
	}
	if len(moved) > 0 {
		if err = w.svc.tombstones.table.NewCreate().
			Entries(&moved).
			Exec(ctx, w.tx); err != nil {
			return nil, err
		}
	}
	return dropped, nil
}

// DroppedTombstones returns the tombstones that MoveTombstones with moves would drop,
// as snap sees them.
func (s *Service) DroppedTombstones(
	ctx context.Context,
	snap gorp.Snapshot,
	moves map[group.Key]*group.Key,
) ([]Tombstone, error) {
	_, dropped, err := s.tombstoneMoves(ctx, snap, moves)
	return dropped, err
}

// tombstoneMoves returns the tombstones that MoveTombstones with moves moves, at their
// new groups, and the ones that it drops, as snap sees them.
func (s *Service) tombstoneMoves(
	ctx context.Context,
	snap gorp.Snapshot,
	moves map[group.Key]*group.Key,
) (moved, dropped []Tombstone, err error) {
	_, dead, err := s.tombstonesIn(ctx, snap, slices.Collect(maps.Keys(moves)))
	if err != nil || len(dead) == 0 {
		return nil, nil, err
	}
	slices.SortFunc(dead, func(a, b Tombstone) int {
		if c := bytes.Compare(a.Group[:], b.Group[:]); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	targets := make(set.Set[path], len(dead))
	names := make([]Name, len(dead))
	for i, t := range dead {
		targets.Add(path{group: movedGroup(t.Group, moves), name: t.Name})
		names[i] = t.Name
	}
	var live []Channel
	if err = s.NewRetrieve().
		Where(MatchNames(names...)).
		Entries(&live).
		Exec(ctx, snap); err != nil {
		return nil, nil, err
	}
	taken := make(set.Set[path], len(live))
	for _, c := range live {
		taken.Add(path{group: movedGroup(c.Group, moves), name: c.Name})
	}
	var held []Tombstone
	if err = s.tombstones.table.NewRetrieve().
		Where(gorp.Match(func(_ gorp.Context, t *Tombstone) (bool, error) {
			p, _ := tombstonePath(*t)
			return targets.Contains(p), nil
		})).
		Entries(&held).
		Exec(ctx, snap); err != nil {
		return nil, nil, err
	}
	for _, t := range held {
		p, _ := tombstonePath(t)
		taken.Add(p)
	}
	for _, t := range dead {
		p := path{group: movedGroup(t.Group, moves), name: t.Name}
		if taken.Contains(p) {
			dropped = append(dropped, t)
			continue
		}
		taken.Add(p)
		to := moves[*t.Group]
		t.Group = nil
		if to != nil {
			t.Group = new(*to)
		}
		moved = append(moved, t)
	}
	return moved, dropped, nil
}

// movedGroup returns the key of the group that g maps to in moves, or of g itself when
// moves does not hold it. No group is the zero key.
func movedGroup(g *group.Key, moves map[group.Key]*group.Key) group.Key {
	if g == nil {
		return group.Key{}
	}
	to, ok := moves[*g]
	if !ok {
		return *g
	}
	if to == nil {
		return group.Key{}
	}
	return *to
}

// writeHeld writes, in tx, the tombstone of c, which a delete removes, when a link
// other than a rewires entry holds it, or when it holds rewires.
func (t *tombstones) writeHeld(
	ctx context.Context,
	tx gorp.Tx,
	c Channel,
	held []gorp.Dependent,
) error {
	if len(c.Rewires) == 0 &&
		!slices.ContainsFunc(
			held,
			func(d gorp.Dependent) bool { return !isRewire(d) },
		) {
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
