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
	v2 "github.com/synnaxlabs/synnax/pkg/service/labjack/versions/v2"
	"go.uber.org/zap"
)

const (
	// ground is LJM_GND, which v2 stored as the negative channel of a single-ended
	// reading.
	ground = 199
	// maxNegChan is the last AIN port that a T7 takes as a negative channel.
	maxNegChan = 127
	// defaultTCPScanMultiplier is the schema default of tcp_scan_multiplier.
	defaultTCPScanMultiplier = 10
)

func storedNegChan(ch v2.ReadChannel) (int32, bool) {
	switch v := ch.Variant.(type) {
	case v2.AnalogReadChannel:
		return v.NegChan, true
	case v2.ThermocoupleReadChannel:
		return v.NegChan, true
	}
	return 0, false
}

// migrateNegChan maps ground to absent and clamps any other value into 0 to 127. The
// bool is true when the value clamps. v2 filled a 0 with ground, so 0 is ground too.
func migrateNegChan(old int32) (*uint16, bool) {
	if old == ground || old == 0 {
		return nil, false
	}
	clamped := min(max(old, 0), maxNegChan)
	return new(uint16(clamped)), clamped != old
}

// MigrateReadChannel lifts a v2 read channel to v3. A ground negative channel becomes
// absent. Any other negative channel outside 0 to 127 clamps into range and disables
// the channel, because the clamped port may not be the one the user wired.
func MigrateReadChannel(ctx context.Context, old v2.ReadChannel) (ReadChannel, error) {
	ch, err := autoMigrateReadChannel(ctx, old)
	if err != nil {
		return ReadChannel{}, err
	}
	stored, ok := storedNegChan(old)
	if !ok {
		return ch, nil
	}
	negChan, clamped := migrateNegChan(stored)
	switch v := ch.Variant.(type) {
	case AnalogReadChannel:
		v.NegChan = negChan
		v.Disabled = v.Disabled || clamped
		ch.Variant = v
	case ThermocoupleReadChannel:
		v.NegChan = negChan
		v.Disabled = v.Disabled || clamped
		ch.Variant = v
	}
	return ch, nil
}

// migrateBacklog maps a count of 0, which let the Driver pick, to absent.
func migrateBacklog(old uint32) *uint32 {
	if old == 0 {
		return nil
	}
	return new(old)
}

// MigrateReadConfig lifts a v2 read config to v3. A scan backlog count of 0 becomes
// absent.
func MigrateReadConfig(ctx context.Context, old v2.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	cfg.DeviceScanBacklogWarnOnCount = migrateBacklog(old.DeviceScanBacklogWarnOnCount)
	cfg.LjmScanBacklogWarnOnCount = migrateBacklog(old.LjmScanBacklogWarnOnCount)
	return cfg, nil
}

func migrateStoredReadConfig(
	ctx context.Context,
	old v2.ReadConfig,
	ins alamos.Instrumentation,
) (ReadConfig, error) {
	cfg, err := MigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	for _, ch := range old.Channels {
		stored, ok := storedNegChan(ch)
		if !ok {
			continue
		}
		if negChan, clamped := migrateNegChan(stored); clamped {
			ins.L.Warn(
				"clamped a LabJack negative channel and disabled its channel",
				zap.Stringer("config", old.Key),
				zap.String("channel", ch.Base().Key),
				zap.String("field", "neg_chan"),
				zap.Int32("old", stored),
				zap.Uint16("new", *negChan),
			)
		}
	}
	return cfg, nil
}

// ReadMigration lifts stored read configs to v3 and logs each clamped value.
var ReadMigration = migrate.NewInstrumentedEntry(
	"v61_labjack_read_narrow",
	migrateStoredReadConfig,
)

// MigrateScanConfig lifts a v2 scan config to v3. A TCP scan multiplier of 0 takes
// the default of 10, as v2 filled it, and a negative one clamps to 1.
func MigrateScanConfig(ctx context.Context, old v2.ScanConfig) (ScanConfig, error) {
	cfg, err := autoMigrateScanConfig(ctx, old)
	if err != nil {
		return ScanConfig{}, err
	}
	if old.TCPScanMultiplier == 0 {
		cfg.TCPScanMultiplier = defaultTCPScanMultiplier
	} else {
		cfg.TCPScanMultiplier = uint32(max(old.TCPScanMultiplier, 1))
	}
	return cfg, nil
}

func migrateStoredScanConfig(
	ctx context.Context,
	old v2.ScanConfig,
	ins alamos.Instrumentation,
) (ScanConfig, error) {
	cfg, err := MigrateScanConfig(ctx, old)
	if err != nil {
		return ScanConfig{}, err
	}
	if old.TCPScanMultiplier < 0 {
		ins.L.Warn(
			"clamped a LabJack TCP scan multiplier",
			zap.Stringer("config", old.Key),
			zap.String("field", "tcp_scan_multiplier"),
			zap.Int32("old", old.TCPScanMultiplier),
			zap.Uint32("new", cfg.TCPScanMultiplier),
		)
	}
	return cfg, nil
}

// ScanMigration lifts stored scan configs to v3 and logs each clamped value.
var ScanMigration = migrate.NewInstrumentedEntry(
	"v61_labjack_scan_narrow",
	migrateStoredScanConfig,
)
