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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	"go.uber.org/zap"
)

// MigrateAnalogWriteConfig lifts a v3 analog write config to v4. A function generator
// frequency that is not positive, or NaN, becomes 1 Hz.
func MigrateAnalogWriteConfig(
	ctx context.Context,
	old v3.AnalogWriteConfig,
) (AnalogWriteConfig, error) {
	cfg, err := autoMigrateAnalogWriteConfig(ctx, old)
	if err != nil {
		return AnalogWriteConfig{}, err
	}
	for i, ch := range cfg.Channels {
		if v, ok := ch.Variant.(AOFuncGenChannel); ok && !(v.Frequency > 0) {
			v.Frequency = 1
			cfg.Channels[i].Variant = v
		}
	}
	return cfg, nil
}

// AnalogWriteMigration lifts stored analog write configs to v4 and logs each function
// generator frequency it changes.
var AnalogWriteMigration = migrate.NewInstrumentedEntry(
	"v62_positive_func_gen_frequency",
	func(
		ctx context.Context,
		old v3.AnalogWriteConfig,
		ins alamos.Instrumentation,
	) (AnalogWriteConfig, error) {
		cfg, err := MigrateAnalogWriteConfig(ctx, old)
		if err != nil {
			return AnalogWriteConfig{}, err
		}
		for i, ch := range old.Channels {
			o, ok := ch.Variant.(v3.AOFuncGenChannel)
			if !ok {
				continue
			}
			n := cfg.Channels[i].Variant.(AOFuncGenChannel)
			if n.Frequency == o.Frequency {
				continue
			}
			ins.L.Warn(
				"narrowed an NI channel field",
				zap.Stringer("config", old.Key),
				zap.String("channel", o.Key),
				zap.String("field", "Frequency"),
				zap.Float64("old", o.Frequency),
				zap.Float64("new", n.Frequency),
			)
		}
		return cfg, nil
	},
)
