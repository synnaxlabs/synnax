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

	"github.com/samber/lo"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	xpayload "github.com/synnaxlabs/x/payload"
	payload "github.com/synnaxlabs/x/payload/versions/v1"
	"github.com/synnaxlabs/x/telem"
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
		if old.JSONType == payload.JSONTypeNumber {
			dt = old.DataType
		}
		c = xpayload.Channel{Kind: xpayload.KindTimestamp, TimeFormat: *old.TimeFormat}
	case old.JSONType == payload.JSONTypeBoolean:
		c = xpayload.Channel{Kind: xpayload.KindBoolean}
	case old.JSONType == payload.JSONTypeString && len(old.EnumValues) == 0:
		c = xpayload.Channel{Kind: xpayload.KindString}
	default:
		c = xpayload.Channel{
			Kind: xpayload.KindFloat,
			Float: lo.Map(
				old.EnumValues,
				func(e payload.EnumEntry, _ int) payload.FloatEntry {
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
var fallbackDataTypes = map[payload.JSONType]telem.DataType{
	payload.JSONTypeString:  telem.StringT,
	payload.JSONTypeBoolean: telem.BooleanT,
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
