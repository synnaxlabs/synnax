// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"

	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/task/versions/v4"
	"github.com/synnaxlabs/x/gorp"
)

// Migration lifts stored Arc task configs from v4 to v5. The module hash maps start
// empty and fill on the next sync.
var Migration = gorp.NewEntryMigration(
	"v60_module_hashes",
	func(ctx context.Context, old v4.Config) (Config, error) {
		cfg, err := autoMigrateConfig(ctx, old)
		if err != nil {
			return Config{}, err
		}
		cfg.Modules, cfg.DeployedModules = []ModuleHash{}, []ModuleHash{}
		return cfg, nil
	},
)
