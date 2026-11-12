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
	"math"
	"reflect"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	"go.uber.org/zap"
)

// MigrateScanConfig lifts a v2 scan config to v3.
func MigrateScanConfig(ctx context.Context, old v2.ScanConfig) (ScanConfig, error) {
	return autoMigrateScanConfig(ctx, old)
}

// ScanMigration lifts stored scan configs to v3.
var ScanMigration = migrate.NewEntry("v61_drop_ignored_models", MigrateScanConfig)

// maxLine is the highest line of a DAQmx digital port.
const maxLine = 31

// narrowPort clamps a port to uint32 and reports whether it changed.
func narrowPort(old int32) (uint32, bool) { return uint32(max(old, 0)), old < 0 }

// narrowLine clamps a line to [0, maxLine] and reports whether it changed.
func narrowLine(old int32) (uint8, bool) {
	line := min(max(old, 0), maxLine)
	return uint8(line), line != old
}

// narrowCount truncates a count to uint32, with lower as its minimum.
func narrowCount(old float64, lower uint32) uint32 {
	if math.IsNaN(old) || old < float64(lower) {
		return lower
	}
	return uint32(min(old, math.MaxUint32))
}

// narrowCountOrDefault is narrowCount, except that 0 becomes def, as v2 read an absent
// count.
func narrowCountOrDefault(old float64, lower, def uint32) uint32 {
	if old == 0 {
		return def
	}
	return narrowCount(old, lower)
}

// narrowDivisor clamps a divisor to at least 1. 0 becomes def, as v2 read an absent
// divisor.
func narrowDivisor(old int32, def uint32) uint32 {
	if old == 0 {
		return def
	}
	return uint32(max(old, 1))
}

// MigrateBaseAIChannel lifts a v2 analog input base to v3. A negative port becomes 0
// and disables the channel.
func MigrateBaseAIChannel(
	ctx context.Context,
	old v2.BaseAIChannel,
) (BaseAIChannel, error) {
	base, err := autoMigrateBaseAIChannel(ctx, old)
	if err != nil {
		return BaseAIChannel{}, err
	}
	var clamped bool
	base.Port, clamped = narrowPort(old.Port)
	base.Disabled = base.Disabled || clamped
	return base, nil
}

// MigrateBaseAOChannel lifts a v2 analog output base to v3. A negative port becomes 0
// and disables the channel.
func MigrateBaseAOChannel(
	ctx context.Context,
	old v2.BaseAOChannel,
) (BaseAOChannel, error) {
	base, err := autoMigrateBaseAOChannel(ctx, old)
	if err != nil {
		return BaseAOChannel{}, err
	}
	var clamped bool
	base.Port, clamped = narrowPort(old.Port)
	base.Disabled = base.Disabled || clamped
	return base, nil
}

// MigrateAOChannel lifts a v2 analog output channel to v3. A function generator drops
// its command and state channels. The other types keep them.
func MigrateAOChannel(ctx context.Context, old v2.AOChannel) (AOChannel, error) {
	ch, err := autoMigrateAOChannel(ctx, old)
	if err != nil {
		return AOChannel{}, err
	}
	base := old.Base()
	commanded := CommandedAOChannel{
		CmdChannel:       base.CmdChannel,
		StateChannel:     base.StateChannel,
		CmdChannelName:   base.CmdChannelName,
		StateChannelName: base.StateChannelName,
	}
	switch v := ch.Variant.(type) {
	case AOCurrentChannel:
		v.CommandedAOChannel = commanded
		ch.Variant = v
	case AOVoltageChannel:
		v.CommandedAOChannel = commanded
		ch.Variant = v
	}
	return ch, nil
}

// MigrateBaseCIChannel lifts a v2 counter input base to v3. A negative port becomes 0
// and disables the channel.
func MigrateBaseCIChannel(
	ctx context.Context,
	old v2.BaseCIChannel,
) (BaseCIChannel, error) {
	base, err := autoMigrateBaseCIChannel(ctx, old)
	if err != nil {
		return BaseCIChannel{}, err
	}
	var clamped bool
	base.Port, clamped = narrowPort(old.Port)
	base.Disabled = base.Disabled || clamped
	return base, nil
}

// MigrateCJC lifts a v2 cold-junction source to v3. A negative port becomes 0.
// MigrateAnalogReadConfig disables the channel that holds it.
func MigrateCJC(ctx context.Context, old v2.CJC) (CJC, error) {
	cjc, err := autoMigrateCJC(ctx, old)
	if err != nil {
		return CJC{}, err
	}
	if c, ok := old.Variant.(v2.ChanCJC); ok {
		port, _ := narrowPort(c.Port)
		cjc.Variant = ChanCJC{Port: port}
	}
	return cjc, nil
}

// MigrateCIChannel lifts a v2 counter input channel to v3. A divisor or pulse count
// below 1 becomes 1, a negative initial count becomes 0, and a fraction truncates. An
// absent divisor or pulse count takes its schema default.
func MigrateCIChannel(ctx context.Context, old v2.CIChannel) (CIChannel, error) {
	ch, err := autoMigrateCIChannel(ctx, old)
	if err != nil {
		return CIChannel{}, err
	}
	switch o := old.Variant.(type) {
	case v2.CIFrequencyChannel:
		var def CIFrequencyChannel
		def.SetDefaults()
		n := ch.Variant.(CIFrequencyChannel)
		n.Divisor = narrowDivisor(o.Divisor, def.Divisor)
		ch.Variant = n
	case v2.CIPeriodChannel:
		var def CIPeriodChannel
		def.SetDefaults()
		n := ch.Variant.(CIPeriodChannel)
		n.Divisor = narrowDivisor(o.Divisor, def.Divisor)
		ch.Variant = n
	case v2.CIEdgeCountChannel:
		n := ch.Variant.(CIEdgeCountChannel)
		n.InitialCount = narrowCount(o.InitialCount, 0)
		ch.Variant = n
	case v2.CIVelocityAngularChannel:
		var def CIVelocityAngularChannel
		def.SetDefaults()
		n := ch.Variant.(CIVelocityAngularChannel)
		n.PulsesPerRev = narrowCountOrDefault(o.PulsesPerRev, 1, def.PulsesPerRev)
		ch.Variant = n
	case v2.CIPositionAngularChannel:
		var def CIPositionAngularChannel
		def.SetDefaults()
		n := ch.Variant.(CIPositionAngularChannel)
		n.PulsesPerRev = narrowCountOrDefault(o.PulsesPerRev, 1, def.PulsesPerRev)
		ch.Variant = n
	}
	return ch, nil
}

// MigrateDIChannel lifts a v2 digital input channel to v3. A port or line out of range
// clamps and disables the channel.
func MigrateDIChannel(ctx context.Context, old v2.DIChannel) (DIChannel, error) {
	ch, err := autoMigrateDIChannel(ctx, old)
	if err != nil {
		return DIChannel{}, err
	}
	var portClamped, lineClamped bool
	ch.Port, portClamped = narrowPort(old.Port)
	ch.Line, lineClamped = narrowLine(old.Line)
	ch.Disabled = ch.Disabled || portClamped || lineClamped
	return ch, nil
}

// MigrateDOChannel lifts a v2 digital output channel to v3. A port or line out of range
// clamps and disables the channel.
func MigrateDOChannel(ctx context.Context, old v2.DOChannel) (DOChannel, error) {
	ch, err := autoMigrateDOChannel(ctx, old)
	if err != nil {
		return DOChannel{}, err
	}
	var portClamped, lineClamped bool
	ch.Port, portClamped = narrowPort(old.Port)
	ch.Line, lineClamped = narrowLine(old.Line)
	ch.Disabled = ch.Disabled || portClamped || lineClamped
	return ch, nil
}

// MigrateAnalogReadConfig lifts a v2 analog read config to v3. A thermocouple whose
// cold-junction port clamps is disabled.
func MigrateAnalogReadConfig(
	ctx context.Context,
	old v2.AnalogReadConfig,
) (AnalogReadConfig, error) {
	cfg, err := autoMigrateAnalogReadConfig(ctx, old)
	if err != nil {
		return AnalogReadConfig{}, err
	}
	for i, ch := range old.Channels {
		tc, ok := ch.Variant.(v2.AIThermocoupleChannel)
		if !ok {
			continue
		}
		if c, ok := tc.Cjc.Variant.(v2.ChanCJC); ok && c.Port < 0 {
			n := cfg.Channels[i].Variant.(AIThermocoupleChannel)
			n.Disabled = true
			cfg.Channels[i].Variant = n
		}
	}
	return cfg, nil
}

// MigrateAnalogWriteConfig lifts a v2 analog write config to v3.
func MigrateAnalogWriteConfig(
	ctx context.Context,
	old v2.AnalogWriteConfig,
) (AnalogWriteConfig, error) {
	return autoMigrateAnalogWriteConfig(ctx, old)
}

// MigrateCounterReadConfig lifts a v2 counter read config to v3.
func MigrateCounterReadConfig(
	ctx context.Context,
	old v2.CounterReadConfig,
) (CounterReadConfig, error) {
	return autoMigrateCounterReadConfig(ctx, old)
}

// MigrateDigitalReadConfig lifts a v2 digital read config to v3.
func MigrateDigitalReadConfig(
	ctx context.Context,
	old v2.DigitalReadConfig,
) (DigitalReadConfig, error) {
	return autoMigrateDigitalReadConfig(ctx, old)
}

// MigrateDigitalWriteConfig lifts a v2 digital write config to v3.
func MigrateDigitalWriteConfig(
	ctx context.Context,
	old v2.DigitalWriteConfig,
) (DigitalWriteConfig, error) {
	return autoMigrateDigitalWriteConfig(ctx, old)
}

// narrowedFields are the Go names of the channel fields a migration can change.
var narrowedFields = []string{
	"Port",
	"Line",
	"Divisor",
	"InitialCount",
	"PulsesPerRev",
	"Disabled",
}

// channelValue returns the struct a channel holds: the variant of a union, or the
// channel itself.
func channelValue(ch reflect.Value) reflect.Value {
	if v := ch.FieldByName("Variant"); v.IsValid() {
		return v.Elem()
	}
	return ch
}

// fieldOf returns the named field of a channel, and false when it has no such field.
func fieldOf(ch reflect.Value, name string) (any, bool) {
	if !ch.IsValid() {
		return nil, false
	}
	f := ch.FieldByName(name)
	if !f.IsValid() {
		return nil, false
	}
	return f.Interface(), true
}

// sameValue reports whether two narrowed field values are equal, comparing numbers of
// different types by value.
func sameValue(a, b any) bool {
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	if av.CanFloat() || av.CanInt() || av.CanUint() {
		return asFloat(av) == asFloat(bv)
	}
	return a == b
}

func asFloat(v reflect.Value) float64 {
	switch {
	case v.CanInt():
		return float64(v.Int())
	case v.CanUint():
		return float64(v.Uint())
	default:
		return v.Float()
	}
}

// logChanges logs each narrowed channel field that differs between a stored config
// and its migration.
func logChanges(ins alamos.Instrumentation, key uuid.UUID, old, migrated any) {
	oldChannels := reflect.ValueOf(old).FieldByName("Channels")
	newChannels := reflect.ValueOf(migrated).FieldByName("Channels")
	for i := range oldChannels.Len() {
		o := channelValue(oldChannels.Index(i))
		n := channelValue(newChannels.Index(i))
		channelKey, _ := fieldOf(o, "Key")
		log := func(field string, before, after any) {
			ins.L.Warn(
				"narrowed an NI channel field",
				zap.Stringer("config", key),
				zap.Any("channel", channelKey),
				zap.String("field", field),
				zap.Any("old", before),
				zap.Any("new", after),
			)
		}
		for _, name := range narrowedFields {
			before, ok := fieldOf(o, name)
			if !ok {
				continue
			}
			if after, _ := fieldOf(n, name); !sameValue(before, after) {
				log(name, before, after)
			}
		}
		oldCJC, ok := fieldOf(o, "Cjc")
		if !ok {
			continue
		}
		newCJC, _ := fieldOf(n, "Cjc")
		before, ok := fieldOf(channelValue(reflect.ValueOf(oldCJC)), "Port")
		if !ok {
			continue
		}
		after, _ := fieldOf(channelValue(reflect.ValueOf(newCJC)), "Port")
		if !sameValue(before, after) {
			log("Cjc.Port", before, after)
		}
	}
}

// logged wraps a config migration so that it logs each channel field it changes.
func logged[O gorp.Entry[uuid.UUID], N any](
	migrateFn func(context.Context, O) (N, error),
) func(context.Context, O, alamos.Instrumentation) (N, error) {
	return func(ctx context.Context, old O, ins alamos.Instrumentation) (N, error) {
		migrated, err := migrateFn(ctx, old)
		if err != nil {
			var zero N
			return zero, err
		}
		logChanges(ins, old.GorpKey(), old, migrated)
		return migrated, nil
	}
}

const narrowMigrationKey = "v61_narrow_field_types"

// AnalogReadMigration lifts stored analog read configs to v3.
var AnalogReadMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateAnalogReadConfig),
)

// AnalogWriteMigration lifts stored analog write configs to v3.
var AnalogWriteMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateAnalogWriteConfig),
)

// CounterReadMigration lifts stored counter read configs to v3.
var CounterReadMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateCounterReadConfig),
)

// DigitalReadMigration lifts stored digital read configs to v3.
var DigitalReadMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateDigitalReadConfig),
)

// DigitalWriteMigration lifts stored digital write configs to v3.
var DigitalWriteMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateDigitalWriteConfig),
)
