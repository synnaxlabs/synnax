// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"context"

	"github.com/synnaxlabs/cesium/inspect"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/storage"
)

// storageKeys converts channel keys to their storage layer form.
func storageKeys(keys []channel.Key) []inspect.ChannelKey {
	out := make([]inspect.ChannelKey, len(keys))
	for i, k := range keys {
		out[i] = k.StorageKey()
	}
	return out
}

// runTS inspects the time-series store under the data directory. A directory with no
// time-series store reports no channels rather than failing: a Core that never wrote
// telemetry creates it lazily.
func runTS(ctx context.Context, cfg Config) (*inspect.Report, error) {
	fs, err := cfg.FS.Sub(cfg.Dirname)
	if err != nil {
		return nil, err
	}
	exists, err := fs.Exists(storage.TSDirName)
	if err != nil || !exists {
		return &inspect.Report{}, err
	}
	if fs, err = fs.Sub(storage.TSDirName); err != nil {
		return nil, err
	}
	report, err := inspect.Run(ctx, inspect.Config{
		Instrumentation: cfg.Instrumentation,
		FS:              fs,
		Deep:            cfg.Deep,
		Channels:        storageKeys(cfg.Channels),
		Progress: func(done, total int) {
			progress(cfg, PhaseTS, done, total)
		},
	})
	return &report, err
}
