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
	"strings"

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

// ApplyOptions are the inputs of the caller to Apply.
type ApplyOptions struct {
	// Picks maps the id of an entry to the pick for it. An entry with no pick takes its
	// default.
	Picks map[string]Choice
}

// Apply carries out c in tx with the picks of opts and returns the plan that it
// applied, which it makes in tx: every entry it met, with the count of the records that
// keep a deleted channel. A restrict entry that takes cancel fails Apply with the
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
	applied, keys, records, err := s.resolve(ctx, tx, keys, opts.Picks)
	if err != nil {
		return Plan{}, err
	}
	if len(records) > 0 {
		if err = s.cfg.Delete(ctx, tx, records); err != nil {
			return Plan{}, err
		}
	}
	if err = s.cfg.Channel.NewWriter(tx).DeleteMany(ctx, keys, false); err != nil {
		return Plan{}, err
	}
	applied.Change = c
	return applied, nil
}

// resolve plans the delete of the channels with keys in tx, then plans again with each
// channel that picks add to it. It returns every entry that it met, sorted by id, the
// keys of the channels to delete, and the other records that picks delete.
func (s *Service) resolve(
	ctx context.Context,
	tx gorp.Tx,
	keys []channel.Key,
	picks map[string]Choice,
) (Plan, []channel.Key, []resource.ID, error) {
	var (
		applied Plan
		seen    = set.New[string]()
		deleted = set.New[resource.ID]()
		records []resource.ID
	)
	for {
		p, err := s.planDelete(ctx, tx, keys)
		if err != nil {
			return Plan{}, nil, nil, err
		}
		applied.Kept = p.Kept
		more, picked, err := choose(p, picks)
		if err != nil {
			return Plan{}, nil, nil, err
		}
		for _, e := range p.Entries {
			if !seen.Contains(e.ID) {
				seen.Add(e.ID)
				applied.Entries = append(applied.Entries, e)
			}
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
		keys = append(keys, more...)
	}
	for id := range picks {
		if !seen.Contains(id) {
			return Plan{}, nil, nil, errors.Wrapf(
				validate.ErrValidation,
				"no entry is %s",
				id,
			)
		}
	}
	slices.SortFunc(applied.Entries, func(a, b Entry) int {
		return strings.Compare(a.ID, b.ID)
	})
	return applied, keys, records, nil
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
