// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package impact plans each change to channels and groups: every record that the change
// touches, the picks for each, and the default. It applies a change with the caller's
// picks.
package impact

import (
	"context"
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening an impact service.
type ServiceConfig struct {
	// Channel is the service of the channels that a change touches.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Group is the service of the groups that a change touches.
	//
	// [REQUIRED]
	Group *group.Service
	// RecordOf returns the record that the holder of a reference stands for.
	//
	// [REQUIRED]
	RecordOf func(holder gorp.Target) resource.ID
	// Delete deletes, in tx, the records other than channels that the caller picks to
	// delete. It fails on a record that it cannot delete.
	//
	// [REQUIRED]
	Delete func(ctx context.Context, tx gorp.Tx, records []resource.ID) error
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Group = override.Nil(c.Group, other.Group)
	c.RecordOf = override.Nil(c.RecordOf, other.RecordOf)
	c.Delete = override.Nil(c.Delete, other.Delete)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("impact")
	v.NotNil("channel", c.Channel)
	v.NotNil("group", c.Group)
	v.NotNil("record_of", c.RecordOf)
	v.NotNil("delete", c.Delete)
	return v.Error()
}

// Service applies changes to channels and groups.
type Service struct{ cfg ServiceConfig }

// OpenService opens a Service from cfgs.
func OpenService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// ApplyOptions are the inputs of the caller to Apply.
type ApplyOptions struct {
	// Picks maps the text form of the record of an entry to the pick for it. An entry
	// with no pick takes its default.
	Picks map[string]Choice
}

// Apply carries out c in tx with the picks of opts. A delete of channels with no picks
// deletes as channel.Writer.DeleteMany does with allowInternal false, which applies the
// default of every record, and returns an empty plan. With picks, it plans c in tx and
// returns the entries that it applied, with the count of the records that keep a
// deleted channel. A restrict entry that takes cancel fails Apply with the
// gorp.RestrictError of its record. A delete pick on a channel adds the channel to the
// delete, and the holders of that channel take their picks or defaults. A delete pick
// on another record deletes it through ServiceConfig.Delete before the channels go. A
// delete of groups deletes the tombstones in them, then the groups as
// group.Writer.Delete does, and plans no entries. A pick for no entry, or one that its
// entry does not allow, fails with a validation error. Given a gorp.DB, Apply runs in a
// transaction of its own.
func (s *Service) Apply(
	ctx context.Context,
	tx gorp.Tx,
	c Change,
	opts ApplyOptions,
) (applied Plan, err error) {
	if db, ok := tx.(*gorp.DB); ok {
		err = db.WithTx(ctx, func(tx gorp.Tx) error {
			applied, err = s.Apply(ctx, tx, c, opts)
			return err
		})
		return applied, err
	}
	switch v := c.Variant.(type) {
	case DeleteChannelsChange:
		return s.deleteChannels(ctx, tx, v.Keys, opts)
	case DeleteGroupsChange:
		return s.deleteGroups(ctx, tx, v.Keys, opts)
	}
	// Formatting the variant would make every change escape to the heap.
	return Plan{}, errors.New("[impact] - unknown change")
}

// deleteGroups deletes the groups with keys and the tombstones in them first. Its plan
// lists the channel of each tombstone as a cascade entry, which takes no pick, with the
// tombstone's path.
func (s *Service) deleteGroups(
	ctx context.Context,
	tx gorp.Tx,
	keys []group.Key,
	opts ApplyOptions,
) (Plan, error) {
	picks, err := parsePicks(opts.Picks)
	if err != nil {
		return Plan{}, err
	}
	if err = pickWithoutEntry(picks, nil); err != nil {
		return Plan{}, err
	}
	dead, err := s.cfg.Channel.NewWriter(tx).DeleteTombstones(ctx, keys...)
	if err != nil {
		return Plan{}, err
	}
	var paths []string
	if len(dead) > 0 {
		if paths, err = s.tombstonePaths(ctx, tx, keys, dead); err != nil {
			return Plan{}, err
		}
	}
	if err = s.cfg.Group.NewWriter(tx).Delete(ctx, keys...); err != nil {
		return Plan{}, err
	}
	var p Plan
	for i, t := range dead {
		p.Entries = append(p.Entries, Entry{
			Record: channel.ResourceID(t.Key),
			Effect: EffectCascade,
			Path:   &paths[i],
		})
	}
	slices.SortFunc(p.Entries, compareEntries)
	return p, nil
}

// tombstonePaths returns the path of each of dead from the deleted group that holds it.
// A group inside a deleted one is deleted too, so each walk ends at a deleted group.
func (s *Service) tombstonePaths(
	ctx context.Context,
	tx gorp.Tx,
	keys []group.Key,
	dead []channel.Tombstone,
) ([]string, error) {
	groups := make([]group.Group, 0, len(keys))
	if err := s.cfg.Group.NewRetrieve().
		Where(group.MatchKeys(keys...)).
		Entries(&groups).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	deleted := make(map[group.Key]group.Group, len(groups))
	for _, g := range groups {
		deleted[g.Key] = g
	}
	paths := make([]string, len(dead))
	var segments []string
	for i, t := range dead {
		segments = append(segments[:0], t.Name)
		g, ok := deleted[*t.Group]
		for ok {
			segments = append(segments, g.Name)
			if g.Parent == nil || g.Parent.Type != resource.TypeGroup {
				break
			}
			parent, err := group.KeyFromResourceID(*g.Parent)
			if err != nil {
				return nil, err
			}
			g, ok = deleted[parent]
		}
		slices.Reverse(segments)
		paths[i] = strings.Join(segments, ".")
	}
	return paths, nil
}

// pickWithoutEntry returns a validation error that names the smallest pick of picks
// whose record no entry has, or nil when each has one.
func pickWithoutEntry(
	picks map[resource.ID]Choice,
	entries set.Set[resource.ID],
) error {
	var missing []string
	for id := range picks {
		if !entries.Contains(id) {
			missing = append(missing, id.String())
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return errors.Wrapf(validate.ErrValidation, "no entry is %s", slices.Min(missing))
}

// deleteChannels deletes the channels with keys, as Apply describes.
func (s *Service) deleteChannels(
	ctx context.Context,
	tx gorp.Tx,
	keys []channel.Key,
	opts ApplyOptions,
) (Plan, error) {
	w := s.cfg.Channel.NewWriter(tx)
	if len(opts.Picks) == 0 {
		return Plan{}, w.DeleteMany(ctx, keys, false)
	}
	applied, last, records, err := s.resolve(ctx, tx, keys, opts.Picks)
	if err != nil {
		return Plan{}, err
	}
	if len(records) > 0 {
		if err = s.cfg.Delete(ctx, tx, records); err != nil {
			return Plan{}, err
		}
	}
	if err = w.DeletePlanned(ctx, last.keys, last.effects); err != nil {
		return Plan{}, err
	}
	applied.Change = Change{Variant: DeleteChannelsChange{Keys: keys}}
	return applied, nil
}

// resolve plans the delete of the channels with keys in tx, then plans again with each
// channel that picks add to it. It returns the entries of the last plan and, for the
// channels that picks added, the entries that earlier plans gave them, sorted by
// record; the last plan; and the records other than channels that picks delete.
func (s *Service) resolve(
	ctx context.Context,
	tx gorp.Tx,
	keys []channel.Key,
	named map[string]Choice,
) (Plan, planned, []resource.ID, error) {
	picks, err := parsePicks(named)
	if err != nil {
		return Plan{}, planned{}, nil, err
	}
	var (
		p       planned
		earlier []Entry
		seen    = set.New[resource.ID]()
		deleted = set.New[resource.ID]()
		records []resource.ID
	)
	for {
		if p, err = s.planDelete(ctx, tx, keys); err != nil {
			return Plan{}, planned{}, nil, err
		}
		more, picked, err := choose(p, picks)
		if err != nil {
			return Plan{}, planned{}, nil, err
		}
		for _, e := range p.Entries {
			seen.Add(e.Record)
		}
		for _, r := range picked {
			if !deleted.Contains(r) {
				deleted.Add(r)
				records = append(records, r)
			}
		}
		if len(more) == 0 {
			break
		}
		earlier = append(earlier, p.Entries...)
		keys = append(keys, more...)
	}
	if err := pickWithoutEntry(picks, seen); err != nil {
		return Plan{}, planned{}, nil, err
	}
	applied := Plan{Entries: p.Entries, Kept: p.Kept}
	if len(earlier) > 0 {
		applied.Entries = withEarlier(p.Entries, earlier)
	}
	return applied, p, records, nil
}

// parsePicks keys named by record. A pick that names no record fails with a validation
// error.
func parsePicks(named map[string]Choice) (map[resource.ID]Choice, error) {
	picks := make(map[resource.ID]Choice, len(named))
	for text, pick := range named {
		id, err := resource.ParseID(text)
		if err != nil {
			return nil, errors.Wrapf(validate.ErrValidation, "no entry is %s", text)
		}
		picks[id] = pick
	}
	return picks, nil
}

// withEarlier returns entries and each entry of earlier for a record that entries do
// not hold, sorted by record.
func withEarlier(entries, earlier []Entry) []Entry {
	held := set.New[resource.ID]()
	for _, e := range entries {
		held.Add(e.Record)
	}
	merged := slices.Clone(entries)
	for _, e := range earlier {
		if !held.Contains(e.Record) {
			held.Add(e.Record)
			merged = append(merged, e)
		}
	}
	slices.SortFunc(merged, compareEntries)
	return merged
}

// choose applies picks to the entries of p. It returns the channels and the other
// records that the picks delete, and the cancel error of the first entry that takes
// cancel.
func choose(
	p planned,
	picks map[resource.ID]Choice,
) ([]channel.Key, []resource.ID, error) {
	var (
		channels []channel.Key
		records  []resource.ID
	)
	for _, e := range p.Entries {
		pick, err := pickOf(e, picks)
		if err != nil {
			return nil, nil, err
		}
		switch {
		case pick == "":
		case pick == ChoiceCancel:
			return nil, nil, p.cancels[e.Record]
		case e.Record.Type != resource.TypeChannel:
			records = append(records, e.Record)
		default:
			k, err := channel.ParseKey(e.Record.Key)
			if err != nil {
				return nil, nil, err
			}
			channels = append(channels, k)
		}
	}
	return channels, records, nil
}

// pickOf returns the pick that picks holds for e, or its default, or nothing when e
// allows no pick.
func pickOf(e Entry, picks map[resource.ID]Choice) (Choice, error) {
	pick, ok := picks[e.Record]
	if !ok {
		if e.Default == nil {
			return "", nil
		}
		return *e.Default, nil
	}
	if slices.Contains(e.Choices, pick) {
		return pick, nil
	}
	return "", errors.Wrapf(
		validate.ErrValidation,
		"entry %s cannot take %s",
		e.Record,
		pick,
	)
}
