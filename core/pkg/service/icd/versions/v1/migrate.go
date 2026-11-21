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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/migrate"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// MigrateICD lifts a v0 ICD to v1.
func MigrateICD(ctx context.Context, old v0.ICD) (ICD, error) {
	return autoMigrateICD(ctx, old)
}

// MigrateMessage lifts a v0 message to v1, with no cycle time. Each binary field takes
// the variant of its kind, with no default. A binary field that is not a number becomes
// unsigned, which reads the same bits.
func MigrateMessage(ctx context.Context, old v0.Message) (Message, error) {
	m, err := autoMigrateMessage(ctx, old)
	if err != nil {
		return Message{}, err
	}
	oldBinary, ok := old.Payload.Variant.(v0.BinaryPayload)
	if !ok {
		return m, nil
	}
	m.Payload.Variant = BinaryPayload{
		Length:     oldBinary.Length,
		Identifier: oldBinary.Identifier,
		Fields:     lo.Map(oldBinary.Fields, migrateBinaryField),
	}
	return m, nil
}

func migrateBinaryField(old v0.BinaryField, _ int) BinaryField {
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
		return BinaryField{Variant: FloatBinaryField{BinaryBaseField: base}}
	case payload.KindInteger:
		return BinaryField{Variant: IntegerBinaryField{BinaryBaseField: base}}
	case payload.KindUnsigned,
		payload.KindString,
		payload.KindBoolean,
		payload.KindTimestamp,
		payload.KindJSON,
		payload.KindUUID,
		payload.KindBytes:
	}
	return BinaryField{Variant: UnsignedBinaryField{BinaryBaseField: base}}
}

var numberKinds = set.New(payload.KindFloat, payload.KindInteger, payload.KindUnsigned)

// migrateStored lifts a stored v0 ICD and logs each binary field that is not a number.
func migrateStored(
	ctx context.Context,
	old v0.ICD,
	ins alamos.Instrumentation,
) (ICD, error) {
	for _, m := range old.Messages {
		b, ok := m.Payload.Variant.(v0.BinaryPayload)
		if !ok {
			continue
		}
		for _, f := range b.Fields {
			if numberKinds.Contains(f.Kind) {
				continue
			}
			ins.L.Warn(
				"lifted a binary field that is not a number to unsigned",
				zap.Stringer("icd", old.Key),
				zap.Stringer("message", m.Key),
				zap.Stringer("field", f.Key),
				zap.String("kind", string(f.Kind)),
			)
		}
	}
	return MigrateICD(ctx, old)
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

// MigrateJSONMatch lifts a v0 JSON match rule to v1.
func MigrateJSONMatch(ctx context.Context, old v0.JSONMatch) (JSONMatch, error) {
	return autoMigrateJSONMatch(ctx, old)
}

// Migration lifts stored ICDs from v0 to v1.
var Migration = migrate.NewInstrumentedEntry("v63_message_cycle_time", migrateStored)
