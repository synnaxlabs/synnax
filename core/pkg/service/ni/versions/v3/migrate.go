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
	"encoding/json/v2"
	"math"
	"reflect"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	"github.com/synnaxlabs/x/errors"
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

// MigrateBaseAIChannel lifts a v2 analog input base to v3. A negative port becomes 0
// and disables the channel.
func MigrateBaseAIChannel(
	ctx context.Context,
	old v2.BaseAIChannel,
) (BaseAIChannel, error) {
	base, err := autoMigrateBaseAIChannel(ctx, old)
	var clamped bool
	base.Port, clamped = narrowPort(old.Port)
	base.Disabled = base.Disabled || clamped
	return base, err
}

// MigrateBaseAOChannel lifts a v2 analog output base to v3. A negative port becomes 0
// and disables the channel.
func MigrateBaseAOChannel(
	ctx context.Context,
	old v2.BaseAOChannel,
) (BaseAOChannel, error) {
	base, err := autoMigrateBaseAOChannel(ctx, old)
	var clamped bool
	base.Port, clamped = narrowPort(old.Port)
	base.Disabled = base.Disabled || clamped
	return base, err
}

// MigrateBaseCIChannel lifts a v2 counter input base to v3. A negative port becomes 0
// and disables the channel.
func MigrateBaseCIChannel(
	ctx context.Context,
	old v2.BaseCIChannel,
) (BaseCIChannel, error) {
	base, err := autoMigrateBaseCIChannel(ctx, old)
	var clamped bool
	base.Port, clamped = narrowPort(old.Port)
	base.Disabled = base.Disabled || clamped
	return base, err
}

// MigrateCJC lifts a v2 cold-junction source to v3. A negative port becomes 0.
// MigrateAnalogReadConfig disables the channel that holds it.
func MigrateCJC(ctx context.Context, old v2.CJC) (CJC, error) {
	cjc, err := autoMigrateCJC(ctx, old)
	if c, ok := old.Variant.(v2.ChanCJC); ok {
		port, _ := narrowPort(c.Port)
		cjc.Variant = ChanCJC{Port: port}
	}
	return cjc, err
}

// MigrateCIChannel lifts a v2 counter input channel to v3. A divisor and a pulse count
// below 1 become 1, a negative initial count becomes 0, and a fraction truncates.
func MigrateCIChannel(ctx context.Context, old v2.CIChannel) (CIChannel, error) {
	ch, err := autoMigrateCIChannel(ctx, old)
	if err != nil {
		return ch, err
	}
	switch o := old.Variant.(type) {
	case v2.CIFrequencyChannel:
		n := ch.Variant.(CIFrequencyChannel)
		n.Divisor = uint32(max(o.Divisor, 1))
		ch.Variant = n
	case v2.CIPeriodChannel:
		n := ch.Variant.(CIPeriodChannel)
		n.Divisor = uint32(max(o.Divisor, 1))
		ch.Variant = n
	case v2.CIEdgeCountChannel:
		n := ch.Variant.(CIEdgeCountChannel)
		n.InitialCount = narrowCount(o.InitialCount, 0)
		ch.Variant = n
	case v2.CIVelocityAngularChannel:
		n := ch.Variant.(CIVelocityAngularChannel)
		n.PulsesPerRev = narrowCount(o.PulsesPerRev, 1)
		ch.Variant = n
	case v2.CIPositionAngularChannel:
		n := ch.Variant.(CIPositionAngularChannel)
		n.PulsesPerRev = narrowCount(o.PulsesPerRev, 1)
		ch.Variant = n
	}
	return ch, nil
}

// MigrateDIChannel lifts a v2 digital input channel to v3. A port or line out of range
// clamps and disables the channel.
func MigrateDIChannel(ctx context.Context, old v2.DIChannel) (DIChannel, error) {
	ch, err := autoMigrateDIChannel(ctx, old)
	var portClamped, lineClamped bool
	ch.Port, portClamped = narrowPort(old.Port)
	ch.Line, lineClamped = narrowLine(old.Line)
	ch.Disabled = ch.Disabled || portClamped || lineClamped
	return ch, err
}

// MigrateDOChannel lifts a v2 digital output channel to v3. A port or line out of range
// clamps and disables the channel.
func MigrateDOChannel(ctx context.Context, old v2.DOChannel) (DOChannel, error) {
	ch, err := autoMigrateDOChannel(ctx, old)
	var portClamped, lineClamped bool
	ch.Port, portClamped = narrowPort(old.Port)
	ch.Line, lineClamped = narrowLine(old.Line)
	ch.Disabled = ch.Disabled || portClamped || lineClamped
	return ch, err
}

// MigrateAnalogReadConfig lifts a v2 analog read config to v3. A thermocouple whose
// cold-junction port clamps is disabled.
func MigrateAnalogReadConfig(
	ctx context.Context,
	old v2.AnalogReadConfig,
) (AnalogReadConfig, error) {
	cfg, err := autoMigrateAnalogReadConfig(ctx, old)
	if err != nil {
		return cfg, err
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

// narrowedFields are the channel fields a migration can change, with the CJC port as
// "cjc.port".
var narrowedFields = []string{
	"port",
	"line",
	"divisor",
	"initial_count",
	"pulses_per_rev",
	"cjc.port",
	"disabled",
}

// channelFields decodes the channels of a config into JSON objects, with the CJC port
// lifted to "cjc.port".
func channelFields(cfg any) ([]map[string]any, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var view struct {
		Channels []map[string]any `json:"channels"`
	}
	if err = json.Unmarshal(b, &view); err != nil {
		return nil, err
	}
	for _, ch := range view.Channels {
		if cjc, ok := ch["cjc"].(map[string]any); ok {
			ch["cjc.port"] = cjc["port"]
		}
	}
	return view.Channels, nil
}

// logged wraps a config migration so that it logs each channel field it changes.
func logged[O gorp.Entry[uuid.UUID], N any](
	migrateFn func(context.Context, O) (N, error),
) func(context.Context, O, alamos.Instrumentation) (N, error) {
	return func(ctx context.Context, old O, ins alamos.Instrumentation) (N, error) {
		migrated, err := migrateFn(ctx, old)
		if err != nil {
			return migrated, err
		}
		oldFields, err := channelFields(old)
		if err != nil {
			return migrated, err
		}
		newFields, err := channelFields(migrated)
		if err != nil {
			return migrated, err
		}
		if len(oldFields) != len(newFields) {
			return migrated, errors.Newf(
				"migrated %d channels into %d", len(oldFields), len(newFields),
			)
		}
		for i, o := range oldFields {
			for _, f := range narrowedFields {
				if reflect.DeepEqual(o[f], newFields[i][f]) {
					continue
				}
				ins.L.Warn(
					"narrowed an NI channel field",
					zap.Stringer("config", old.GorpKey()),
					zap.Any("channel", o["key"]),
					zap.String("field", f),
					zap.Any("old", o[f]),
					zap.Any("new", newFields[i][f]),
				)
			}
		}
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
