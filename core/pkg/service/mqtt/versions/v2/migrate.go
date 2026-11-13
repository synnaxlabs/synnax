// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"fmt"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	"github.com/synnaxlabs/x/errors"
	xpayload "github.com/synnaxlabs/x/payload"
	payloadv0 "github.com/synnaxlabs/x/payload/versions/v0"
	payload "github.com/synnaxlabs/x/payload/versions/v1"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// MigrateChannelField lifts a v1 channel field to v2 by its stored shape, without its
// Synnax channel. A timestamp data type makes a timestamp field in its time format, ISO
// 8601 when it has none. v1 encoded a timestamp channel in its time format whatever
// the shape, so a field with a channel, a time format, and no entries also becomes a
// timestamp field, and its data type names the kind its JSON type writes, for the
// caller to fall back to when the channel does not hold timestamps. Otherwise, a
// boolean JSON type makes a boolean field, a string with no entries a string field,
// and a number, or a string with entries, a float field that keeps the entries. The
// caller reconciles the field against its channel, or against its data type when it
// has none.
func MigrateChannelField(_ context.Context, old v1.ChannelField) (ChannelField, error) {
	old.ApplyDefaults()
	dt := old.DataType
	base := BaseChannelField{
		Pointer:  old.Pointer,
		Channel:  old.Channel,
		Name:     old.Name,
		DataType: &dt,
	}
	var c xpayload.Channel
	switch {
	case dt == telem.TimestampT:
		c = xpayload.Channel{
			Kind:       xpayload.KindTimestamp,
			TimeFormat: lo.FromPtrOr(old.TimeFormat, payload.TimeFormatISO8601),
		}
	case old.TimeFormat != nil && old.Channel != 0 && len(old.EnumValues) == 0:
		dt = fallbackDataTypes[old.JSONType]
		if old.JSONType == payloadv0.JSONTypeNumber {
			dt = old.DataType
		}
		c = xpayload.Channel{Kind: xpayload.KindTimestamp, TimeFormat: *old.TimeFormat}
	case old.JSONType == payloadv0.JSONTypeBoolean:
		c = xpayload.Channel{Kind: xpayload.KindBoolean}
	case old.JSONType == payloadv0.JSONTypeString && len(old.EnumValues) == 0:
		c = xpayload.Channel{Kind: xpayload.KindString}
	default:
		c = xpayload.Channel{
			Kind: xpayload.KindFloat,
			Float: lo.Map(
				old.EnumValues,
				func(e payloadv0.EnumEntry, _ int) payload.FloatEntry {
					return payload.FloatEntry{
						Label: e.Label,
						Value: payload.Float(e.Value),
					}
				},
			),
		}
	}
	return channelField(base, c), nil
}

// fallbackDataTypes holds the data type whose kind writes each v1 JSON type.
var fallbackDataTypes = map[payloadv0.JSONType]telem.DataType{
	payloadv0.JSONTypeString:  telem.StringT,
	payloadv0.JSONTypeBoolean: telem.BooleanT,
}

// channelField returns the channel field of base with the kind and the options of c.
// It is a copy of the live conversion, so that this version stays fixed when the live
// types change.
func channelField(base BaseChannelField, c xpayload.Channel) ChannelField {
	var v ChannelFieldVariant
	switch c.Kind {
	case xpayload.KindFloat:
		v = FloatChannelField{BaseChannelField: base, EnumValues: c.Float}
	case xpayload.KindInteger:
		v = IntegerChannelField{BaseChannelField: base, EnumValues: c.Integer}
	case xpayload.KindUnsigned:
		v = UnsignedChannelField{BaseChannelField: base, EnumValues: c.Unsigned}
	case xpayload.KindString:
		v = StringChannelField{BaseChannelField: base, EnumValues: c.String}
	case xpayload.KindBoolean:
		v = BooleanChannelField{BaseChannelField: base, EnumValues: c.Boolean}
	case xpayload.KindTimestamp:
		v = TimestampChannelField{BaseChannelField: base, TimeFormat: c.TimeFormat}
	case xpayload.KindJSON:
		v = JsonChannelField{BaseChannelField: base}
	case xpayload.KindUUID:
		v = UuidChannelField{BaseChannelField: base}
	case xpayload.KindBytes:
		v = BytesChannelField{BaseChannelField: base}
	default:
		panic("mqtt: unknown channel field kind " + string(c.Kind))
	}
	return ChannelField{Variant: v}
}

// MigrateWriteTarget lifts a v1 write target to v2. A plain target's channel field
// lifts as MigrateChannelField does, and its additional fields as
// migratePlainWriteTargetFields does. A Sparkplug B target keeps its fields.
func MigrateWriteTarget(ctx context.Context, old v1.WriteTarget) (WriteTarget, error) {
	switch t := old.Variant.(type) {
	case v1.PlainWriteTarget:
		ch, err := MigrateChannelField(ctx, t.Channel)
		if err != nil {
			return WriteTarget{}, err
		}
		fields, err := migratePlainWriteTargetFields(ctx, t.Fields)
		if err != nil {
			return WriteTarget{}, err
		}
		return WriteTarget{Variant: PlainWriteTarget{
			BaseWriteTarget: t.BaseWriteTarget,
			Topic:           t.Topic,
			Qos:             t.Qos,
			Retained:        t.Retained,
			Channel:         ch,
			Fields:          fields,
		}}, nil
	case v1.SparkplugWriteTarget:
		return WriteTarget{Variant: SparkplugWriteTarget{
			BaseWriteTarget: t.BaseWriteTarget,
			Group:           t.Group,
			EdgeNode:        t.EdgeNode,
			Device:          t.Device,
			Tag:             t.Tag,
			TagType:         t.TagType,
			Channel:         t.Channel,
			Name:            t.Name,
		}}, nil
	}
	return WriteTarget{}, nil
}

// MigrateWriteConfig lifts a v1 write config and each of its targets to v2.
func MigrateWriteConfig(ctx context.Context, old v1.WriteConfig) (WriteConfig, error) {
	cfg, err := autoMigrateWriteConfig(ctx, old)
	if err != nil {
		return WriteConfig{}, err
	}
	targets, err := lo.MapErr(
		old.Targets,
		func(t v1.WriteTarget, _ int) (WriteTarget, error) {
			return MigrateWriteTarget(ctx, t)
		},
	)
	if err != nil {
		return WriteConfig{}, err
	}
	cfg.Targets = targets
	return cfg, nil
}

// migratePlainWriteTargetFields lifts the additional fields of a plain target of a
// released Core to typed write fields, as payload.LiftStaticWriteField and
// payload.LiftGeneratedWriteField do. A null field and a field with an unknown
// generator, which no Driver ran, drop.
func migratePlainWriteTargetFields(
	_ context.Context,
	old []v1.WriteField,
) ([]payload.WriteField, error) {
	var fields []payload.WriteField
	for _, o := range old {
		switch v := o.Variant.(type) {
		case v1.StaticWriteField:
			f, err := payload.LiftStaticWriteField(
				payload.BaseWriteField(v.BaseWriteField),
				v.Value,
			)
			if err != nil {
				return nil, err
			}
			fields = append(fields, f)
		case v1.GeneratedWriteField:
			if f, ok := payload.LiftGeneratedWriteField(
				payload.BaseWriteField(v.BaseWriteField),
				v.Generator,
				v.TimeFormat,
			); ok {
				fields = append(fields, f)
			}
		}
	}
	return fields, nil

}

// MigrateReadField lifts a v1 read field to v2, and describes each entry it drops. The
// stored data type picks the kind, and a numeric kind keeps it as the data type of a
// new channel. A float entry converts to an integer or unsigned kind when the kind
// holds it exactly. A boolean entry becomes true when its value is not 0. A string
// field drops its entries, and a timestamp field with no time format takes ISO 8601.
// Each kind but string drops an entry with an empty label, which it can never match.
// An empty data type is the v1 default, float64. An unknown one lifts to the float
// kind with no data type, so that reconciling with the channel can set the kind. A
// float64 field with a time format and no entries lifts to the timestamp kind and keeps
// the format, because v1 left the data type at its default and read the channel's
// data type. A field with entries stays numeric, and its time format drops.
func MigrateReadField(old v1.ReadField) (ReadField, []string) {
	return migrateReadField(old, nil)
}

// migrateReadField is MigrateReadField with the field's channel, when known. The
// channel's data type then picks the kind, as v1 read it, and a numeric kind takes no
// data type, because the channel exists.
func migrateReadField(old v1.ReadField, ch *Channel) (ReadField, []string) {
	var changes []string
	note := func(format string, args ...any) {
		changes = append(changes, fmt.Sprintf(format, args...))
	}
	dt := telem.DataType(old.DataType)
	if dt == "" {
		dt = telem.Float64T
	}
	if ch != nil && ch.DataType != dt {
		note("data type %s became %s, the data type of its channel", dt, ch.DataType)
		dt = ch.DataType
	}
	kind, known := xpayload.KindOf(dt)
	if !known {
		kind = xpayload.KindFloat
		note("unknown data type %q became the float kind", dt)
	}
	switch {
	case ch == nil && old.TimeFormat != nil && dt == telem.Float64T &&
		len(old.EnumValues) == 0:
		kind = xpayload.KindTimestamp
		note("data type %s became the timestamp kind, which its time format reads", dt)
	case old.TimeFormat != nil && kind != xpayload.KindTimestamp:
		note(
			"time format %s was dropped, because the field is %s",
			*old.TimeFormat,
			kind,
		)
	}
	r := xpayload.Read{Kind: kind}
	drop := func(e payloadv0.EnumEntry, why string) {
		note("enum label %q was dropped: %s", e.Label, why)
	}
	switch kind {
	case xpayload.KindFloat, xpayload.KindInteger, xpayload.KindUnsigned:
		if known && ch == nil {
			r.Number.DataType = &dt
		}
		for _, e := range old.EnumValues {
			v, ok := xpayload.FloatNumber(xpayload.Float(e.Value)).To(kind)
			switch {
			case e.Label == "":
				drop(e, "it is empty")
			case !ok:
				drop(
					e,
					strconv.FormatFloat(e.Value, 'g', -1, 64)+" is not "+string(kind),
				)
			default:
				r.Number.EnumValues = append(
					r.Number.EnumValues,
					xpayload.NumberEntry{Label: e.Label, Value: v},
				)
			}
		}
	case xpayload.KindBoolean:
		for _, e := range old.EnumValues {
			if e.Label == "" {
				drop(e, "it is empty")
				continue
			}
			r.Boolean.EnumValues = append(
				r.Boolean.EnumValues,
				xpayload.BooleanEntry{Label: e.Label, Value: e.Value != 0},
			)
		}
	case xpayload.KindTimestamp:
		r.Timestamp.TimeFormat = xpayload.TimeFormatISO8601
		if old.TimeFormat != nil {
			r.Timestamp.TimeFormat = *old.TimeFormat
		}
	}
	if kind != xpayload.KindFloat && kind != xpayload.KindInteger &&
		kind != xpayload.KindUnsigned && kind != xpayload.KindBoolean {
		for _, e := range old.EnumValues {
			drop(e, "a "+string(kind)+" field takes no numeric entries")
		}
	}
	return NewReadField(BaseReadField{
		Key:      old.Key,
		Name:     old.Name,
		Disabled: old.Disabled,
		Channel:  old.Channel,
		Pointer:  old.Pointer,
	}, r), changes
}

// migrateReadConfig lifts a v1 read config to v2, with the channels of its fields in
// found, and calls note with the path and a description of each change to a field.
func migrateReadConfig(
	ctx context.Context,
	old v1.ReadConfig,
	found map[channel.Key]Channel,
	note func(field, change string),
) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	for i, e := range old.Entries {
		plain, ok := e.Variant.(v1.PlainReadEntry)
		if !ok {
			continue
		}
		fields := make([]ReadField, len(plain.Fields))
		for j, f := range plain.Fields {
			path := "entries." + strconv.Itoa(i) + ".fields." + strconv.Itoa(j)
			var changes []string
			var ch *Channel
			if c, ok := found[f.Channel]; ok {
				ch = &c
			}
			fields[j], changes = migrateReadField(f, ch)
			for _, c := range changes {
				note(path, c)
			}
		}
		entry, ok := cfg.Entries[i].Variant.(PlainReadEntry)
		if !ok {
			return ReadConfig{}, errors.Newf("entries.%d: expected a plain entry", i)
		}
		entry.Fields = fields
		cfg.Entries[i].Variant = entry
	}
	return cfg, nil
}

// MigrateReadConfig lifts a v1 read config to v2, as MigrateReadField lifts each of its
// fields.
func MigrateReadConfig(ctx context.Context, old v1.ReadConfig) (ReadConfig, error) {
	return migrateReadConfig(ctx, old, nil, func(string, string) {})
}

// NewReadMigration returns the migration that lifts stored read configs to v2, with the
// kind of each field's channel read through channels, then reconciles each field with
// its channel. It logs each change to a field.
func NewReadMigration(channels Channels) gorp.Migration {
	return migrate.New(
		"v62_mqtt_read_typed_fields",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v1.ReadConfig
			if err := gorp.NewRetrieve[uuid.UUID, v1.ReadConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil {
				return err
			}
			news := make([]ReadConfig, 0, len(olds))
			for _, old := range olds {
				found, err := v1FieldChannels(ctx, tx, channels, old)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				cfg, err := migrateReadConfig(
					ctx,
					old,
					found,
					func(field, change string) {
						ins.L.Warn(
							"changed an MQTT read field to v2",
							zap.Stringer("config", old.Key),
							zap.String("field", field),
							zap.String("change", change),
						)
					},
				)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				if err = ReconcileReadConfig(ctx, tx, channels, ins, &cfg); err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, news...)
		},
	)
}

func v1FieldChannels(
	ctx context.Context,
	tx gorp.Tx,
	channels Channels,
	cfg v1.ReadConfig,
) (map[channel.Key]Channel, error) {
	var keys []channel.Key
	for _, e := range cfg.Entries {
		if plain, ok := e.Variant.(v1.PlainReadEntry); ok {
			for _, f := range plain.Fields {
				if f.Channel != 0 {
					keys = append(keys, f.Channel)
				}
			}
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return channels(ctx, tx, lo.Uniq(keys))
}
