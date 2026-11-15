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
	"fmt"
	"math"
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

// changeLog logs the fields a migration narrows in one stored channel.
type changeLog struct {
	ins     alamos.Instrumentation
	config  uuid.UUID
	channel string
}

func (l changeLog) warn(field string, old, new zap.Field) {
	l.ins.L.Warn(
		"narrowed an NI channel field",
		zap.Stringer("config", l.config),
		zap.String("channel", l.channel),
		zap.String("field", field),
		old,
		new,
	)
}

type number interface {
	~int32 | ~uint32 | ~uint8 | ~float64
}

// logNumber logs field when its old and new values differ.
func logNumber[O, N number](l changeLog, field string, old O, new N) {
	if float64(old) != float64(new) {
		l.warn(field, zap.Any("old", old), zap.Any("new", new))
	}
}

// logFlag logs field when its old and new values differ.
func logFlag(l changeLog, field string, old, new bool) {
	if old != new {
		l.warn(field, zap.Bool("old", old), zap.Bool("new", new))
	}
}

// aiBases returns the shared fields of a v2 analog input variant and of its migration.
// It panics for a variant without shared fields.
func aiBases(
	old v2.AIChannelVariant,
	n AIChannelVariant,
) (v2.BaseAIChannel, BaseAIChannel) {
	switch o := old.(type) {
	case v2.AIVoltageChannel:
		return o.BaseAIChannel, n.(AIVoltageChannel).BaseAIChannel
	case v2.AIAccelChannel:
		return o.BaseAIChannel, n.(AIAccelChannel).BaseAIChannel
	case v2.AIBridgeChannel:
		return o.BaseAIChannel, n.(AIBridgeChannel).BaseAIChannel
	case v2.AICurrentChannel:
		return o.BaseAIChannel, n.(AICurrentChannel).BaseAIChannel
	case v2.AIForceBridgeTableChannel:
		return o.BaseAIChannel, n.(AIForceBridgeTableChannel).BaseAIChannel
	case v2.AIForceBridgeTwoPointLinChannel:
		return o.BaseAIChannel, n.(AIForceBridgeTwoPointLinChannel).BaseAIChannel
	case v2.AIForceIEPEChannel:
		return o.BaseAIChannel, n.(AIForceIEPEChannel).BaseAIChannel
	case v2.AIMicrophoneChannel:
		return o.BaseAIChannel, n.(AIMicrophoneChannel).BaseAIChannel
	case v2.AIPressureBridgeTableChannel:
		return o.BaseAIChannel, n.(AIPressureBridgeTableChannel).BaseAIChannel
	case v2.AIPressureBridgeTwoPointLinChannel:
		return o.BaseAIChannel, n.(AIPressureBridgeTwoPointLinChannel).BaseAIChannel
	case v2.AIResistanceChannel:
		return o.BaseAIChannel, n.(AIResistanceChannel).BaseAIChannel
	case v2.AIRTDChannel:
		return o.BaseAIChannel, n.(AIRTDChannel).BaseAIChannel
	case v2.AIStrainGaugeChannel:
		return o.BaseAIChannel, n.(AIStrainGaugeChannel).BaseAIChannel
	case v2.AIThermocoupleChannel:
		return o.BaseAIChannel, n.(AIThermocoupleChannel).BaseAIChannel
	case v2.AITorqueBridgeTableChannel:
		return o.BaseAIChannel, n.(AITorqueBridgeTableChannel).BaseAIChannel
	case v2.AITorqueBridgeTwoPointLinChannel:
		return o.BaseAIChannel, n.(AITorqueBridgeTwoPointLinChannel).BaseAIChannel
	case v2.AIVelocityIEPEChannel:
		return o.BaseAIChannel, n.(AIVelocityIEPEChannel).BaseAIChannel
	case v2.AIAccel4WireDCVoltageChannel:
		return o.BaseAIChannel, n.(AIAccel4WireDCVoltageChannel).BaseAIChannel
	case v2.AIAccelChargeChannel:
		return o.BaseAIChannel, n.(AIAccelChargeChannel).BaseAIChannel
	case v2.AIChargeChannel:
		return o.BaseAIChannel, n.(AIChargeChannel).BaseAIChannel
	case v2.AICurrentRMSChannel:
		return o.BaseAIChannel, n.(AICurrentRMSChannel).BaseAIChannel
	case v2.AIForceBridgePolynomialChannel:
		return o.BaseAIChannel, n.(AIForceBridgePolynomialChannel).BaseAIChannel
	case v2.AIFreqVoltageChannel:
		return o.BaseAIChannel, n.(AIFreqVoltageChannel).BaseAIChannel
	case v2.AIPressureBridgePolynomialChannel:
		return o.BaseAIChannel, n.(AIPressureBridgePolynomialChannel).BaseAIChannel
	case v2.AIThermistorIexChannel:
		return o.BaseAIChannel, n.(AIThermistorIexChannel).BaseAIChannel
	case v2.AIThermistorVexChannel:
		return o.BaseAIChannel, n.(AIThermistorVexChannel).BaseAIChannel
	case v2.AITorqueBridgePolynomialChannel:
		return o.BaseAIChannel, n.(AITorqueBridgePolynomialChannel).BaseAIChannel
	case v2.AIVoltageRMSChannel:
		return o.BaseAIChannel, n.(AIVoltageRMSChannel).BaseAIChannel
	case v2.AIVoltageWithExcitChannel:
		return o.BaseAIChannel, n.(AIVoltageWithExcitChannel).BaseAIChannel
	default:
		panic(fmt.Sprintf("analog input variant %T has no base", old))
	}
}

func logAIChannel(l changeLog, old v2.AIChannel, migrated AIChannel) {
	if o, ok := old.Variant.(v2.AITempBuiltinChannel); ok {
		l.channel = o.Key
		n := migrated.Variant.(AITempBuiltinChannel)
		logFlag(l, "Disabled", o.Disabled, n.Disabled)
		return
	}
	if old.Variant == nil {
		return
	}
	o, n := aiBases(old.Variant, migrated.Variant)
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	logFlag(l, "Disabled", o.Disabled, n.Disabled)
	tc, ok := old.Variant.(v2.AIThermocoupleChannel)
	if !ok {
		return
	}
	oldCJC, ok := tc.Cjc.Variant.(v2.ChanCJC)
	if !ok {
		return
	}
	newCJC := migrated.Variant.(AIThermocoupleChannel).Cjc.Variant.(ChanCJC)
	logNumber(l, "Cjc.Port", oldCJC.Port, newCJC.Port)
}

func logAOChannel(l changeLog, old v2.AOChannel, migrated AOChannel) {
	if old.Variant == nil {
		return
	}
	o, n := old.Base(), migrated.Base()
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	logFlag(l, "Disabled", o.Disabled, n.Disabled)
}

func logCIChannel(l changeLog, old v2.CIChannel, migrated CIChannel) {
	if old.Variant == nil {
		return
	}
	o, n := old.Base(), migrated.Base()
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	switch v := old.Variant.(type) {
	case v2.CIFrequencyChannel:
		after := migrated.Variant.(CIFrequencyChannel).Divisor
		logNumber(l, "Divisor", v.Divisor, after)
	case v2.CIPeriodChannel:
		after := migrated.Variant.(CIPeriodChannel).Divisor
		logNumber(l, "Divisor", v.Divisor, after)
	case v2.CIEdgeCountChannel:
		after := migrated.Variant.(CIEdgeCountChannel).InitialCount
		logNumber(l, "InitialCount", v.InitialCount, after)
	case v2.CIVelocityAngularChannel:
		after := migrated.Variant.(CIVelocityAngularChannel).PulsesPerRev
		logNumber(l, "PulsesPerRev", v.PulsesPerRev, after)
	case v2.CIPositionAngularChannel:
		after := migrated.Variant.(CIPositionAngularChannel).PulsesPerRev
		logNumber(l, "PulsesPerRev", v.PulsesPerRev, after)
	}
	logFlag(l, "Disabled", o.Disabled, n.Disabled)
}

func logDIChannel(l changeLog, old v2.DIChannel, migrated DIChannel) {
	l.channel = old.Key
	logNumber(l, "Port", old.Port, migrated.Port)
	logNumber(l, "Line", old.Line, migrated.Line)
	logFlag(l, "Disabled", old.Disabled, migrated.Disabled)
}

func logDOChannel(l changeLog, old v2.DOChannel, migrated DOChannel) {
	l.channel = old.Key
	logNumber(l, "Port", old.Port, migrated.Port)
	logNumber(l, "Line", old.Line, migrated.Line)
	logFlag(l, "Disabled", old.Disabled, migrated.Disabled)
}

func logAnalogRead(l changeLog, old v2.AnalogReadConfig, migrated AnalogReadConfig) {
	for i, ch := range old.Channels {
		logAIChannel(l, ch, migrated.Channels[i])
	}
}

func logAnalogWrite(l changeLog, old v2.AnalogWriteConfig, migrated AnalogWriteConfig) {
	for i, ch := range old.Channels {
		logAOChannel(l, ch, migrated.Channels[i])
	}
}

func logCounterRead(l changeLog, old v2.CounterReadConfig, migrated CounterReadConfig) {
	for i, ch := range old.Channels {
		logCIChannel(l, ch, migrated.Channels[i])
	}
}

func logDigitalRead(l changeLog, old v2.DigitalReadConfig, migrated DigitalReadConfig) {
	for i, ch := range old.Channels {
		logDIChannel(l, ch, migrated.Channels[i])
	}
}

func logDigitalWrite(
	l changeLog,
	old v2.DigitalWriteConfig,
	migrated DigitalWriteConfig,
) {
	for i, ch := range old.Channels {
		logDOChannel(l, ch, migrated.Channels[i])
	}
}

// logged wraps a config migration so that it logs each channel field it changes.
func logged[O gorp.Entry[uuid.UUID], N any](
	migrateFn func(context.Context, O) (N, error),
	logFn func(changeLog, O, N),
) func(context.Context, O, alamos.Instrumentation) (N, error) {
	return func(ctx context.Context, old O, ins alamos.Instrumentation) (N, error) {
		migrated, err := migrateFn(ctx, old)
		if err != nil {
			var zero N
			return zero, err
		}
		logFn(changeLog{ins: ins, config: old.GorpKey()}, old, migrated)
		return migrated, nil
	}
}

const narrowMigrationKey = "v61_narrow_field_types"

// AnalogReadMigration lifts stored analog read configs to v3.
var AnalogReadMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateAnalogReadConfig, logAnalogRead),
)

// AnalogWriteMigration lifts stored analog write configs to v3.
var AnalogWriteMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateAnalogWriteConfig, logAnalogWrite),
)

// CounterReadMigration lifts stored counter read configs to v3.
var CounterReadMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateCounterReadConfig, logCounterRead),
)

// DigitalReadMigration lifts stored digital read configs to v3.
var DigitalReadMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateDigitalReadConfig, logDigitalRead),
)

// DigitalWriteMigration lifts stored digital write configs to v3.
var DigitalWriteMigration = migrate.NewInstrumentedEntry(
	narrowMigrationKey,
	logged(MigrateDigitalWriteConfig, logDigitalWrite),
)
