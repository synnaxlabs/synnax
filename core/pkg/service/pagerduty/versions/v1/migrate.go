// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp/migrate"
	status "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
)

// MigrateStatusKeys points each alert at the key that the status migration gave the
// status it watches.
func MigrateStatusKeys(_ context.Context, cfg TaskConfig) (TaskConfig, error) {
	for i := range cfg.Alerts {
		cfg.Alerts[i].Status, _ = status.MigrateKey(cfg.Alerts[i].Status)
	}
	return cfg, nil
}

// StatusKeysMigration runs MigrateStatusKeys on every stored alert config.
var StatusKeysMigration = migrate.NewEntry[uuid.UUID, uuid.UUID](
	"v61_status_owner_keys",
	MigrateStatusKeys,
)
