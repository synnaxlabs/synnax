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

// ErrPlanChanged is the error of an Apply whose plan changed since the caller planned.
var ErrPlanChanged = errors.New("the records that the change touches changed")

// PlanChangedError wraps ErrPlanChanged with the new plan.
type PlanChangedError struct {
	// Plan is the plan as it is now.
	Plan Plan
}

// Error implements error.
func (PlanChangedError) Error() string { return ErrPlanChanged.Error() }

// Unwrap returns ErrPlanChanged.
func (PlanChangedError) Unwrap() error { return ErrPlanChanged }

// Service plans and applies changes to channels.
type Service struct{ cfg ServiceConfig }

// OpenService opens a Service from cfgs.
func OpenService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// Plan returns every record that c touches, as seen by snap, with the picks for each.
// It writes nothing.
func (s *Service) Plan(
	ctx context.Context,
	snap gorp.Snapshot,
	c Change,
) (Plan, error) {
	keys, err := deletedKeys(c)
	if err != nil {
		return Plan{}, err
	}
	p, err := s.planDelete(ctx, snap, keys)
	return p.Plan, err
}

// ApplyOptions are the inputs of the caller to Apply.
type ApplyOptions struct {
	// Token is the token of the plan that the caller saw. When it is empty, Apply
	// applies the picks to the plan as it is now.
	Token string
	// Picks maps the id of an entry to the pick for it. An entry with no pick takes its
	// default.
	Picks map[string]Choice
}

// Apply carries out c in tx with the picks of opts and returns the plan that it
// applied. It plans c again in tx, and fails with a PlanChangedError when opts.Token
// names another plan. A restrict entry that takes cancel fails Apply with the
// gorp.RestrictError of its record. A delete pick on a channel adds the channel to the
// delete, and the holders of that channel take their picks or defaults. A delete pick
// on another record deletes it through ServiceConfig.Delete before the channels go. A
// pick for no entry, or one that its entry does not allow, fails with a validation
// error. Apply deletes channels as channel.Writer.DeleteMany does with allowInternal
// false.
func (s *Service) Apply(
	ctx context.Context,
	tx gorp.Tx,
	c Change,
	opts ApplyOptions,
) (Plan, error) {
	keys, err := deletedKeys(c)
	if err != nil {
		return Plan{}, err
	}
	p, err := s.planDelete(ctx, tx, keys)
	if err != nil {
		return Plan{}, err
	}
	if opts.Token != "" && opts.Token != p.Token {
		return Plan{}, PlanChangedError{Plan: p.Plan}
	}
	var (
		picked  = set.New[string]()
		deleted = set.New[resource.ID]()
		records []resource.ID
	)
	for {
		more, picks, err := choose(p, opts.Picks)
		if err != nil {
			return Plan{}, err
		}
		for _, e := range p.Entries {
			if _, ok := opts.Picks[e.ID]; ok {
				picked.Add(e.ID)
			}
		}
		for _, r := range picks {
			if !deleted.Contains(r) {
				deleted.Add(r)
				records = append(records, r)
			}
		}
		if len(more) == 0 {
			break
		}
		keys = append(keys, more...)
		if p, err = s.planDelete(ctx, tx, keys); err != nil {
			return Plan{}, err
		}
	}
	for id := range opts.Picks {
		if !picked.Contains(id) {
			return Plan{}, errors.Wrapf(validate.ErrValidation, "no entry is %s", id)
		}
	}
	if len(records) > 0 {
		if err = s.cfg.Delete(ctx, tx, records); err != nil {
			return Plan{}, err
		}
	}
	if err = s.cfg.Channel.NewWriter(tx).DeleteMany(ctx, keys, false); err != nil {
		return Plan{}, err
	}
	return p.Plan, nil
}

// choose applies picks to the entries of p. It returns the channels and the other
// records that the picks delete, and the cancel error of the first entry that takes
// cancel.
func choose(
	p planned,
	picks map[string]Choice,
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
			return nil, nil, p.cancels[e.ID]
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
func pickOf(e Entry, picks map[string]Choice) (Choice, error) {
	pick, ok := picks[e.ID]
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
		e.ID,
		pick,
	)
}

// deletedKeys returns the keys of the channels that c deletes.
func deletedKeys(c Change) ([]channel.Key, error) {
	d, ok := c.Variant.(DeleteChannelsChange)
	if !ok {
		return nil, errors.Wrapf(validate.ErrValidation, "unknown change %T", c.Variant)
	}
	return d.Keys, nil
}
