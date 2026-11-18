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
		n := as[CIFrequencyChannel](ch.Variant)
		n.Divisor = narrowDivisor(o.Divisor, def.Divisor)
		ch.Variant = n
	case v2.CIPeriodChannel:
		var def CIPeriodChannel
		def.SetDefaults()
		n := as[CIPeriodChannel](ch.Variant)
		n.Divisor = narrowDivisor(o.Divisor, def.Divisor)
		ch.Variant = n
	case v2.CIEdgeCountChannel:
		n := as[CIEdgeCountChannel](ch.Variant)
		n.InitialCount = narrowCount(o.InitialCount, 0)
		ch.Variant = n
	case v2.CIVelocityAngularChannel:
		var def CIVelocityAngularChannel
		def.SetDefaults()
		n := as[CIVelocityAngularChannel](ch.Variant)
		n.PulsesPerRev = narrowCountOrDefault(o.PulsesPerRev, 1, def.PulsesPerRev)
		ch.Variant = n
	case v2.CIPositionAngularChannel:
		var def CIPositionAngularChannel
		def.SetDefaults()
		n := as[CIPositionAngularChannel](ch.Variant)
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
			n := as[AIThermocoupleChannel](cfg.Channels[i].Variant)
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

func (l changeLog) warn(field string, old, next zap.Field) {
	l.ins.L.Warn(
		"narrowed an NI channel field",
		zap.Stringer("config", l.config),
		zap.String("channel", l.channel),
		zap.String("field", field),
		old,
		next,
	)
}

type number interface {
	~int32 | ~uint32 | ~uint8 | ~float64
}

// logNumber logs field when its old and new values differ.
func logNumber[O, N number](l changeLog, field string, old O, next N) {
	if float64(old) != float64(next) {
		l.warn(field, zap.Any("old", old), zap.Any("new", next))
	}
}

// logDisabled logs the Disabled field when its old and new values differ.
func logDisabled(l changeLog, old, next bool) {
	if old != next {
		l.warn("Disabled", zap.Bool("old", old), zap.Bool("new", next))
	}
}

// aiBases returns the shared fields of a v2 analog input variant and of its migration.
// It panics for a variant without shared fields, or for a migration of another kind.
func aiBases(
	old v2.AIChannelVariant,
	n AIChannelVariant,
) (v2.BaseAIChannel, BaseAIChannel) {
	for _, pair := range aiPairs {
		if o, b, ok := pair(old, n); ok {
			return o, b
		}
	}
	panic(fmt.Sprintf("analog input variant %T has no base", old))
}

// aiPair returns the shared fields of old and n when old is an O. It panics when old
// is an O and n is not an N.
func aiPair[
	O interface {
		v2.AIChannelVariant
		AIBase() v2.BaseAIChannel
	},
	N interface {
		AIChannelVariant
		aiBase() BaseAIChannel
	},
](old v2.AIChannelVariant, n AIChannelVariant) (v2.BaseAIChannel, BaseAIChannel, bool) {
	o, ok := old.(O)
	if !ok {
		return v2.BaseAIChannel{}, BaseAIChannel{}, false
	}
	return o.AIBase(), as[N](n).aiBase(), true
}

// aiPairs pairs each v2 analog input variant that has shared fields with the variant
// it migrates to.
var aiPairs = [...]func(
	v2.AIChannelVariant,
	AIChannelVariant,
) (v2.BaseAIChannel, BaseAIChannel, bool){
	aiPair[v2.AIVoltageChannel, AIVoltageChannel],
	aiPair[v2.AIAccelChannel, AIAccelChannel],
	aiPair[v2.AIBridgeChannel, AIBridgeChannel],
	aiPair[v2.AICurrentChannel, AICurrentChannel],
	aiPair[v2.AIForceBridgeTableChannel, AIForceBridgeTableChannel],
	aiPair[v2.AIForceBridgeTwoPointLinChannel, AIForceBridgeTwoPointLinChannel],
	aiPair[v2.AIForceIEPEChannel, AIForceIEPEChannel],
	aiPair[v2.AIMicrophoneChannel, AIMicrophoneChannel],
	aiPair[v2.AIPressureBridgeTableChannel, AIPressureBridgeTableChannel],
	aiPair[v2.AIPressureBridgeTwoPointLinChannel, AIPressureBridgeTwoPointLinChannel],
	aiPair[v2.AIResistanceChannel, AIResistanceChannel],
	aiPair[v2.AIRTDChannel, AIRTDChannel],
	aiPair[v2.AIStrainGaugeChannel, AIStrainGaugeChannel],
	aiPair[v2.AIThermocoupleChannel, AIThermocoupleChannel],
	aiPair[v2.AITorqueBridgeTableChannel, AITorqueBridgeTableChannel],
	aiPair[v2.AITorqueBridgeTwoPointLinChannel, AITorqueBridgeTwoPointLinChannel],
	aiPair[v2.AIVelocityIEPEChannel, AIVelocityIEPEChannel],
	aiPair[v2.AIAccel4WireDCVoltageChannel, AIAccel4WireDCVoltageChannel],
	aiPair[v2.AIAccelChargeChannel, AIAccelChargeChannel],
	aiPair[v2.AIChargeChannel, AIChargeChannel],
	aiPair[v2.AICurrentRMSChannel, AICurrentRMSChannel],
	aiPair[v2.AIForceBridgePolynomialChannel, AIForceBridgePolynomialChannel],
	aiPair[v2.AIFreqVoltageChannel, AIFreqVoltageChannel],
	aiPair[v2.AIPressureBridgePolynomialChannel, AIPressureBridgePolynomialChannel],
	aiPair[v2.AIThermistorIexChannel, AIThermistorIexChannel],
	aiPair[v2.AIThermistorVexChannel, AIThermistorVexChannel],
	aiPair[v2.AITorqueBridgePolynomialChannel, AITorqueBridgePolynomialChannel],
	aiPair[v2.AIVoltageRMSChannel, AIVoltageRMSChannel],
	aiPair[v2.AIVoltageWithExcitChannel, AIVoltageWithExcitChannel],
}

func logAIChannel(l changeLog, old v2.AIChannel, migrated AIChannel) {
	if o, ok := old.Variant.(v2.AITempBuiltinChannel); ok {
		l.channel = o.Key
		n := as[AITempBuiltinChannel](migrated.Variant)
		logDisabled(l, o.Disabled, n.Disabled)
		return
	}
	if old.Variant == nil {
		return
	}
	o, n := aiBases(old.Variant, migrated.Variant)
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	logDisabled(l, o.Disabled, n.Disabled)
	tc, ok := old.Variant.(v2.AIThermocoupleChannel)
	if !ok {
		return
	}
	oldCJC, ok := tc.Cjc.Variant.(v2.ChanCJC)
	if !ok {
		return
	}
	newTC := as[AIThermocoupleChannel](migrated.Variant)
	newCJC := as[ChanCJC](newTC.Cjc.Variant)
	logNumber(l, "Cjc.Port", oldCJC.Port, newCJC.Port)
}

func logAOChannel(l changeLog, old v2.AOChannel, migrated AOChannel) {
	if old.Variant == nil {
		return
	}
	o, n := old.Base(), migrated.Base()
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	logDisabled(l, o.Disabled, n.Disabled)
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
		after := as[CIFrequencyChannel](migrated.Variant).Divisor
		logNumber(l, "Divisor", v.Divisor, after)
	case v2.CIPeriodChannel:
		after := as[CIPeriodChannel](migrated.Variant).Divisor
		logNumber(l, "Divisor", v.Divisor, after)
	case v2.CIEdgeCountChannel:
		after := as[CIEdgeCountChannel](migrated.Variant).InitialCount
		logNumber(l, "InitialCount", v.InitialCount, after)
	case v2.CIVelocityAngularChannel:
		after := as[CIVelocityAngularChannel](migrated.Variant).PulsesPerRev
		logNumber(l, "PulsesPerRev", v.PulsesPerRev, after)
	case v2.CIPositionAngularChannel:
		after := as[CIPositionAngularChannel](migrated.Variant).PulsesPerRev
		logNumber(l, "PulsesPerRev", v.PulsesPerRev, after)
	}
	logDisabled(l, o.Disabled, n.Disabled)
}

func logDIChannel(l changeLog, old v2.DIChannel, migrated DIChannel) {
	l.channel = old.Key
	logNumber(l, "Port", old.Port, migrated.Port)
	logNumber(l, "Line", old.Line, migrated.Line)
	logDisabled(l, old.Disabled, migrated.Disabled)
}

func logDOChannel(l changeLog, old v2.DOChannel, migrated DOChannel) {
	l.channel = old.Key
	logNumber(l, "Port", old.Port, migrated.Port)
	logNumber(l, "Line", old.Line, migrated.Line)
	logDisabled(l, old.Disabled, migrated.Disabled)
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

// as returns a migrated variant as a T. It panics when v is not a T, since a generated
// migration keeps each variant's kind.
func as[T any](v any) T {
	t, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("migrated variant is a %T, not a %T", v, t))
	}
	return t
}
