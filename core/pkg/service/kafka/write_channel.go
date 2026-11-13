// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"context"
	"math"
	"slices"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// fieldOf returns how wc writes samples of data type dt. It returns an error wrapping
// validate.ErrValidation when wc's kind cannot write dt, or an entry of wc is not a
// value of dt.
func fieldOf(wc WriteChannel, dt telem.DataType) (xjson.Field, error) {
	if kinds := payload.ChannelKinds(dt); !slices.Contains(kinds, wc.Type()) {
		return xjson.Field{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s channel cannot be written as %s, only as %v",
			dt, wc.Type(), kinds,
		)
	}
	f := xjson.Field{DataType: dt, Type: xjson.String}
	var err error
	switch v := wc.Variant.(type) {
	case FloatWriteChannel:
		f.Type = xjson.Number
		f.Enums = lo.Map(v.EnumValues, func(e payload.FloatEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, float64(e.Value))
		})
	case IntegerWriteChannel:
		f.Type = xjson.Number
		f.Enums = lo.Map(v.EnumValues, func(e payload.IntegerEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, int64(e.Value))
		})
	case UnsignedWriteChannel:
		f.Type = xjson.Number
		f.Enums = lo.Map(v.EnumValues, func(e payload.UnsignedEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, uint64(e.Value))
		})
	case StringWriteChannel:
		f.Enums = make([]xjson.Enum, len(v.EnumValues))
		for i, e := range v.EnumValues {
			if f.Enums[i], err = textEnum(e, dt); err != nil {
				return xjson.Field{}, err
			}
		}
	case BooleanWriteChannel:
		f.Type = xjson.Boolean
		f.Enums = lo.Map(v.EnumValues, func(e payload.BooleanEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, e.Value)
		})
	case TimestampWriteChannel:
		if f.TimeFormat, err = xjson.ParseTimeFormat(string(v.TimeFormat)); err != nil {
			return xjson.Field{}, err
		}
	case JsonWriteChannel:
		f.Type = xjson.JSON
	}
	return f, nil
}

// textEnum returns the enum of e, whose value is the text of a sample of dt.
func textEnum(e payload.StringEntry, dt telem.DataType) (xjson.Enum, error) {
	var err error
	switch kind, _ := payload.KindOf(dt); kind {
	case payload.KindFloat:
		var v float64
		if v, err = strconv.ParseFloat(e.Value, 64); err == nil {
			return xjson.NewEnum(e.Label, v), nil
		}
	case payload.KindInteger:
		var v int64
		if v, err = strconv.ParseInt(e.Value, 10, 64); err == nil {
			return xjson.NewEnum(e.Label, v), nil
		}
	case payload.KindUnsigned, payload.KindBoolean:
		var v uint64
		if v, err = strconv.ParseUint(e.Value, 10, 64); err == nil {
			return xjson.NewEnum(e.Label, v), nil
		}
	default:
		return xjson.NewEnum(e.Label, e.Value), nil
	}
	return xjson.Enum{}, errors.Wrapf(
		validate.ErrValidation,
		"enum label %q: %q is not a %s value",
		e.Label, e.Value, dt,
	)
}

// validateWriteConfig checks cfg without reading its Synnax channels. The values of
// each channel's entries must differ, so two NaN entries, or 0 and -0, collide.
func validateWriteConfig(cfg *WriteConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	for _, wc := range cfg.Channels {
		var dup int
		switch v := wc.Variant.(type) {
		case FloatWriteChannel:
			dup = duplicate(v.EnumValues, func(a, b payload.FloatEntry) bool {
				x, y := float64(a.Value), float64(b.Value)
				return x == y || (math.IsNaN(x) && math.IsNaN(y))
			})
		case IntegerWriteChannel:
			dup = duplicate(v.EnumValues, func(a, b payload.IntegerEntry) bool {
				return a.Value == b.Value
			})
		case UnsignedWriteChannel:
			dup = duplicate(v.EnumValues, func(a, b payload.UnsignedEntry) bool {
				return a.Value == b.Value
			})
		case StringWriteChannel:
			dup = duplicate(v.EnumValues, func(a, b payload.StringEntry) bool {
				return a.Value == b.Value
			})
		case BooleanWriteChannel:
			dup = duplicate(v.EnumValues, func(a, b payload.BooleanEntry) bool {
				return a.Value == b.Value
			})
		default:
			dup = -1
		}
		if dup >= 0 {
			return errors.Wrapf(
				validate.ErrValidation,
				"channel %q: enum_values[%d]: value is already labeled",
				wc.Base().Key, dup,
			)
		}
	}
	return nil
}

func duplicate[E any](entries []E, equal func(a, b E) bool) int {
	for i, e := range entries {
		if slices.ContainsFunc(
			entries[:i],
			func(prev E) bool { return equal(prev, e) },
		) {
			return i
		}
	}
	return -1
}

// readChannels returns the channels among keys, read through snap. A key with no
// channel is absent from the result.
func readChannels(
	ctx context.Context,
	snap gorp.Snapshot,
	svc *channel.Service,
	keys channel.Keys,
) (map[channel.Key]channel.Channel, error) {
	var channels []channel.Channel
	if err := svc.NewRetrieve().
		Where(func(channel.Retrieve) gorp.Filter[channel.Key, channel.Channel] {
			return gorp.MatchKeysFunc[channel.Key, channel.Channel](
				func(gorp.Context) ([]channel.Key, error) { return keys, nil },
			)
		}).
		Entries(&channels).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	return lo.KeyBy(channels, channel.Channel.Key), nil
}

// resolveWriteConfig checks each enabled channel of cfg against its Synnax channel,
// read through snap, and narrows float entries to a float32 channel. It skips channel
// 0 and a channel that does not exist.
func resolveWriteConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	svc *channel.Service,
	cfg *WriteConfig,
) error {
	channels, err := readChannels(ctx, snap, svc, lo.FilterMap(
		cfg.Channels,
		func(wc WriteChannel, _ int) (channel.Key, bool) {
			b := wc.Base()
			return b.Channel, !b.Disabled && b.Channel != 0
		},
	))
	if err != nil {
		return err
	}
	for _, wc := range cfg.Channels {
		b := wc.Base()
		ch, ok := channels[b.Channel]
		if b.Disabled || !ok {
			continue
		}
		if err = checkWriteChannel(wc, ch.DataType); err != nil {
			return errors.Wrapf(err, "channel %q", b.Key)
		}
		if v, ok := wc.Variant.(FloatWriteChannel); ok &&
			ch.DataType == telem.Float32T {
			for j, e := range v.EnumValues {
				v.EnumValues[j].Value = payload.Float(float32(e.Value))
			}
		}
	}
	return nil
}

func checkWriteChannel(wc WriteChannel, dt telem.DataType) error {
	f, err := fieldOf(wc, dt)
	if err != nil {
		return err
	}
	if _, err = xjson.NewEncoder(f); err != nil {
		return err
	}
	return xjson.CheckEnums(f)
}

// reconcileWriteConfig changes each channel of cfg to agree with its Synnax channel,
// read through tx, and logs each change. A kind the channel's data type does not allow
// becomes the first numeric kind it allows when the kind is numeric or timestamp, and
// the data type's own kind otherwise. Entries convert between numeric kinds when they
// stay exact, and drop on other kinds. An entry that does not fit the channel, or whose
// value collides with an earlier one's, drops.
func reconcileWriteConfig(
	ctx context.Context,
	tx gorp.Tx,
	svc *channel.Service,
	ins alamos.Instrumentation,
	cfg *WriteConfig,
) error {
	channels, err := readChannels(ctx, tx, svc, lo.FilterMap(
		cfg.Channels,
		func(wc WriteChannel, _ int) (channel.Key, bool) {
			return wc.Base().Channel, wc.Base().Channel != 0
		},
	))
	if err != nil {
		return err
	}
	for i, wc := range cfg.Channels {
		ch, ok := channels[wc.Base().Channel]
		if !ok {
			continue
		}
		fields := []zap.Field{
			zap.Stringer("config", cfg.Key),
			zap.String("field", wc.Base().Key),
			zap.Stringer("channel", ch.Key()),
			zap.String("channel_name", ch.Name),
			zap.String("data_type", string(ch.DataType)),
		}
		warn := func(msg string, extra ...zap.Field) {
			ins.L.Warn(msg, append(slices.Clip(fields), extra...)...)
		}
		cfg.Channels[i] = reconcileWriteChannel(wc, ch.DataType, warn)
	}
	return nil
}

// newWriteMigration returns the migration that lifts stored write configs to v1 and
// reconciles their channels against the channels in svc.
func newWriteMigration(svc *channel.Service) gorp.Migration {
	return migrate.New(
		"v62_kafka_typed_write_channels",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v0.WriteConfig
			if err := gorp.NewRetrieve[uuid.UUID, v0.WriteConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil {
				return err
			}
			news := make([]WriteConfig, 0, len(olds))
			for _, old := range olds {
				cfg, err := v1.MigrateWriteConfig(ctx, old)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				if err = reconcileWriteConfig(ctx, tx, svc, ins, &cfg); err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, WriteConfig](tx).Set(ctx, news...)
		},
	)
}

type warnFunc func(msg string, fields ...zap.Field)

func reconcileWriteChannel(
	wc WriteChannel,
	dt telem.DataType,
	warn warnFunc,
) WriteChannel {
	kinds := payload.ChannelKinds(dt)
	if len(kinds) == 0 {
		return wc
	}
	if !slices.Contains(kinds, wc.Type()) {
		kind := kinds[0]
		// A v0 row that kept a time format after it moved off a timestamp channel
		// wrote numbers there.
		if numeric(wc.Type()) || wc.Type() == payload.KindTimestamp {
			if i := slices.IndexFunc(kinds, numeric); i >= 0 {
				kind = kinds[i]
			}
		}
		warn(
			"changed the kind of a Kafka write channel to match its channel",
			zap.String("old", string(wc.Type())),
			zap.String("new", string(kind)),
		)
		wc = convertWriteChannel(wc, kind, warn)
	}
	return dropEntries(wc, dt, warn)
}

func numeric(k payload.Kind) bool {
	return k == payload.KindFloat || k == payload.KindInteger ||
		k == payload.KindUnsigned
}

// convertWriteChannel returns wc as kind. Entries convert between numeric kinds when
// they stay exact, and drop otherwise.
func convertWriteChannel(
	wc WriteChannel,
	kind payload.Kind,
	warn warnFunc,
) WriteChannel {
	from := numericValues(wc)
	drop := func(label string, value any) {
		warn(
			"dropped a Kafka write channel enum entry that does not convert",
			zap.String("label", label),
			zap.Any("value", value),
		)
	}
	out := WriteChannel{}
	switch kind {
	case payload.KindFloat:
		var v FloatWriteChannel
		for _, e := range from {
			if f, ok := toFloat64(e.value); ok {
				v.EnumValues = append(
					v.EnumValues,
					payload.FloatEntry{Label: e.label, Value: payload.Float(f)},
				)
			} else {
				drop(e.label, e.value)
			}
		}
		out.Variant = v
	case payload.KindInteger:
		var v IntegerWriteChannel
		for _, e := range from {
			if i, ok := toInt64(e.value); ok {
				v.EnumValues = append(
					v.EnumValues,
					payload.IntegerEntry{Label: e.label, Value: payload.Integer(i)},
				)
			} else {
				drop(e.label, e.value)
			}
		}
		out.Variant = v
	case payload.KindUnsigned:
		var v UnsignedWriteChannel
		for _, e := range from {
			if u, ok := toUint64(e.value); ok {
				v.EnumValues = append(
					v.EnumValues,
					payload.UnsignedEntry{Label: e.label, Value: payload.Unsigned(u)},
				)
			} else {
				drop(e.label, e.value)
			}
		}
		out.Variant = v
	default:
		if from != nil || hasTextEntries(wc) {
			warn("dropped the enum entries of a Kafka write channel")
		}
		out.Variant = emptyVariants[kind]
	}
	out.SetBase(wc.Base())
	return out
}

// emptyVariants holds the variant of each kind with no numeric entries.
var emptyVariants = map[payload.Kind]WriteChannelVariant{
	payload.KindString:    StringWriteChannel{},
	payload.KindBoolean:   BooleanWriteChannel{},
	payload.KindTimestamp: TimestampWriteChannel{TimeFormat: payload.TimeFormatISO8601},
	payload.KindJSON:      JsonWriteChannel{},
	payload.KindUUID:      UuidWriteChannel{},
	payload.KindBytes:     BytesWriteChannel{},
}

type numericValue struct {
	label string
	// value is a float64, int64, or uint64.
	value any
}

func numericValues(wc WriteChannel) []numericValue {
	var out []numericValue
	switch v := wc.Variant.(type) {
	case FloatWriteChannel:
		for _, e := range v.EnumValues {
			out = append(out, numericValue{label: e.Label, value: float64(e.Value)})
		}
	case IntegerWriteChannel:
		for _, e := range v.EnumValues {
			out = append(out, numericValue{label: e.Label, value: int64(e.Value)})
		}
	case UnsignedWriteChannel:
		for _, e := range v.EnumValues {
			out = append(out, numericValue{label: e.Label, value: uint64(e.Value)})
		}
	}
	return out
}

func hasTextEntries(wc WriteChannel) bool {
	switch v := wc.Variant.(type) {
	case StringWriteChannel:
		return len(v.EnumValues) > 0
	case BooleanWriteChannel:
		return len(v.EnumValues) > 0
	}
	return false
}

// maxExactFloat is the largest magnitude up to which a float64 holds every integer.
const maxExactFloat = 1 << 53

func toFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), x >= -maxExactFloat && x <= maxExactFloat
	case uint64:
		return float64(x), x <= maxExactFloat
	default:
		return v.(float64), true
	}
}

func toInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case float64:
		ok := x == math.Trunc(x) && x >= math.MinInt64 && x < math.MaxInt64
		return int64(x), ok
	case uint64:
		return int64(x), x <= math.MaxInt64
	default:
		return v.(int64), true
	}
}

func toUint64(v any) (uint64, bool) {
	switch x := v.(type) {
	case float64:
		return uint64(x), x == math.Trunc(x) && x >= 0 && x < math.MaxUint64
	case int64:
		return uint64(x), x >= 0
	default:
		return v.(uint64), true
	}
}

// dropEntries drops each entry of wc that does not fit a channel of data type dt, or
// whose value matches an earlier entry's there, and logs it.
func dropEntries(wc WriteChannel, dt telem.DataType, warn warnFunc) WriteChannel {
	switch v := wc.Variant.(type) {
	case FloatWriteChannel:
		v.EnumValues = keepEntries(v.EnumValues, dt, warn,
			func(e payload.FloatEntry) string { return e.Label },
			func(es []payload.FloatEntry) WriteChannelVariant {
				v.EnumValues = es
				return v
			})
		wc.Variant = v
	case IntegerWriteChannel:
		v.EnumValues = keepEntries(v.EnumValues, dt, warn,
			func(e payload.IntegerEntry) string { return e.Label },
			func(es []payload.IntegerEntry) WriteChannelVariant {
				v.EnumValues = es
				return v
			})
		wc.Variant = v
	case UnsignedWriteChannel:
		v.EnumValues = keepEntries(v.EnumValues, dt, warn,
			func(e payload.UnsignedEntry) string { return e.Label },
			func(es []payload.UnsignedEntry) WriteChannelVariant {
				v.EnumValues = es
				return v
			})
		wc.Variant = v
	case StringWriteChannel:
		v.EnumValues = keepEntries(v.EnumValues, dt, warn,
			func(e payload.StringEntry) string { return e.Label },
			func(es []payload.StringEntry) WriteChannelVariant {
				v.EnumValues = es
				return v
			})
		wc.Variant = v
	case BooleanWriteChannel:
		v.EnumValues = keepEntries(v.EnumValues, dt, warn,
			func(e payload.BooleanEntry) string { return e.Label },
			func(es []payload.BooleanEntry) WriteChannelVariant {
				v.EnumValues = es
				return v
			})
		wc.Variant = v
	}
	return wc
}

func keepEntries[E any](
	entries []E,
	dt telem.DataType,
	warn warnFunc,
	label func(E) string,
	with func([]E) WriteChannelVariant,
) []E {
	kept := make([]E, 0, len(entries))
	for _, e := range entries {
		candidate := WriteChannel{Variant: with(append(slices.Clip(kept), e))}
		if err := checkWriteChannel(candidate, dt); err != nil {
			warn(
				"dropped a Kafka write channel enum entry",
				zap.String("label", label(e)),
				zap.Error(err),
			)
			continue
		}
		kept = append(kept, e)
	}
	return kept
}
