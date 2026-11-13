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
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	xpayload "github.com/synnaxlabs/x/payload"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// Warn reports a lossy change to a read field.
type Warn func(msg string, fields ...zap.Field)

func ignore(string, ...zap.Field) {}

// MigrateReadConfig lifts a v4 HTTP read task config to v5, as MigrateReadEndpoint does
// for each endpoint.
func MigrateReadConfig(ctx context.Context, old v4.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// MigrateReadEndpoint lifts a v4 read endpoint to v5, as MigrateReadField does for each
// field. The field that indexes the endpoint is a timestamp field.
func MigrateReadEndpoint(_ context.Context, old v4.ReadEndpoint) (ReadEndpoint, error) {
	return liftReadEndpoint(old, nil, ignore), nil
}

// MigrateReadField lifts a v4 read field to v5. The stored data type picks the kind. A
// field with a time format and the float64 default or an unknown data type is a
// timestamp field, because v4 used a time format only on a timestamp channel. On any
// other data type the time format is left over, and the field keeps its kind. A float,
// integer, or unsigned field keeps the data type for a new channel. An enum entry
// converts when the kind holds its value, and a boolean entry is true when its value is
// not 0. A string or timestamp field drops its entries, and a timestamp field with no
// time format takes ISO 8601. Of entries that repeat a label, the first stays, and an
// empty label drops on a kind other than string.
//
// An optional field's fill converts to the kind as the v4 Driver read it: an enum label
// converts to the value of its entry, other text on a numeric field reads as a number,
// and a fill that is not text on a string field reads as its JSON text. An optional
// float field with no fill takes NaN. A fill that does not convert, a fill on a field
// that is not optional, and an optional field with no fill on any other kind drop,
// because the Driver rejected each of them. An unknown data type reads as float64.
func MigrateReadField(old v4.ReadField) ReadField {
	return liftReadField(old, false, nil, ignore)
}

// NewReadMigration returns the migration that lifts stored v4 read configs to v5,
// reconciles them with channels, and logs each lossy change. A field whose channel
// exists lifts by the channel's data type, as the v4 Driver read it.
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
				found, err := channels(ctx, tx, channelsOf(old))
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				cfg := liftReadConfig(old, found, warn)
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

func channelsOf(cfg v4.ReadConfig) []channel.Key {
	var keys []channel.Key
	for _, ep := range cfg.Endpoints {
		for _, f := range ep.Fields {
			if f.Channel != 0 {
				keys = append(keys, f.Channel)
			}
		}
	}
	return lo.Uniq(keys)
}

func liftReadConfig(
	old v4.ReadConfig,
	found map[channel.Key]Channel,
	warn Warn,
) ReadConfig {
	return ReadConfig{
		PersistConfig: old.PersistConfig,
		Device:        old.Device,
		Rate:          old.Rate,
		Endpoints: lo.Map(old.Endpoints, func(ep v4.ReadEndpoint, _ int) ReadEndpoint {
			return liftReadEndpoint(ep, found, warn)
		}),
		NanFillEnabled: old.NanFillEnabled,
	}
}

func liftReadEndpoint(
	old v4.ReadEndpoint,
	found map[channel.Key]Channel,
	warn Warn,
) ReadEndpoint {
	return ReadEndpoint{
		Key:         old.Key,
		Method:      old.Method,
		Path:        old.Path,
		Headers:     old.Headers,
		QueryParams: old.QueryParams,
		Body:        old.Body,
		Fields: lo.Map(old.Fields, func(f v4.ReadField, _ int) ReadField {
			var channelType *telem.DataType
			if ch, ok := found[f.Channel]; ok {
				channelType = &ch.DataType
			}
			return liftReadField(f, f.Key == old.Index, channelType, warn)
		}),
		Index: old.Index,
	}
}

func liftReadField(
	old v4.ReadField,
	index bool,
	channelType *telem.DataType,
	warn Warn,
) ReadField {
	warn = withField(warn, old.Key)
	switch {
	case channelType != nil:
		old.DataType = *channelType
	case index:
		old.DataType = telem.TimestampT
	case old.TimeFormat != nil && defaultDataType(old.DataType):
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
	o.entries = withoutEmptyLabels(o.entries, kind, warn)
	return assemble(kind, BaseReadField{
		Key:      old.Key,
		Name:     old.Name,
		Disabled: old.Disabled,
		Channel:  old.Channel,
		Pointer:  old.Pointer,
	}, o, nil, warn)
}

// defaultDataType returns true when dt is the v4 default of float64 or not a known data
// type, so it says nothing about the field's channel.
func defaultDataType(dt telem.DataType) bool {
	_, known := xpayload.KindOf(dt)
	return dt == telem.Float64T || !known
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
	if kind == payload.KindString {
		if _, ok := raw.(string); !ok {
			b, err := json.Marshal(raw)
			if err != nil {
				warn("dropped a fill that does not convert", zap.Error(err))
				return nil
			}
			return string(b)
		}
	}
	if label, ok := raw.(string); ok && (numeric(kind) || kind == payload.KindBoolean) {
		e, found := lo.Find(old.EnumValues, func(e payload.EnumEntry) bool {
			return e.Label == label
		})
		switch {
		case found:
			raw = e.Value
		case numeric(kind):
			text := strings.TrimSpace(label)
			if f, err := strconv.ParseFloat(text, 64); err == nil &&
				kind == payload.KindFloat && (math.IsNaN(f) || math.IsInf(f, 0)) {
				return payload.Float(f)
			}
			raw = jsontext.Value(text)
		default:
			warn(
				"dropped a fill label that no enum entry has",
				zap.String("label", label),
			)
			return nil
		}
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
