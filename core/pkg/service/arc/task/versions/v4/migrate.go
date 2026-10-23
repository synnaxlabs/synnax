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

	v3 "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions/v3"
	"github.com/synnaxlabs/x/gorp"
)

// Migration lifts stored Arc task configs from v3 to v4, converting the loop mode to
// a performance level.
var Migration = gorp.NewEntryMigration(
	"v59_performance",
	func(ctx context.Context, old v3.Config) (Config, error) {
		cfg, err := autoMigrateConfig(ctx, old)
		if err != nil {
			return Config{}, err
		}
		cfg.Performance = PerformanceOf(old.ExecutionMode)
		return cfg, nil
	},
)

// performances maps each v3 loop mode to the level that selects it in v4.
var performances = map[v3.ExecutionMode]Performance{
	v3.ExecutionModeAuto:        PerformanceAuto,
	v3.ExecutionModeEventDriven: PerformanceLow,
	v3.ExecutionModeHybrid:      PerformanceMedium,
	v3.ExecutionModeRtEvent:     PerformanceMedium,
	v3.ExecutionModeHighRate:    PerformanceHigh,
	v3.ExecutionModeBusyWait:    PerformanceHigh,
}

// PerformanceOf returns the level for a v3 loop mode. An unknown mode passes through
// unchanged, so validation rejects it.
func PerformanceOf(mode v3.ExecutionMode) Performance {
	if p, ok := performances[mode]; ok {
		return p
	}
	return Performance(mode)
}
