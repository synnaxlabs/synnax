// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp/migrate"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/errors"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
)

// MigrateICD lifts a v0 ICD to v1.
func MigrateICD(ctx context.Context, old v0.ICD) (ICD, error) {
	return autoMigrateICD(ctx, old)
}

// MigrateMessage lifts a v0 message to v1, with no cycle time. Each binary field takes
// the variant of its kind, with no default. It fails on a binary field of any other
// kind.
func MigrateMessage(ctx context.Context, old v0.Message) (Message, error) {
	m, err := autoMigrateMessage(ctx, old)
	if err != nil {
		return Message{}, err
	}
	oldBinary, ok := old.Payload.Variant.(v0.BinaryPayload)
	if !ok {
		return m, nil
	}
	fields, err := lo.MapErr(oldBinary.Fields, migrateBinaryField)
	if err != nil {
		return Message{}, err
	}
	m.Payload.Variant = BinaryPayload{
		Length:     oldBinary.Length,
		Identifier: oldBinary.Identifier,
		Fields:     fields,
	}
	return m, nil
}

func migrateBinaryField(old v0.BinaryField, _ int) (BinaryField, error) {
	base := BinaryBaseField{
		BaseField:       old.BaseField,
		StartBit:        old.StartBit,
		BitLength:       old.BitLength,
		ByteOrder:       old.ByteOrder,
		Multiplexor:     old.Multiplexor,
		MultiplexValues: old.MultiplexValues,
	}
	switch old.Kind {
	case payload.KindFloat:
		return BinaryField{Variant: FloatBinaryField{BinaryBaseField: base}}, nil
	case payload.KindInteger:
		return BinaryField{Variant: IntegerBinaryField{BinaryBaseField: base}}, nil
	case payload.KindUnsigned:
		return BinaryField{Variant: UnsignedBinaryField{BinaryBaseField: base}}, nil
	case payload.KindString,
		payload.KindBoolean,
		payload.KindTimestamp,
		payload.KindJSON,
		payload.KindUUID,
		payload.KindBytes:
	}
	return BinaryField{}, errors.Newf(
		"binary field %q has kind %q, not float, integer, or unsigned",
		old.Name,
		old.Kind,
	)
}

// MigrateTextField lifts a v0 text field to v1, with no default.
func MigrateTextField(ctx context.Context, old v0.TextField) (TextField, error) {
	f, err := autoMigrateTextField(ctx, old)
	if err != nil || f.Variant == nil {
		return f, err
	}
	f.SetBase(TextBaseField{BaseField: old.Base()})
	return f, nil
}

// MigrateJSONField lifts a v0 JSON field to v1. A timestamp default becomes a constant
// default. The fields of the JSON, UUID, and bytes types get no default.
func MigrateJSONField(ctx context.Context, old v0.JSONField) (JSONField, error) {
	oldStamp, ok := old.Variant.(v0.TimestampJSONField)
	if !ok {
		return autoMigrateJSONField(ctx, old)
	}
	stamp := TimestampJSONField{
		JSONBaseField: oldStamp.JSONBaseField,
		TimeFormat:    oldStamp.TimeFormat,
	}
	if oldStamp.Default != nil {
		stamp.Default = &TimestampDefault{
			Variant: ConstantTimestampDefault{Value: *oldStamp.Default},
		}
	}
	return JSONField{Variant: stamp}, nil
}

// Migration lifts stored ICDs from v0 to v1.
var Migration = migrate.NewEntry("v63_message_cycle_time", MigrateICD)
