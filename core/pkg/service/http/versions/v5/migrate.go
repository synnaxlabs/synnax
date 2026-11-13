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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	xpayload "github.com/synnaxlabs/x/payload"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	payloadv1 "github.com/synnaxlabs/x/payload/versions/v1"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// MigrateWriteConfig lifts a v4 HTTP write task config to v5.
func MigrateWriteConfig(ctx context.Context, old v4.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

// MigrateWriteEndpoint lifts a v4 write endpoint to v5. Its channel field lifts as
// MigrateChannelField does.
func MigrateWriteEndpoint(
	ctx context.Context,
	old v4.WriteEndpoint,
) (WriteEndpoint, error) {
	ep, err := autoMigrateWriteEndpoint(ctx, old)
	if err != nil {
		return WriteEndpoint{}, err
	}
	ep.Channel = MigrateChannelField(old.Channel)
	return ep, nil
}

// MigrateChannelField lifts a v4 channel field to v5 by its stored shape, without its
// Synnax channel. The v4 Driver wrote a timestamp channel in its time format whatever
// the JSON type, so a field with a channel and no entries whose data type is a
// timestamp, or that has a time format, becomes a timestamp field in that format, ISO
// 8601 when it has none. Its data type then names the kind its JSON type writes, for
// the caller to fall back to when the channel holds no timestamps. A field with no
// channel and a timestamp data type becomes a timestamp field the same way. Otherwise,
// a boolean JSON type makes a boolean field, and a number a float field that keeps its
// entries. A string makes a string field whose entries take the decimal text of their
// values, so a numeric sample writes the same JSON string as in v4. An entry whose
// value repeats an earlier one's drops, because v4 wrote the first label. The caller
// reconciles the field against its channel, or against its data type when it has none.
func MigrateChannelField(old v4.ChannelField) ChannelField {
	jsonType := lo.CoalesceOrEmpty(old.JSONType, payload.JSONTypeNumber)
	stored := lo.CoalesceOrEmpty(old.DataType, telem.Float64T)
	dt := stored
	base := BaseChannelField{
		Pointer:  old.Pointer,
		Channel:  old.Channel,
		Name:     old.Name,
		DataType: &dt,
	}
	entries := lo.UniqBy(old.EnumValues, func(e payload.EnumEntry) string {
		return valueText(e.Value)
	})
	var c xpayload.Channel
	switch {
	case old.Channel != 0 && len(old.EnumValues) == 0 &&
		(stored == telem.TimestampT || old.TimeFormat != nil):
		dt = fallbackDataTypes[jsonType]
		if jsonType == payload.JSONTypeNumber {
			dt = lo.Ternary(stored == telem.TimestampT, telem.Float64T, stored)
		}
		c = xpayload.Channel{
			Kind:       xpayload.KindTimestamp,
			TimeFormat: lo.FromPtrOr(old.TimeFormat, payload.TimeFormatISO8601),
		}
	case old.Channel == 0 && stored == telem.TimestampT:
		c = xpayload.Channel{
			Kind:       xpayload.KindTimestamp,
			TimeFormat: lo.FromPtrOr(old.TimeFormat, payload.TimeFormatISO8601),
		}
	case jsonType == payload.JSONTypeBoolean:
		c = xpayload.Channel{Kind: xpayload.KindBoolean}
	case jsonType == payload.JSONTypeString:
		c = xpayload.Channel{
			Kind: xpayload.KindString,
			String: lo.Map(
				entries,
				func(e payload.EnumEntry, _ int) payload.StringEntry {
					return payload.StringEntry{
						Label: e.Label,
						Value: valueText(e.Value),
					}
				},
			),
		}
	default:
		c = xpayload.Channel{
			Kind: xpayload.KindFloat,
			Float: lo.Map(
				entries,
				func(e payload.EnumEntry, _ int) payload.FloatEntry {
					return payload.FloatEntry{
						Label: e.Label,
						Value: payload.Float(e.Value),
					}
				},
			),
		}
	}
	return newChannelField(base, c)
}

func valueText(v float64) string {
	// -0 would format as "-0", which no unsigned channel parses.
	if v == 0 {
		v = 0
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// fallbackDataTypes holds the data type whose kind writes each v4 JSON type.
var fallbackDataTypes = map[payload.JSONType]telem.DataType{
	payload.JSONTypeString:  telem.StringT,
	payload.JSONTypeBoolean: telem.BooleanT,
}

// MigrateWriteField lifts a v4 write field to a typed write field, as
// payloadv1.LiftStaticWriteField and payloadv1.LiftGeneratedWriteField do. ok is false
// for a field with an unknown generator, which no Driver ran, and for a null field.
func MigrateWriteField(old v4.WriteField) (f payloadv1.WriteField, ok bool, err error) {
	switch v := old.Variant.(type) {
	case nil:
		return payloadv1.WriteField{}, false, nil
	case v4.StaticWriteField:
		f, err = payloadv1.LiftStaticWriteField(
			payloadv1.BaseWriteField(v.BaseWriteField),
			v.Value,
		)
		return f, err == nil, err
	case v4.GeneratedWriteField:
		f, ok = payloadv1.LiftGeneratedWriteField(
			payloadv1.BaseWriteField(v.BaseWriteField),
			v.Generator,
			v.TimeFormat,
		)
		return f, ok, nil
	default:
		return payloadv1.WriteField{}, false, errors.Newf(
			"unknown WriteField variant %T",
			v,
		)
	}
}

func migrateWriteEndpointFields(
	_ context.Context,
	old []v4.WriteField,
) ([]payloadv1.WriteField, error) {
	var fields []payloadv1.WriteField
	for _, o := range old {
		f, ok, err := MigrateWriteField(o)
		if err != nil {
			return nil, err
		}
		if ok {
			fields = append(fields, f)
		}
	}
	return fields, nil
}

// Warn reports a change that a migration or a reconcile makes to a field.
type Warn func(msg string, fields ...zap.Field)

func ignore(string, ...zap.Field) {}

// MigrateReadConfig lifts a v4 HTTP read task config to v5, as MigrateReadEndpoint does
// for each endpoint.
func MigrateReadConfig(ctx context.Context, old v4.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// MigrateReadEndpoint lifts a v4 read endpoint to v5, as MigrateReadField does for each
// field. The field that indexes the endpoint is a timestamp field whenever it has a
// time format, because the v4 Console marked its timing field that way.
func MigrateReadEndpoint(_ context.Context, old v4.ReadEndpoint) (ReadEndpoint, error) {
	return liftReadEndpoint(old, ignore), nil
}

// MigrateReadField lifts a v4 read field to v5. The stored data type picks the kind. A
// field with a time format that is not optional and has no enum entries is a timestamp
// field, because v4 used a time format only on a timestamp channel and defaulted the
// data type to float64. A float, integer, or unsigned field keeps the data type for a
// new channel. An enum entry converts when the kind holds its value, and a boolean
// entry is true when its value is not 0. A string or timestamp field drops its entries,
// and a timestamp field with no time format takes ISO 8601. Of entries that repeat a
// label, the first stays.
//
// An optional field's fill converts to the stored data type, and an enum label converts
// to the value of its entry. An optional float field with no fill takes NaN. A fill
// that does not convert, a fill on a field that is not optional, and an optional field
// with no fill on any other kind drop, because the Driver rejected each of them. An
// unknown data type reads as float64.
func MigrateReadField(old v4.ReadField) ReadField {
	return liftReadField(old, false, ignore)
}

// NewReadMigration returns the migration that lifts stored v4 read configs to v5,
// reconciles them with channels, and logs each lossy change.
func NewReadMigration(channels Channels) gorp.Migration {
	return migrate.New(
		"v62_typed_read_field",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v4.ReadConfig
			if err := gorp.NewRetrieve[uuid.UUID, v4.ReadConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil {
				return err
			}
			news := make([]ReadConfig, 0, len(olds))
			for _, old := range olds {
				warn := func(msg string, fields ...zap.Field) {
					ins.L.Warn(msg, append(fields, zap.Stringer("config", old.Key))...)
				}
				cfg := liftReadConfig(old, warn)
				if err := ReconcileReadConfig(
					ctx,
					tx,
					channels,
					&cfg,
					warn,
				); err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, news...)
		},
	)
}

func liftReadConfig(old v4.ReadConfig, warn Warn) ReadConfig {
	return ReadConfig{
		PersistConfig: old.PersistConfig,
		Device:        old.Device,
		Rate:          old.Rate,
		Endpoints: lo.Map(old.Endpoints, func(ep v4.ReadEndpoint, _ int) ReadEndpoint {
			return liftReadEndpoint(ep, warn)
		}),
		NanFillEnabled: old.NanFillEnabled,
	}
}

func liftReadEndpoint(old v4.ReadEndpoint, warn Warn) ReadEndpoint {
	return ReadEndpoint{
		Key:         old.Key,
		Method:      old.Method,
		Path:        old.Path,
		Headers:     old.Headers,
		QueryParams: old.QueryParams,
		Body:        old.Body,
		Fields: lo.Map(old.Fields, func(f v4.ReadField, _ int) ReadField {
			return liftReadField(f, f.Key == old.Index, warn)
		}),
		Index: old.Index,
	}
}

func liftReadField(old v4.ReadField, index bool, warn Warn) ReadField {
	warn = withField(warn, old.Key)
	if old.TimeFormat != nil &&
		(index || (!old.Optional && len(old.EnumValues) == 0)) {
		old.DataType = telem.TimestampT
	}
	kind, ok := xpayload.KindOf(old.DataType)
	if !ok {
		warn(
			"read an unknown data type as float64",
			zap.String("data_type", string(old.DataType)),
		)
		old.DataType = telem.Float64T
		kind = payload.KindFloat
	}
	o := options{fill: liftFill(old, kind, warn), timeFormat: payload.TimeFormatISO8601}
	if old.TimeFormat != nil {
		o.timeFormat = *old.TimeFormat
	}
	if numeric(kind) {
		o.dataType = new(old.DataType)
	}
	for _, e := range uniqueLabels(old.EnumValues, warn) {
		var value any = payload.Float(e.Value)
		if kind == payload.KindBoolean {
			value = e.Value != 0
		}
		o.entries = append(o.entries, entry{label: e.Label, value: value})
	}
	return assemble(kind, BaseReadField{
		Key:      old.Key,
		Name:     old.Name,
		Disabled: old.Disabled,
		Channel:  old.Channel,
		Pointer:  old.Pointer,
	}, o, nil, warn)
}

func uniqueLabels(entries []payload.EnumEntry, warn Warn) []payload.EnumEntry {
	seen := make(set.Set[string], len(entries))
	return lo.Filter(entries, func(e payload.EnumEntry, _ int) bool {
		if seen.Contains(e.Label) {
			warn(
				"dropped an enum entry that repeats a label",
				zap.String("label", e.Label),
			)
			return false
		}
		seen.Add(e.Label)
		return true
	})
}

// liftFill returns the fill of old as a sample of its kind, or nil when it has none.
func liftFill(old v4.ReadField, kind payload.Kind, warn Warn) any {
	if !old.Optional {
		if old.FillValue != nil {
			warn("dropped the fill of a field that is not optional")
		}
		return nil
	}
	if old.FillValue == nil {
		if kind == payload.KindFloat {
			return payload.Float(math.NaN())
		}
		warn("dropped the optional flag of a field with no fill on a non-float channel")
		return nil
	}
	raw := *old.FillValue
	if label, ok := raw.(string); ok && (numeric(kind) || kind == payload.KindBoolean) {
		e, found := lo.Find(old.EnumValues, func(e payload.EnumEntry) bool {
			return e.Label == label
		})
		if !found {
			warn(
				"dropped a fill label that no enum entry has",
				zap.String("label", label),
			)
			return nil
		}
		raw = e.Value
	}
	sample, err := decodeFill(raw, old)
	if err != nil {
		warn("dropped a fill that does not convert", zap.Error(err))
		return nil
	}
	return sample
}

func decodeFill(raw any, old v4.ReadField) (any, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	f := xjson.Field{DataType: old.DataType}
	if old.TimeFormat != nil {
		if f.TimeFormat, err = xjson.ParseTimeFormat(
			string(*old.TimeFormat),
		); err != nil {
			return nil, err
		}
	}
	dec, err := xjson.NewDecoder(f)
	if err != nil {
		return nil, err
	}
	data, err := dec.Append(nil, jsontext.Value(b))
	if err != nil {
		return nil, err
	}
	s := telem.Series{DataType: old.DataType, Data: data}
	switch old.DataType {
	case telem.Float64T:
		return payload.Float(s.ValueAt[float64](0)), nil
	case telem.Float32T:
		return payload.Float(s.ValueAt[float32](0)), nil
	case telem.Int8T:
		return payload.Integer(s.ValueAt[int8](0)), nil
	case telem.Int16T:
		return payload.Integer(s.ValueAt[int16](0)), nil
	case telem.Int32T:
		return payload.Integer(s.ValueAt[int32](0)), nil
	case telem.Int64T:
		return payload.Integer(s.ValueAt[int64](0)), nil
	case telem.Uint8T:
		return payload.Unsigned(s.ValueAt[uint8](0)), nil
	case telem.Uint16T:
		return payload.Unsigned(s.ValueAt[uint16](0)), nil
	case telem.Uint32T:
		return payload.Unsigned(s.ValueAt[uint32](0)), nil
	case telem.Uint64T:
		return payload.Unsigned(s.ValueAt[uint64](0)), nil
	case telem.BooleanT:
		return s.ValueAt[uint8](0) != 0, nil
	case telem.TimestampT:
		return s.ValueAt[telem.TimeStamp](0), nil
	case telem.StringT:
		text, _ := telem.UnmarshalVariableSample(data)
		return string(text), nil
	}
	return nil, errors.Newf("a %s channel takes no fill", old.DataType)
}
