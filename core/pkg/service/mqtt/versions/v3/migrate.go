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

	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
)

// MigrateReadConfig lifts a v2 read config to v3. It keeps the inline entries, for
// ResolveReadConfig to add back with an ICD.
func MigrateReadConfig(ctx context.Context, old v2.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	cfg.Inline = old.Inline
	return cfg, err
}
