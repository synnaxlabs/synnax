// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package impact plans the delete of records: every record that the delete touches, the
// picks for each, and the default. It applies a delete with the caller's picks.
package impact

import (
	"context"
	"fmt"
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Deleter plans and applies the delete of the records of one type.
type Deleter struct {
	// Plan returns the records that deleting ids removes, ids included, and what the
	// delete does to each record that holds one of them. It reads snap only.
	Plan func(ctx context.Context, snap gorp.Snapshot, ids []resource.ID) (
		removed []resource.ID,
		effects gorp.DeleteEffects,
		err error,
	)
	// Delete deletes removed in tx, with the effects that Plan returned for them in tx.
	Delete func(
		ctx context.Context,
		tx gorp.Tx,
		removed []resource.ID,
		effects gorp.DeleteEffects,
	) error
}

// ServiceConfig is the configuration for opening an impact service.
type ServiceConfig struct {
	// Deleters plans and deletes each type of record that a delete may remove: the
	// records that the caller deletes, and the records that the caller picks to delete.
	//
	// [REQUIRED]
	Deleters map[resource.Type]Deleter
	// RecordOf returns the record that the holder of a reference stands for.
	//
	// [REQUIRED]
	RecordOf func(holder gorp.Target) resource.ID
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Deleters = override.Nil(c.Deleters, other.Deleters)
	c.RecordOf = override.Nil(c.RecordOf, other.RecordOf)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("impact")
	v.NotNil("deleters", c.Deleters)
	v.NotNil("record_of", c.RecordOf)
	return v.Error()
}

// Service plans and applies the delete of records.
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

// Plan returns every record that the delete of records touches with the picks of opts,
// as seen by snap, and the token that Apply checks. A delete pick on a record adds it
// to the delete, as Apply does. A pick for a record that the plan does not reach has no
// effect, and the token does not hold it. Plan writes nothing. A pick that is not a
// choice, or that its entry does not allow, fails with a validation error, and so does
// a record of a type that the service cannot delete.
func (s *Service) Plan(
	ctx context.Context,
	snap gorp.Snapshot,
	records []resource.ID,
	opts PlanOptions,
) (Plan, error) {
	chosen, err := newPicks(opts)
	if err != nil {
		return Plan{}, err
	}
	r, err := s.resolve(ctx, snap, records, chosen)
	if err != nil {
		return Plan{}, err
	}
	r.plan.Token = token(records, r.plan)
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
// record, fails with a validation error for the pick with the smallest key. Choices are
// checked first.
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
	var invalid, unparsed []string
	for text, pick := range opts.Picks {
		if !pick.IsValid() {
			invalid = append(invalid, text)
			continue
		}
		id, err := resource.ParseID(text)
		if err != nil {
			unparsed = append(unparsed, text)
			continue
		}
		p.named[id] = pick
	}
	if len(invalid) > 0 {
		return picks{}, errors.Wrapf(
			validate.ErrValidation,
			"%s is not a choice",
			opts.Picks[slices.Min(invalid)],
		)
	}
	if len(unparsed) > 0 {
		return picks{}, errors.Wrapf(
			validate.ErrValidation,
			"%s is not a record",
			slices.Min(unparsed),
		)
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

// Apply deletes records in tx with the picks of opts. It plans the delete with the
// picks in tx, as Plan does, and fails with a PlanChangedError before any write when
// opts.Token names another plan; an empty token skips the check. With no picks, no
// hidden pick, and no token, it returns a plan with no entries. Otherwise it returns
// the entries that it applied, as opts.Visible lets the caller see them. A restrict
// entry that takes cancel fails Apply with the gorp.RestrictError of its record. Each
// type's records go through its Deleter, each holder before what it holds. Picks fail
// as they fail Plan. Given a gorp.DB, Apply runs in a transaction of its own.
func (s *Service) Apply(
	ctx context.Context,
	tx gorp.Tx,
	records []resource.ID,
	opts ApplyOptions,
) (applied Plan, err error) {
	if db, ok := tx.(*gorp.DB); ok {
		err = db.WithTx(ctx, func(tx gorp.Tx) error {
			applied, err = s.Apply(ctx, tx, records, opts)
			return err
		})
		return applied, err
	}
	chosen, err := newPicks(PlanOptions{
		Picks:   opts.Picks,
		Hidden:  opts.Hidden,
		Visible: opts.Visible,
	})
	if err != nil {
		return Plan{}, err
	}
	r, err := s.resolve(ctx, tx, records, chosen)
	if err != nil {
		return Plan{}, err
	}
	if opts.Token != "" {
		if !matches(opts.Token, records, r.plan) {
			r.plan.Token = token(records, r.plan)
			return Plan{}, PlanChangedError{Plan: r.plan}
		}
		r.plan.Token = opts.Token
	}
	if r.cancel != nil {
		return Plan{}, *r.cancel
	}
	for _, b := range r.last.batches {
		if err = b.deleter.Delete(ctx, tx, b.removed, b.effects); err != nil {
			return Plan{}, err
		}
	}
	if len(opts.Picks) == 0 && opts.Hidden == nil && opts.Token == "" {
		return Plan{Records: records}, nil
	}
	return r.plan, nil
}

// seen returns p for the delete of records, with the entries whose records visible
// hides counted in Hidden by effect instead.
func seen(p Plan, records []resource.ID, visible func(resource.ID) bool) Plan {
	p.Records = records
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

// resolution is the plan of a delete with picks, and what applying it does.
type resolution struct {
	// plan holds the entries that the delete touches with the picks, as the caller sees
	// them, with no token.
	plan Plan
	// last is the plan of the last delete, with every record that picks add.
	last planned
	// cancel is the error of the first entry that takes cancel, or nil.
	cancel *gorp.RestrictError
}

// resolve plans the delete of records in snap with chosen, then plans again with each
// record that picks add to it. The plan holds the entries of the last plan and, for the
// records that picks added, the entries that earlier plans gave them, sorted by record.
func (s *Service) resolve(
	ctx context.Context,
	snap gorp.Snapshot,
	records []resource.ID,
	chosen picks,
) (resolution, error) {
	var (
		r       resolution
		all     = records
		earlier []Entry
		deleted = set.New(records...)
	)
	for {
		p, err := s.planDelete(ctx, snap, all)
		if err != nil {
			return resolution{}, err
		}
		picked, cancel, err := choose(p, chosen)
		if err != nil {
			return resolution{}, err
		}
		if r.cancel == nil {
			r.cancel = cancel
		}
		var more []resource.ID
		for _, rec := range picked {
			if !deleted.Contains(rec) {
				deleted.Add(rec)
				more = append(more, rec)
			}
		}
		r.last = p
		if len(more) == 0 {
			break
		}
		earlier = append(earlier, p.Entries...)
		all = append(slices.Clip(all), more...)
	}
	applied := Plan{Entries: r.last.Entries, Kept: r.last.Kept}
	if len(earlier) > 0 {
		applied.Entries = withEarlier(r.last.Entries, earlier)
	}
	r.plan = seen(applied, records, chosen.visible)
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

// choose applies chosen to the entries of p. It returns the records that the picks
// delete, and the error of the first entry that takes cancel, in the order that the
// delete reaches the entries.
func choose(p planned, chosen picks) ([]resource.ID, *gorp.RestrictError, error) {
	var (
		records   []resource.ID
		cancelled set.Set[resource.ID]
	)
	for _, e := range p.Entries {
		pick, err := chosen.pickOf(e)
		if err != nil {
			return nil, nil, err
		}
		switch pick {
		case ChoiceCancel:
			if cancelled == nil {
				cancelled = set.New[resource.ID]()
			}
			cancelled.Add(e.Record)
		case ChoiceDelete:
			records = append(records, e.Record)
		}
	}
	for _, r := range p.restricts {
		if cancelled.Contains(r) {
			restrict := p.cancels[r]
			return records, &restrict, nil
		}
	}
	return records, nil, nil
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
