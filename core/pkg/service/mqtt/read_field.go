// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/validate"
)

// validateReadConfig checks the rules of cfg that need no other resource: each field
// has a kind, the enum labels of a field are valid, and a bytes field with an empty
// pointer is the only enabled field of its entry.
func validateReadConfig(cfg *ReadConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	return v2.PlainFields(cfg, func(path string, entry PlainReadEntry) error {
		fields := entry.Fields
		enabled := lo.CountBy(
			fields,
			func(f ReadField) bool { return !f.Base().Disabled },
		)
		for i, f := range fields {
			fieldPath := path + "." + strconv.Itoa(i)
			if f.Variant == nil {
				return errors.Wrapf(
					validate.ErrValidation,
					"%s.type: required",
					fieldPath,
				)
			}
			if err := v2.ReadOf(f).Validate(); err != nil {
				return errors.Wrap(err, fieldPath)
			}
			base := f.Base()
			if f.Type() == payload.KindBytes && base.Pointer == "" &&
				!base.Disabled && enabled > 1 {
				return errors.Wrapf(
					validate.ErrValidation,
					"%s: a bytes field with an empty pointer must be the only "+
						"enabled field",
					fieldPath,
				)
			}
		}
		return nil
	})
}

// readChannels returns the v2.Channels that reads channels from svc.
func readChannels(svc *channel.Service) v2.Channels {
	return func(
		ctx context.Context,
		snap gorp.Snapshot,
		keys []channel.Key,
	) (map[channel.Key]v2.Channel, error) {
		var found []channel.Channel
		if err := svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&found).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		return lo.SliceToMap(found, func(c channel.Channel) (channel.Key, v2.Channel) {
			return c.Key(), v2.Channel{
				Name:     c.Name,
				DataType: c.DataType,
				Index:    c.IsIndex,
			}
		}), nil
	}
}

// resolveReadConfig checks each enabled field of an enabled entry of cfg against its
// channel and clears its data type. It skips a field with no channel, or one that does
// not exist.
func resolveReadConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels v2.Channels,
	cfg *ReadConfig,
) error {
	found, err := v2.FieldChannels(ctx, snap, channels, cfg)
	if err != nil {
		return err
	}
	return v2.PlainFields(cfg, func(path string, entry PlainReadEntry) error {
		if entry.Disabled {
			return nil
		}
		fields := entry.Fields
		for i, f := range fields {
			base := f.Base()
			ch, ok := found[base.Channel]
			if base.Disabled || !ok {
				continue
			}
			r := v2.ReadOf(f)
			if err := r.Check(ch.DataType, ch.Index); err != nil {
				return errors.Wrapf(err, "%s.%d", path, i)
			}
			r.Number.DataType = nil
			fields[i] = v2.NewReadField(base, r)
		}
		return nil
	})
}
