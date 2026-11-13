// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"
	"fmt"
	"math"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	"go.uber.org/zap"
)

// narrowPort clamps a port to [0, math.MaxUint16] and reports whether it changed.
func narrowPort(old uint32) (uint16, bool) {
	port := min(old, math.MaxUint16)
	return uint16(port), port != old
}

// MigrateBaseAIChannel lifts a v3 analog input base to v4. A port above 65535 clamps
// and disables the channel.
func MigrateBaseAIChannel(
	ctx context.Context,
	old v3.BaseAIChannel,
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

// MigrateBaseAOChannel lifts a v3 analog output base to v4. A port above 65535 clamps
// and disables the channel.
func MigrateBaseAOChannel(
	ctx context.Context,
	old v3.BaseAOChannel,
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

// MigrateBaseCIChannel lifts a v3 counter input base to v4. A port above 65535 clamps
// and disables the channel.
func MigrateBaseCIChannel(
	ctx context.Context,
	old v3.BaseCIChannel,
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

// MigrateCJC lifts a v3 cold-junction source to v4. A port above 65535 clamps.
// MigrateAnalogReadConfig disables the channel that holds it.
func MigrateCJC(ctx context.Context, old v3.CJC) (CJC, error) {
	cjc, err := autoMigrateCJC(ctx, old)
	if err != nil {
		return CJC{}, err
	}
	if c, ok := old.Variant.(v3.ChanCJC); ok {
		port, _ := narrowPort(c.Port)
		cjc.Variant = ChanCJC{Port: port}
	}
	return cjc, nil
}

// MigrateTerminal lifts a v3 terminal to v4. The NI device schema now defines its
// terminal configuration, with the same values.
func MigrateTerminal(ctx context.Context, old v3.Terminal) (Terminal, error) {
	return autoMigrateTerminal(ctx, old)
}

// MigrateDIChannel lifts a v3 digital input channel to v4. A port above 65535 clamps
// and disables the channel.
func MigrateDIChannel(ctx context.Context, old v3.DIChannel) (DIChannel, error) {
	ch, err := autoMigrateDIChannel(ctx, old)
	if err != nil {
		return DIChannel{}, err
	}
	var clamped bool
	ch.Port, clamped = narrowPort(old.Port)
	ch.Disabled = ch.Disabled || clamped
	return ch, nil
}

// MigrateDOChannel lifts a v3 digital output channel to v4. A port above 65535 clamps
// and disables the channel.
func MigrateDOChannel(ctx context.Context, old v3.DOChannel) (DOChannel, error) {
	ch, err := autoMigrateDOChannel(ctx, old)
	if err != nil {
		return DOChannel{}, err
	}
	var clamped bool
	ch.Port, clamped = narrowPort(old.Port)
	ch.Disabled = ch.Disabled || clamped
	return ch, nil
}

// MigrateAnalogReadConfig lifts a v3 analog read config to v4. A thermocouple whose
// cold-junction port clamps is disabled.
func MigrateAnalogReadConfig(
	ctx context.Context,
	old v3.AnalogReadConfig,
) (AnalogReadConfig, error) {
	cfg, err := autoMigrateAnalogReadConfig(ctx, old)
	if err != nil {
		return AnalogReadConfig{}, err
	}
	for i, ch := range old.Channels {
		tc, ok := ch.Variant.(v3.AIThermocoupleChannel)
		if !ok {
			continue
		}
		c, ok := tc.Cjc.Variant.(v3.ChanCJC)
		if !ok {
			continue
		}
		if _, clamped := narrowPort(c.Port); clamped {
			n := cfg.Channels[i].Variant.(AIThermocoupleChannel)
			n.Disabled = true
			cfg.Channels[i].Variant = n
		}
	}
	return cfg, nil
}

// MigrateAnalogWriteConfig lifts a v3 analog write config to v4. A function generator
// frequency that is not a positive finite number becomes 1 Hz.
func MigrateAnalogWriteConfig(
	ctx context.Context,
	old v3.AnalogWriteConfig,
) (AnalogWriteConfig, error) {
	cfg, err := autoMigrateAnalogWriteConfig(ctx, old)
	if err != nil {
		return AnalogWriteConfig{}, err
	}
	for i, ch := range cfg.Channels {
		v, ok := ch.Variant.(AOFuncGenChannel)
		if ok && (!(v.Frequency > 0) || math.IsInf(v.Frequency, 1)) {
			v.Frequency = 1
			cfg.Channels[i].Variant = v
		}
	}
	return cfg, nil
}

// MigrateCounterReadConfig lifts a v3 counter read config to v4.
func MigrateCounterReadConfig(
	ctx context.Context,
	old v3.CounterReadConfig,
) (CounterReadConfig, error) {
	return autoMigrateCounterReadConfig(ctx, old)
}

// MigrateDigitalReadConfig lifts a v3 digital read config to v4.
func MigrateDigitalReadConfig(
	ctx context.Context,
	old v3.DigitalReadConfig,
) (DigitalReadConfig, error) {
	return autoMigrateDigitalReadConfig(ctx, old)
}

// MigrateDigitalWriteConfig lifts a v3 digital write config to v4.
func MigrateDigitalWriteConfig(
	ctx context.Context,
	old v3.DigitalWriteConfig,
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
	~uint16 | ~uint32 | ~float64
}

// logNumber logs field when its old and new values differ. A NaN old value always
// differs.
func logNumber[O, N number](l changeLog, field string, old O, new N) {
	if float64(old) != float64(new) {
		l.warn(
			field,
			zap.Float64("old", float64(old)),
			zap.Float64("new", float64(new)),
		)
	}
}

// logFlag logs field when its old and new values differ.
func logFlag(l changeLog, field string, old, new bool) {
	if old != new {
		l.warn(field, zap.Bool("old", old), zap.Bool("new", new))
	}
}

// aiBases returns the shared fields of a v3 analog input variant and of its migration.
// It panics for a variant without shared fields.
func aiBases(
	old v3.AIChannelVariant,
	n AIChannelVariant,
) (v3.BaseAIChannel, BaseAIChannel) {
	switch o := old.(type) {
	case v3.AIVoltageChannel:
		return o.BaseAIChannel, n.(AIVoltageChannel).BaseAIChannel
	case v3.AIAccelChannel:
		return o.BaseAIChannel, n.(AIAccelChannel).BaseAIChannel
	case v3.AIBridgeChannel:
		return o.BaseAIChannel, n.(AIBridgeChannel).BaseAIChannel
	case v3.AICurrentChannel:
		return o.BaseAIChannel, n.(AICurrentChannel).BaseAIChannel
	case v3.AIForceBridgeTableChannel:
		return o.BaseAIChannel, n.(AIForceBridgeTableChannel).BaseAIChannel
	case v3.AIForceBridgeTwoPointLinChannel:
		return o.BaseAIChannel, n.(AIForceBridgeTwoPointLinChannel).BaseAIChannel
	case v3.AIForceIEPEChannel:
		return o.BaseAIChannel, n.(AIForceIEPEChannel).BaseAIChannel
	case v3.AIMicrophoneChannel:
		return o.BaseAIChannel, n.(AIMicrophoneChannel).BaseAIChannel
	case v3.AIPressureBridgeTableChannel:
		return o.BaseAIChannel, n.(AIPressureBridgeTableChannel).BaseAIChannel
	case v3.AIPressureBridgeTwoPointLinChannel:
		return o.BaseAIChannel, n.(AIPressureBridgeTwoPointLinChannel).BaseAIChannel
	case v3.AIResistanceChannel:
		return o.BaseAIChannel, n.(AIResistanceChannel).BaseAIChannel
	case v3.AIRTDChannel:
		return o.BaseAIChannel, n.(AIRTDChannel).BaseAIChannel
	case v3.AIStrainGaugeChannel:
		return o.BaseAIChannel, n.(AIStrainGaugeChannel).BaseAIChannel
	case v3.AIThermocoupleChannel:
		return o.BaseAIChannel, n.(AIThermocoupleChannel).BaseAIChannel
	case v3.AITorqueBridgeTableChannel:
		return o.BaseAIChannel, n.(AITorqueBridgeTableChannel).BaseAIChannel
	case v3.AITorqueBridgeTwoPointLinChannel:
		return o.BaseAIChannel, n.(AITorqueBridgeTwoPointLinChannel).BaseAIChannel
	case v3.AIVelocityIEPEChannel:
		return o.BaseAIChannel, n.(AIVelocityIEPEChannel).BaseAIChannel
	case v3.AIAccel4WireDCVoltageChannel:
		return o.BaseAIChannel, n.(AIAccel4WireDCVoltageChannel).BaseAIChannel
	case v3.AIAccelChargeChannel:
		return o.BaseAIChannel, n.(AIAccelChargeChannel).BaseAIChannel
	case v3.AIChargeChannel:
		return o.BaseAIChannel, n.(AIChargeChannel).BaseAIChannel
	case v3.AICurrentRMSChannel:
		return o.BaseAIChannel, n.(AICurrentRMSChannel).BaseAIChannel
	case v3.AIForceBridgePolynomialChannel:
		return o.BaseAIChannel, n.(AIForceBridgePolynomialChannel).BaseAIChannel
	case v3.AIFreqVoltageChannel:
		return o.BaseAIChannel, n.(AIFreqVoltageChannel).BaseAIChannel
	case v3.AIPressureBridgePolynomialChannel:
		return o.BaseAIChannel, n.(AIPressureBridgePolynomialChannel).BaseAIChannel
	case v3.AIThermistorIexChannel:
		return o.BaseAIChannel, n.(AIThermistorIexChannel).BaseAIChannel
	case v3.AIThermistorVexChannel:
		return o.BaseAIChannel, n.(AIThermistorVexChannel).BaseAIChannel
	case v3.AITorqueBridgePolynomialChannel:
		return o.BaseAIChannel, n.(AITorqueBridgePolynomialChannel).BaseAIChannel
	case v3.AIVoltageRMSChannel:
		return o.BaseAIChannel, n.(AIVoltageRMSChannel).BaseAIChannel
	case v3.AIVoltageWithExcitChannel:
		return o.BaseAIChannel, n.(AIVoltageWithExcitChannel).BaseAIChannel
	default:
		panic(fmt.Sprintf("analog input variant %T has no base", old))
	}
}

func logAIChannel(l changeLog, old v3.AIChannel, migrated AIChannel) {
	if o, ok := old.Variant.(v3.AITempBuiltinChannel); ok {
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
	tc, ok := old.Variant.(v3.AIThermocoupleChannel)
	if !ok {
		return
	}
	oldCJC, ok := tc.Cjc.Variant.(v3.ChanCJC)
	if !ok {
		return
	}
	newCJC := migrated.Variant.(AIThermocoupleChannel).Cjc.Variant.(ChanCJC)
	logNumber(l, "Cjc.Port", oldCJC.Port, newCJC.Port)
}

func logAOChannel(l changeLog, old v3.AOChannel, migrated AOChannel) {
	if old.Variant == nil {
		return
	}
	o, n := old.Base(), migrated.Base()
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	if gen, ok := old.Variant.(v3.AOFuncGenChannel); ok {
		after := migrated.Variant.(AOFuncGenChannel).Frequency
		logNumber(l, "Frequency", gen.Frequency, after)
	}
	logFlag(l, "Disabled", o.Disabled, n.Disabled)
}

func logCIChannel(l changeLog, old v3.CIChannel, migrated CIChannel) {
	if old.Variant == nil {
		return
	}
	o, n := old.Base(), migrated.Base()
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	logFlag(l, "Disabled", o.Disabled, n.Disabled)
}

func logDIChannel(l changeLog, old v3.DIChannel, migrated DIChannel) {
	l.channel = old.Key
	logNumber(l, "Port", old.Port, migrated.Port)
	logFlag(l, "Disabled", old.Disabled, migrated.Disabled)
}

func logDOChannel(l changeLog, old v3.DOChannel, migrated DOChannel) {
	l.channel = old.Key
	logNumber(l, "Port", old.Port, migrated.Port)
	logFlag(l, "Disabled", old.Disabled, migrated.Disabled)
}

func logAnalogRead(l changeLog, old v3.AnalogReadConfig, migrated AnalogReadConfig) {
	for i, ch := range old.Channels {
		logAIChannel(l, ch, migrated.Channels[i])
	}
}

func logAnalogWrite(l changeLog, old v3.AnalogWriteConfig, migrated AnalogWriteConfig) {
	for i, ch := range old.Channels {
		logAOChannel(l, ch, migrated.Channels[i])
	}
}

func logCounterRead(l changeLog, old v3.CounterReadConfig, migrated CounterReadConfig) {
	for i, ch := range old.Channels {
		logCIChannel(l, ch, migrated.Channels[i])
	}
}

func logDigitalRead(l changeLog, old v3.DigitalReadConfig, migrated DigitalReadConfig) {
	for i, ch := range old.Channels {
		logDIChannel(l, ch, migrated.Channels[i])
	}
}

func logDigitalWrite(
	l changeLog,
	old v3.DigitalWriteConfig,
	migrated DigitalWriteConfig,
) {
	for i, ch := range old.Channels {
		logDOChannel(l, ch, migrated.Channels[i])
	}
}

// logged wraps a config migration so that it logs each channel field it changes.
func logged[O, N gorp.Entry[uuid.UUID]](
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

const migrationKey = "v62_ni_task_v4"

// AnalogReadMigration lifts stored analog read configs to v4.
var AnalogReadMigration = migrate.NewInstrumentedEntry(
	migrationKey,
	logged(MigrateAnalogReadConfig, logAnalogRead),
)

// AnalogWriteMigration lifts stored analog write configs to v4.
var AnalogWriteMigration = migrate.NewInstrumentedEntry(
	migrationKey,
	logged(MigrateAnalogWriteConfig, logAnalogWrite),
)

// CounterReadMigration lifts stored counter read configs to v4.
var CounterReadMigration = migrate.NewInstrumentedEntry(
	migrationKey,
	logged(MigrateCounterReadConfig, logCounterRead),
)

// DigitalReadMigration lifts stored digital read configs to v4.
var DigitalReadMigration = migrate.NewInstrumentedEntry(
	migrationKey,
	logged(MigrateDigitalReadConfig, logDigitalRead),
)

// DigitalWriteMigration lifts stored digital write configs to v4.
var DigitalWriteMigration = migrate.NewInstrumentedEntry(
	migrationKey,
	logged(MigrateDigitalWriteConfig, logDigitalWrite),
)
