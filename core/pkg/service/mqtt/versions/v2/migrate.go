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
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/telem"
)

// MigrateChannelField lifts a v1 channel field to v2 by its stored shape, without its
// Synnax channel. A timestamp data type, or a number with a time format and no entries,
// the default shape of a v1 timestamp field, makes a timestamp field in its time
// format, ISO 8601 when it has none. A boolean JSON type makes a boolean field, and a
// string with no entries a string field. A number, or a string with entries, makes a
// float field that keeps the entries. v1 encoded by the data type of the channel, so
// the caller reconciles the field against its channel, or against its data type when
// it has none.
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
	case dt == telem.TimestampT || (old.TimeFormat != nil &&
		old.JSONType == payload.JSONTypeNumber && len(old.EnumValues) == 0):
		c = xpayload.Channel{
			Kind:       xpayload.KindTimestamp,
			TimeFormat: lo.FromPtrOr(old.TimeFormat, payload.TimeFormatISO8601),
		}
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
// lifts as MigrateChannelField does, and a Sparkplug B target keeps its fields.
func MigrateWriteTarget(ctx context.Context, old v1.WriteTarget) (WriteTarget, error) {
	switch t := old.Variant.(type) {
	case v1.PlainWriteTarget:
		ch, err := MigrateChannelField(ctx, t.Channel)
		if err != nil {
			return WriteTarget{}, err
		}
		return WriteTarget{Variant: PlainWriteTarget{
			BaseWriteTarget: t.BaseWriteTarget,
			Topic:           t.Topic,
			Qos:             t.Qos,
			Retained:        t.Retained,
			Channel:         ch,
			Fields:          t.Fields,
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
