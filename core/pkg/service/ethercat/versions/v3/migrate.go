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

	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v2"
)

// MigrateReadConfig lifts a v2 read config to v3. Each entry channel applies no scale.
func MigrateReadConfig(ctx context.Context, old v2.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	for i, ch := range cfg.Channels {
		if entry, ok := ch.Variant.(EntryReadChannel); ok {
			entry.Scale = Scale{Variant: NoneScale{}}
			cfg.Channels[i].Variant = entry
		}
	}
	return cfg, nil
}

// ReadMigration lifts stored read configs to v3.
var ReadMigration = migrate.NewEntry("v61_ethercat_read_scale", MigrateReadConfig)
