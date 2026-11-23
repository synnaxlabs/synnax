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
	"fmt"
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
	// Picks maps the text form of the record of a visible entry to the pick for it. An
	// entry with no pick takes its default.
	Picks map[string]Choice
	// Hidden is the pick for every entry that Visible hides and that allows it. Nil
	// leaves those entries at their defaults.
	Hidden *Choice
	// Visible reports whether the caller may see a record. An entry whose record it
	// cannot see counts in Plan.Hidden instead. Nil sees every record.
	Visible func(resource.ID) bool
}

// Plan returns every record that c touches with the picks of opts, as seen by snap, and
// the token that Apply checks. A delete pick on a channel adds its holders, as Apply
// does. It writes nothing. A pick that is not a choice, that names no visible entry, or
// that its entry does not allow fails with a validation error.
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
	chosen, err := newPicks(opts)
	if err != nil {
		return Plan{}, err
	}
	r, err := s.resolve(ctx, snap, keys, chosen)
	if err != nil {
		return Plan{}, err
	}
	if r.unknown != nil {
		return Plan{}, r.unknown
	}
	r.plan.Token = token(keys, r.plan)
	return r.plan, nil
}

// ApplyOptions are the inputs of the caller to Apply.
type ApplyOptions struct {
	// Token is the token of the plan that the caller saw. Empty skips the check.
	Token string
	// Picks are the picks of PlanOptions.Picks.
	Picks map[string]Choice
	// Hidden is the pick of PlanOptions.Hidden.
	Hidden *Choice
	// Visible reports whether the caller may see a record, as PlanOptions.Visible does.
	Visible func(resource.ID) bool
}

// picks is PlanOptions with its picks keyed by record.
type picks struct {
	named   map[resource.ID]Choice
	hidden  *Choice
	visible func(resource.ID) bool
}

// newPicks parses the picks of opts. A pick that is not a choice, or that names no
// record, fails with a validation error.
func newPicks(opts PlanOptions) (picks, error) {
	if opts.Hidden != nil && !opts.Hidden.IsValid() {
		return picks{}, errors.Wrapf(
			validate.ErrValidation,
			"%s is not a choice",
			*opts.Hidden,
		)
	}
	p := picks{
		named:   make(map[resource.ID]Choice, len(opts.Picks)),
		hidden:  opts.Hidden,
		visible: opts.Visible,
	}
	for text, pick := range opts.Picks {
		if !pick.IsValid() {
			return picks{}, errors.Wrapf(
				validate.ErrValidation,
				"%s is not a choice",
				pick,
			)
		}
		id, err := resource.ParseID(text)
		if err != nil {
			return picks{}, errors.Wrapf(validate.ErrValidation, "no entry is %s", text)
		}
		p.named[id] = pick
	}
	return p, nil
}

// sees reports whether the caller may see r.
func (p picks) sees(r resource.ID) bool { return p.visible == nil || p.visible(r) }

// pickOf returns the pick for e, or its default, or nothing when e allows no pick. A
// pick that names a hidden entry does not reach it, so it answers as a pick for no
// entry does.
func (p picks) pickOf(e Entry) (Choice, error) {
	var (
		pick    Choice
		ok      bool
		visible = p.sees(e.Record)
	)
	switch {
	case visible:
		pick, ok = p.named[e.Record]
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
	return "", errors.Wrapf(
		validate.ErrValidation,
		"entry %s cannot take %s",
		e.Record,
		pick,
	)
}

// Apply carries out c in tx with the picks of opts. With no picks, no hidden pick, and
// no token, it deletes as channel.Writer.DeleteMany does with allowInternal false,
// which applies the default of every record, and returns a plan of c with no entries.
// Otherwise it plans c with the picks in tx, as Plan does, and fails with a
// PlanChangedError before any write when opts.Token names another plan. It returns the
// entries that it applied, as opts.Visible lets the caller see them. A restrict entry
// that takes cancel fails Apply with the gorp.RestrictError of its record. A delete
// pick on a record other than a channel deletes it through ServiceConfig.Delete before
// the channels go. Picks fail as they fail Plan. Given a gorp.DB, Apply runs in a
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
	keys, err := deletedKeys(c)
	if err != nil {
		return Plan{}, err
	}
	w := s.cfg.Channel.NewWriter(tx)
	if len(opts.Picks) == 0 && opts.Hidden == nil && opts.Token == "" {
		if err = w.DeleteMany(ctx, keys, false); err != nil {
			return Plan{}, err
		}
		return Plan{Change: c}, nil
	}
	chosen, err := newPicks(PlanOptions{
		Picks:   opts.Picks,
		Hidden:  opts.Hidden,
		Visible: opts.Visible,
	})
	if err != nil {
		return Plan{}, err
	}
	r, err := s.resolve(ctx, tx, keys, chosen)
	if err != nil {
		return Plan{}, err
	}
	if opts.Token != "" {
		if !matches(opts.Token, keys, r.plan) {
			r.plan.Token = token(keys, r.plan)
			return Plan{}, PlanChangedError{Plan: r.plan}
		}
		r.plan.Token = opts.Token
	}
	if r.unknown != nil {
		return Plan{}, r.unknown
	}
	if r.cancel != nil {
		return Plan{}, *r.cancel
	}
	if len(r.records) > 0 {
		if err = s.cfg.Delete(ctx, tx, r.records); err != nil {
			return Plan{}, err
		}
	}
	if err = w.DeletePlanned(ctx, r.last.keys, r.last.effects); err != nil {
		return Plan{}, err
	}
	return r.plan, nil
}

// seen returns p for the delete of the channels with keys, with the entries whose
// records visible hides counted in Hidden by effect instead.
func seen(p Plan, keys []channel.Key, visible func(resource.ID) bool) Plan {
	p.Change = Change{Variant: DeleteChannelsChange{Keys: keys}}
	if visible == nil {
		return p
	}
	first := slices.IndexFunc(p.Entries, func(e Entry) bool {
		return !visible(e.Record)
	})
	if first < 0 {
		return p
	}
	entries := make([]Entry, first, len(p.Entries)-1)
	copy(entries, p.Entries)
	for _, e := range p.Entries[first:] {
		if visible(e.Record) {
			entries = append(entries, e)
		} else {
			p.Hidden.count(e.Effect)
		}
	}
	p.Entries = entries
	return p
}

// resolution is the plan of a change with picks, and what applying it does.
type resolution struct {
	// plan holds the entries that the change touches with the picks, as the caller sees
	// them, with no token.
	plan Plan
	// last is the plan of the last delete, with every channel that picks add.
	last planned
	// records are the records other than channels that picks delete.
	records []resource.ID
	// cancel is the error of the first entry that takes cancel, or nil.
	cancel *gorp.RestrictError
	// unknown is the error of a pick that names no entry, or nil. Apply reads the token
	// first, so a picked entry that another change removed reads as a changed plan.
	unknown error
}

// resolve plans the delete of the channels with keys in snap with chosen, then plans
// again with each channel that picks add to it. The plan holds the entries of the last
// plan and, for the channels that picks added, the entries that earlier plans gave
// them, sorted by record.
func (s *Service) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []channel.Key,
	chosen picks,
) (resolution, error) {
	var (
		r       resolution
		all     = keys
		earlier []Entry
		visible = set.New[resource.ID]()
		deleted = set.New[resource.ID]()
	)
	for {
		p, err := s.planDelete(ctx, snap, all)
		if err != nil {
			return resolution{}, err
		}
		more, picked, cancel, err := choose(p, chosen)
		if err != nil {
			return resolution{}, err
		}
		if r.cancel == nil {
			r.cancel = cancel
		}
		for _, e := range p.Entries {
			if chosen.sees(e.Record) {
				visible.Add(e.Record)
			}
		}
		for _, rec := range picked {
			if !deleted.Contains(rec) {
				deleted.Add(rec)
				r.records = append(r.records, rec)
			}
		}
		r.last = p
		if len(more) == 0 {
			break
		}
		earlier = append(earlier, p.Entries...)
		all = append(slices.Clip(all), more...)
	}
	for id := range chosen.named {
		if !visible.Contains(id) {
			r.unknown = errors.Wrapf(validate.ErrValidation, "no entry is %s", id)
			break
		}
	}
	applied := Plan{Entries: r.last.Entries, Kept: r.last.Kept}
	if len(earlier) > 0 {
		applied.Entries = withEarlier(r.last.Entries, earlier)
	}
	r.plan = seen(applied, keys, chosen.visible)
	return r, nil
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
// records that the picks delete, and the error of the first entry that takes cancel.
func choose(
	p planned,
	chosen picks,
) ([]channel.Key, []resource.ID, *gorp.RestrictError, error) {
	var (
		channels []channel.Key
		records  []resource.ID
		cancel   *gorp.RestrictError
	)
	for _, e := range p.Entries {
		pick, err := chosen.pickOf(e)
		if err != nil {
			return nil, nil, nil, err
		}
		switch {
		case pick == "":
		case pick == ChoiceCancel:
			if cancel == nil {
				restrict := p.cancels[e.Record]
				cancel = &restrict
			}
		case e.Record.Type != resource.TypeChannel:
			records = append(records, e.Record)
		default:
			k, err := channel.ParseKey(e.Record.Key)
			if err != nil {
				return nil, nil, nil, err
			}
			channels = append(channels, k)
		}
	}
	return channels, records, cancel, nil
}

// deletedKeys returns the keys of the channels that c deletes.
func deletedKeys(c Change) ([]channel.Key, error) {
	d, ok := c.Variant.(DeleteChannelsChange)
	if !ok {
		return nil, errors.Wrap(validate.ErrValidation, "unknown change")
	}
	return d.Keys, nil
}

// count counts one hidden entry with effect e.
func (h *Hidden) count(e Effect) {
	switch e {
	case EffectRestrict:
		h.Restrict++
	case EffectCascade:
		h.Cascade++
	case EffectDetach:
		h.Detach++
	default:
		panic(fmt.Sprintf("[impact] - no hidden count for effect %q", e))
	}
}
