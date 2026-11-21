// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package impact plans each change to channels: every record that the change touches,
// the picks for each, and the default. It applies a change with the caller's picks.
package impact

import (
	"context"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
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
	c.RecordOf = override.Nil(c.RecordOf, other.RecordOf)
	c.Delete = override.Nil(c.Delete, other.Delete)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("impact")
	v.NotNil("channel", c.Channel)
	v.NotNil("record_of", c.RecordOf)
	v.NotNil("delete", c.Delete)
	return v.Error()
}

// Service applies changes to channels.
type Service struct{ cfg ServiceConfig }

// OpenService opens a Service from cfgs.
func OpenService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// PlanChangedError is the error of an Apply whose plan changed after the caller
// planned.
type PlanChangedError struct {
	// Plan is the plan as it is now, as the caller may see it.
	Plan Plan
}

// Error implements error.
func (PlanChangedError) Error() string {
	return "the records that the change touches changed"
}

// PlanOptions are the inputs of the caller to Plan.
type PlanOptions struct {
	// Visible reports whether the caller may see a record. An entry whose record it
	// cannot see counts in Plan.Hidden instead. Nil sees every record.
	Visible func(resource.ID) bool
}

// Plan returns every record that c touches, as seen by snap, with the picks for each
// and the token that Apply checks. It writes nothing.
func (s *Service) Plan(
	ctx context.Context,
	snap gorp.Snapshot,
	c Change,
	opts PlanOptions,
) (Plan, error) {
	keys, err := deletedKeys(c)
	if err != nil {
		return Plan{}, err
	}
	p, err := s.planDelete(ctx, snap, keys)
	if err != nil {
		return Plan{}, err
	}
	p.Token = token(keys, p.Entries)
	return seen(p.Plan, keys, opts.Visible), nil
}

// ApplyOptions are the inputs of the caller to Apply.
type ApplyOptions struct {
	// Token is the token of the plan that the caller saw. Empty skips the check.
	Token string
	// Picks maps the text form of the record of a visible entry to the pick for it. An
	// entry with no pick takes its default.
	Picks map[string]Choice
	// Hidden is the pick for every entry that Visible hides and that allows it. Nil
	// leaves those entries at their defaults.
	Hidden *Choice
	// Visible reports whether the caller may see a record, as PlanOptions.Visible does.
	Visible func(resource.ID) bool
}

// picks is ApplyOptions with its picks keyed by record.
type picks struct {
	named   map[resource.ID]Choice
	hidden  *Choice
	visible func(resource.ID) bool
}

// newPicks parses the picks of opts. A pick that names no record fails with a
// validation error.
func newPicks(opts ApplyOptions) (picks, error) {
	p := picks{
		named:   make(map[resource.ID]Choice, len(opts.Picks)),
		hidden:  opts.Hidden,
		visible: opts.Visible,
	}
	for text, pick := range opts.Picks {
		id, err := resource.ParseID(text)
		if err != nil {
			return picks{}, errors.Wrapf(validate.ErrValidation, "no entry is %s", text)
		}
		p.named[id] = pick
	}
	return p, nil
}

// pickOf returns the pick for e, or its default, or nothing when e allows no pick.
func (p picks) pickOf(e Entry) (Choice, error) {
	var (
		pick    Choice
		ok      bool
		visible = p.visible == nil || p.visible(e.Record)
	)
	switch _, named := p.named[e.Record]; {
	case visible:
		pick, ok = p.named[e.Record]
	case named:
		return "", errors.Wrapf(validate.ErrValidation, "no entry is %s", e.Record)
	case p.hidden != nil && len(e.Choices) > 0:
		pick, ok = *p.hidden, true
	}
	if !ok {
		if e.Default == nil {
			return "", nil
		}
		return *e.Default, nil
	}
	if slices.Contains(e.Choices, pick) {
		return pick, nil
	}
	if !visible {
		return "", errors.Wrapf(
			validate.ErrValidation,
			"a hidden entry cannot take %s",
			pick,
		)
	}
	return "", errors.Wrapf(
		validate.ErrValidation,
		"entry %s cannot take %s",
		e.Record,
		pick,
	)
}

// Apply carries out c in tx with the picks of opts. With no picks, no hidden pick, and
// no token, it deletes as channel.Writer.DeleteMany does with allowInternal false,
// which applies the default of every record, and returns an empty plan. Otherwise it
// plans c in tx, fails with a PlanChangedError before any write when opts.Token names
// another plan, and returns the entries that it applied, as opts.Visible lets the
// caller see them. A restrict entry that takes cancel fails Apply with the
// gorp.RestrictError of its record. A delete pick on a channel adds the channel to the
// delete, and the holders of that channel take their picks or defaults. A delete pick
// on another record deletes it through ServiceConfig.Delete before the channels go. A
// pick for no visible entry, or one that its entry does not allow, fails with a
// validation error. Given a gorp.DB, Apply runs in a transaction of its own.
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
	keys, err := deletedKeys(c)
	if err != nil {
		return Plan{}, err
	}
	w := s.cfg.Channel.NewWriter(tx)
	if len(opts.Picks) == 0 && opts.Hidden == nil && opts.Token == "" {
		return Plan{}, w.DeleteMany(ctx, keys, false)
	}
	chosen, err := newPicks(opts)
	if err != nil {
		return Plan{}, err
	}
	first, err := s.planDelete(ctx, tx, keys)
	if err != nil {
		return Plan{}, err
	}
	if opts.Token != "" {
		if first.Token = token(keys, first.Entries); first.Token != opts.Token {
			return Plan{}, PlanChangedError{Plan: seen(first.Plan, keys, opts.Visible)}
		}
	}
	applied, last, records, err := s.resolve(ctx, tx, keys, first, chosen)
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
	return seen(applied, keys, opts.Visible), nil
}

// seen returns p for the delete of the channels with keys, with the entries whose
// records visible hides counted in Hidden instead.
func seen(p Plan, keys []channel.Key, visible func(resource.ID) bool) Plan {
	p.Change = Change{Variant: DeleteChannelsChange{Keys: keys}}
	if visible == nil {
		return p
	}
	entries := p.Entries
	p.Entries = make([]Entry, 0, len(entries))
	for _, e := range entries {
		if visible(e.Record) {
			p.Entries = append(p.Entries, e)
		} else {
			p.Hidden++
		}
	}
	return p
}

// resolve applies chosen to first, the plan of the delete of the channels with keys in
// tx, then plans again with each channel that picks add to it. It returns the entries
// of the last plan and, for the channels that picks added, the entries that earlier
// plans gave them, sorted by record; the last plan; and the records other than channels
// that picks delete.
func (s *Service) resolve(
	ctx context.Context,
	tx gorp.Tx,
	keys []channel.Key,
	first planned,
	chosen picks,
) (Plan, planned, []resource.ID, error) {
	var (
		p       = first
		earlier []Entry
		seen    = set.New[resource.ID]()
		deleted = set.New[resource.ID]()
		records []resource.ID
	)
	for {
		more, picked, err := choose(p, chosen)
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
		if p, err = s.planDelete(ctx, tx, keys); err != nil {
			return Plan{}, planned{}, nil, err
		}
	}
	for id := range chosen.named {
		if !seen.Contains(id) {
			return Plan{}, planned{}, nil, errors.Wrapf(
				validate.ErrValidation,
				"no entry is %s",
				id,
			)
		}
	}
	applied := Plan{Entries: p.Entries, Kept: p.Kept}
	if len(earlier) > 0 {
		applied.Entries = withEarlier(p.Entries, earlier)
	}
	return applied, p, records, nil
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

// choose applies chosen to the entries of p. It returns the channels and the other
// records that the picks delete, and the cancel error of the first entry that takes
// cancel.
func choose(p planned, chosen picks) ([]channel.Key, []resource.ID, error) {
	var (
		channels []channel.Key
		records  []resource.ID
	)
	for _, e := range p.Entries {
		pick, err := chosen.pickOf(e)
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

// deletedKeys returns the keys of the channels that c deletes.
func deletedKeys(c Change) ([]channel.Key, error) {
	d, ok := c.Variant.(DeleteChannelsChange)
	if !ok {
		return nil, errors.Wrap(validate.ErrValidation, "unknown change")
	}
	return d.Keys, nil
}
