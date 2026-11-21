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
	"math"
	"reflect"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	"github.com/synnaxlabs/x/errors"
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
			n, ok := cfg.Channels[i].Variant.(AIThermocoupleChannel)
			if !ok {
				return AnalogReadConfig{}, errVariantChanged(
					tc.Key,
					cfg.Channels[i].Variant,
				)
			}
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

func (l changeLog) warn(field string, before, after zap.Field) {
	l.ins.L.Warn(
		"narrowed an NI channel field",
		zap.Stringer("config", l.config),
		zap.String("channel", l.channel),
		zap.String("field", field),
		before,
		after,
	)
}

type number interface {
	~uint16 | ~uint32 | ~float64
}

// logNumber logs field when its old and new values differ. A NaN old value always
// differs.
func logNumber[O, N number](l changeLog, field string, before O, after N) {
	if float64(before) != float64(after) {
		l.warn(
			field,
			zap.Float64("old", float64(before)),
			zap.Float64("new", float64(after)),
		)
	}
}

// logDisabled logs the Disabled field when its old and new values differ.
func logDisabled(l changeLog, before, after bool) {
	if before != after {
		l.warn("Disabled", zap.Bool("old", before), zap.Bool("new", after))
	}
}

func errVariantChanged(channel string, variant any) error {
	return errors.Newf("channel %s migrated to %T", channel, variant)
}

// aiBase returns the BaseAIChannel that an analog input variant embeds, or false for a
// variant without one. Every variant but the built-in temperature channel embeds one.
func aiBase[B any](variant any) (B, bool) {
	v := reflect.ValueOf(variant)
	if v.Kind() != reflect.Struct {
		var zero B
		return zero, false
	}
	field := v.FieldByName("BaseAIChannel")
	if !field.IsValid() {
		var zero B
		return zero, false
	}
	return reflect.TypeAssert[B](field)
}

func logAIChannel(l changeLog, old v3.AIChannel, migrated AIChannel) error {
	if o, ok := old.Variant.(v3.AITempBuiltinChannel); ok {
		n, ok := migrated.Variant.(AITempBuiltinChannel)
		if !ok {
			return errVariantChanged(o.Key, migrated.Variant)
		}
		l.channel = o.Key
		logDisabled(l, o.Disabled, n.Disabled)
		return nil
	}
	if old.Variant == nil {
		return nil
	}
	o, ok := aiBase[v3.BaseAIChannel](old.Variant)
	if !ok {
		return errors.Newf("analog input variant %T has no base", old.Variant)
	}
	n, ok := aiBase[BaseAIChannel](migrated.Variant)
	if !ok {
		return errVariantChanged(o.Key, migrated.Variant)
	}
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	logDisabled(l, o.Disabled, n.Disabled)
	tc, ok := old.Variant.(v3.AIThermocoupleChannel)
	if !ok {
		return nil
	}
	oldCJC, ok := tc.Cjc.Variant.(v3.ChanCJC)
	if !ok {
		return nil
	}
	newTC, ok := migrated.Variant.(AIThermocoupleChannel)
	if !ok {
		return errVariantChanged(o.Key, migrated.Variant)
	}
	newCJC, ok := newTC.Cjc.Variant.(ChanCJC)
	if !ok {
		return errVariantChanged(o.Key, newTC.Cjc.Variant)
	}
	logNumber(l, "Cjc.Port", oldCJC.Port, newCJC.Port)
	return nil
}

func logAOChannel(l changeLog, old v3.AOChannel, migrated AOChannel) error {
	if old.Variant == nil {
		return nil
	}
	o, n := old.Base(), migrated.Base()
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	if gen, ok := old.Variant.(v3.AOFuncGenChannel); ok {
		after, ok := migrated.Variant.(AOFuncGenChannel)
		if !ok {
			return errVariantChanged(o.Key, migrated.Variant)
		}
		logNumber(l, "Frequency", gen.Frequency, after.Frequency)
	}
	logDisabled(l, o.Disabled, n.Disabled)
	return nil
}

func logCIChannel(l changeLog, old v3.CIChannel, migrated CIChannel) error {
	if old.Variant == nil {
		return nil
	}
	o, n := old.Base(), migrated.Base()
	l.channel = o.Key
	logNumber(l, "Port", o.Port, n.Port)
	logDisabled(l, o.Disabled, n.Disabled)
	return nil
}

func logDIChannel(l changeLog, old v3.DIChannel, migrated DIChannel) error {
	l.channel = old.Key
	logNumber(l, "Port", old.Port, migrated.Port)
	logDisabled(l, old.Disabled, migrated.Disabled)
	return nil
}

func logDOChannel(l changeLog, old v3.DOChannel, migrated DOChannel) error {
	l.channel = old.Key
	logNumber(l, "Port", old.Port, migrated.Port)
	logDisabled(l, old.Disabled, migrated.Disabled)
	return nil
}

// logChannels logs each channel of a config with logFn.
func logChannels[O, N any](
	l changeLog,
	old []O,
	migrated []N,
	logFn func(changeLog, O, N) error,
) error {
	for i, ch := range old {
		if err := logFn(l, ch, migrated[i]); err != nil {
			return err
		}
	}
	return nil
}

func logAnalogRead(
	l changeLog,
	old v3.AnalogReadConfig,
	migrated AnalogReadConfig,
) error {
	return logChannels(l, old.Channels, migrated.Channels, logAIChannel)
}

func logAnalogWrite(
	l changeLog,
	old v3.AnalogWriteConfig,
	migrated AnalogWriteConfig,
) error {
	return logChannels(l, old.Channels, migrated.Channels, logAOChannel)
}

func logCounterRead(
	l changeLog,
	old v3.CounterReadConfig,
	migrated CounterReadConfig,
) error {
	return logChannels(l, old.Channels, migrated.Channels, logCIChannel)
}

func logDigitalRead(
	l changeLog,
	old v3.DigitalReadConfig,
	migrated DigitalReadConfig,
) error {
	return logChannels(l, old.Channels, migrated.Channels, logDIChannel)
}

func logDigitalWrite(
	l changeLog,
	old v3.DigitalWriteConfig,
	migrated DigitalWriteConfig,
) error {
	return logChannels(l, old.Channels, migrated.Channels, logDOChannel)
}

// logged wraps a config migration so that it logs each channel field it changes.
func logged[O, N gorp.Entry[uuid.UUID]](
	migrateFn func(context.Context, O) (N, error),
	logFn func(changeLog, O, N) error,
) func(context.Context, O, alamos.Instrumentation) (N, error) {
	return func(ctx context.Context, old O, ins alamos.Instrumentation) (N, error) {
		migrated, err := migrateFn(ctx, old)
		if err != nil {
			var zero N
			return zero, err
		}
		l := changeLog{ins: ins, config: old.GorpKey()}
		if err = logFn(l, old, migrated); err != nil {
			var zero N
			return zero, err
		}
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
