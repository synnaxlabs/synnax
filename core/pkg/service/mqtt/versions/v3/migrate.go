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
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
)

// MigrateReadConfig lifts a v2 read config to v3. It keeps the inline entries, for
// ResolveReadConfig to add back with an ICD.
func MigrateReadConfig(ctx context.Context, old v2.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	cfg.Inline = old.Inline
	return cfg, nil
}

// ReadMigration lifts stored read configs from v2 to v3, so that a binding with an
// empty data type has none.
var ReadMigration = migrate.NewEntry(
	"v63_mqtt_read_binding_data_type",
	MigrateReadConfig,
)
