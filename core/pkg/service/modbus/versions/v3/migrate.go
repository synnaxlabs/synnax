// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v2"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

const (
	// MinStringLength is the fewest characters a string channel reads.
	MinStringLength = 1
	// MaxStringLength is the most characters a string channel reads, so that it fits
	// in one request of 125 registers.
	MaxStringLength = 250
)

func clampLength(length int32) uint16 {
	return uint16(min(max(length, MinStringLength), MaxStringLength))
}

// storedString is a v2 register value of the string data type.
type storedString struct {
	base         v2.BaseReadChannel
	bytesSwapped *bool
	length       int32
	holding      bool
}

func parseStoredString(old v2.ReadChannel) (storedString, bool) {
	var (
		s     storedString
		value v2.RegisterValue
	)
	switch v := old.Variant.(type) {
	case v2.HoldingRegisterReadChannel:
		s = storedString{base: v.BaseReadChannel, length: v.StringLength, holding: true}
		value = v.RegisterValue
	case v2.InputRegisterReadChannel:
		s = storedString{base: v.BaseReadChannel, length: v.StringLength}
		value = v.RegisterValue
	default:
		return s, false
	}
	s.bytesSwapped = value.BytesSwapped
	return s, value.DataType == telem.StringT
}

// MigrateReadChannel lifts a v2 read channel to v3. A register value of the string
// data type becomes the string variant of its register space, with its length clamped
// to 1 through 250. Every other variant keeps its shape. An empty v2 data type becomes
// uint8, the v2 default.
func MigrateReadChannel(ctx context.Context, old v2.ReadChannel) (ReadChannel, error) {
	old.ApplyDefaults()
	s, ok := parseStoredString(old)
	if !ok {
		return autoMigrateReadChannel(ctx, old)
	}
	str := RegisterString{Length: clampLength(s.length), BytesSwapped: s.bytesSwapped}
	if s.holding {
		return ReadChannel{Variant: HoldingRegisterStringReadChannel{
			BaseReadChannel: s.base,
			RegisterString:  str,
		}}, nil
	}
	return ReadChannel{Variant: InputRegisterStringReadChannel{
		BaseReadChannel: s.base,
		RegisterString:  str,
	}}, nil
}

// MigrateReadConfig lifts a v2 read config to v3.
func MigrateReadConfig(ctx context.Context, old v2.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// migrateStoredRead is MigrateReadConfig for the stored-data migration. It logs each
// string length that the migration clamps.
func migrateStoredRead(
	ctx context.Context,
	old v2.ReadConfig,
	ins alamos.Instrumentation,
) (ReadConfig, error) {
	for _, ch := range old.Channels {
		s, ok := parseStoredString(ch)
		if !ok || int32(clampLength(s.length)) == s.length {
			continue
		}
		ins.L.Warn(
			"clamped the length of a Modbus string channel",
			zap.Stringer("config", old.Key),
			zap.String("channel", s.base.Key),
			zap.String("field", "length"),
			zap.Int32("old", s.length),
			zap.Uint16("new", clampLength(s.length)),
		)
	}
	return MigrateReadConfig(ctx, old)
}

// MigrateWriteChannel lifts a v2 write channel to v3. Every v2 variant exists in v3. An
// empty v2 data type becomes uint8, the v2 default.
func MigrateWriteChannel(
	ctx context.Context,
	old v2.WriteChannel,
) (WriteChannel, error) {
	old.ApplyDefaults()
	return autoMigrateWriteChannel(ctx, old)
}

// MigrateWriteConfig lifts a v2 write config to v3.
func MigrateWriteConfig(ctx context.Context, old v2.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

var (
	// ReadMigration lifts stored read configs to v3.
	ReadMigration = migrate.NewInstrumentedEntry(
		"v61_string_registers",
		migrateStoredRead,
	)
	// WriteMigration lifts stored write configs to v3.
	WriteMigration = migrate.NewEntry("v61_register_bits", MigrateWriteConfig)
)
