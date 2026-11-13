// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	"github.com/synnaxlabs/x/errors"
	xpayload "github.com/synnaxlabs/x/payload"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Channel holds what a read field depends on in the channel it writes to.
type Channel struct {
	// DataType is the data type of the channel.
	DataType telem.DataType
	// IsIndex is true when the channel indexes others.
	IsIndex bool
}

// Channels returns the channels among keys, read through snap. A key with no channel is
// absent from the result.
type Channels func(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []channel.Key,
) (map[channel.Key]Channel, error)

func readChannels(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	cfg *ReadConfig,
) (map[channel.Key]Channel, error) {
	var keys []channel.Key
	for _, ep := range cfg.Endpoints {
		for _, f := range ep.Fields {
			if k := f.Base().Channel; k != 0 {
				keys = append(keys, k)
			}
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return channels(ctx, snap, lo.Uniq(keys))
}

// ReconcileReadConfig changes each field of cfg whose channel exists to agree with it:
// the field takes the kind of the channel's data type and drops its data type. Between
// float, integer, and unsigned kinds, entries and a fill convert when the channel holds
// them exactly, and a number becomes a boolean that is true when it is not 0, as the
// migration reads one. Any other entry or fill that the channel cannot hold drops, as
// do a fill on an index channel and an empty label on a kind other than string. warn
// reports each change.
func ReconcileReadConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	cfg *ReadConfig,
	warn Warn,
) error {
	found, err := readChannels(ctx, snap, channels, cfg)
	if err != nil {
		return err
	}
	for _, ep := range cfg.Endpoints {
		for i, f := range ep.Fields {
			base := f.Base()
			ch, ok := found[base.Channel]
			if !ok {
				continue
			}
			kind, ok := xpayload.KindOf(ch.DataType)
			if !ok {
				continue
			}
			fieldWarn := withField(warn, base.Key)
			if f.Type() != kind {
				fieldWarn(
					"changed the kind of a field to that of its channel",
					zap.Uint32("channel", uint32(base.Channel)),
					zap.String("old", string(f.Type())),
					zap.String("new", string(kind)),
				)
			}
			o := optionsOf(f)
			o.dataType = nil
			if ch.IsIndex && o.fill != nil {
				fieldWarn("dropped the fill of a field on an index channel")
				o.fill = nil
			}
			if o.timeFormat == "" {
				o.timeFormat = payload.TimeFormatISO8601
			}
			o.entries = withoutEmptyLabels(o.entries, kind, fieldWarn)
			ep.Fields[i] = assemble(kind, base, o, &ch.DataType, fieldWarn)
		}
	}
	return nil
}

// ResolveReadConfig checks each enabled field of cfg against the channel it writes to
// and drops the field's data type. It skips a field with no channel or a channel that
// does not exist. It returns an error wrapping validate.ErrValidation when a field's
// kind is not that of its channel's data type, an entry or fill does not fit the
// channel, or a field on an index channel has a fill.
func ResolveReadConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	cfg *ReadConfig,
) error {
	found, err := readChannels(ctx, snap, channels, cfg)
	if err != nil {
		return err
	}
	v := validate.New("http.read_config")
	for ei, ep := range cfg.Endpoints {
		for fi, f := range ep.Fields {
			base := f.Base()
			ch, ok := found[base.Channel]
			if base.Disabled || !ok {
				continue
			}
			v.Exec(func() error {
				return validate.PathedError(
					resolveReadField(&ep.Fields[fi], ch),
					"endpoints", strconv.Itoa(ei), "fields", strconv.Itoa(fi),
				)
			})
		}
	}
	return v.Error()
}

func resolveReadField(f *ReadField, ch Channel) error {
	kind, _ := xpayload.KindOf(ch.DataType)
	if f.Type() != kind {
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation,
			"kind %s does not match kind %s of data type %s",
			f.Type(), kind, ch.DataType,
		), "type")
	}
	o := optionsOf(*f)
	if ch.IsIndex && o.fill != nil {
		return validate.PathedError(errors.Wrap(
			validate.ErrValidation,
			"a field on an index channel cannot have a fill",
		), "fill")
	}
	if o.fill != nil {
		if _, ok := convert(o.fill, kind, &ch.DataType); !ok {
			return validate.PathedError(errors.Wrapf(
				validate.ErrValidation,
				"%v does not fit data type %s", o.fill, ch.DataType,
			), "fill")
		}
	}
	for i, e := range o.entries {
		if _, ok := convert(e.value, kind, &ch.DataType); !ok {
			return validate.PathedError(errors.Wrapf(
				validate.ErrValidation,
				"%v does not fit data type %s", e.value, ch.DataType,
			), "enum_values", strconv.Itoa(i), "value")
		}
	}
	o.dataType = nil
	*f = assemble(kind, f.Base(), o, nil, ignore)
	return nil
}

// ValidateReadConfig returns an error wrapping validate.ErrValidation when cfg breaks
// its schema rules, a field repeats an enum label, a field that is not a string field
// has an empty enum label, or an endpoint has an enabled bytes field with no pointer
// and another enabled field.
func ValidateReadConfig(cfg *ReadConfig) error {
	v := validate.New("http.read_config")
	v.Exec(cfg.Validate)
	for ei, ep := range cfg.Endpoints {
		enabled := lo.Filter(ep.Fields, func(f ReadField, _ int) bool {
			return !f.Base().Disabled
		})
		for fi, f := range ep.Fields {
			path := []string{"endpoints", strconv.Itoa(ei), "fields", strconv.Itoa(fi)}
			labels := make(set.Set[string])
			for i, e := range optionsOf(f).entries {
				v.Exec(func() error {
					labelPath := append(path, "enum_values", strconv.Itoa(i), "label")
					if e.label == "" && f.Type() != payload.KindString {
						return validate.PathedError(errors.Wrap(
							validate.ErrValidation,
							"enum label is empty, and only a string field reads one",
						), labelPath...)
					}
					if !labels.Contains(e.label) {
						labels.Add(e.label)
						return nil
					}
					return validate.PathedError(errors.Wrapf(
						validate.ErrValidation, "enum label %q repeats", e.label,
					), labelPath...)
				})
			}
			v.Exec(func() error {
				_, raw := f.Variant.(BytesReadField)
				if !raw || f.Base().Pointer != "" || f.Base().Disabled ||
					len(enabled) == 1 {
					return nil
				}
				return validate.PathedError(errors.Wrap(
					validate.ErrValidation,
					"a bytes field with no pointer reads the whole body, so it "+
						"must be the only enabled field",
				), append(path, "pointer")...)
			})
		}
	}
	return v.Error()
}
