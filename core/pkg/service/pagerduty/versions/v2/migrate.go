// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
)

// MigrateTaskConfig lifts a v1 task config to v2. A routing key moves to
// PlainRoutingKey, which ResolveTaskConfig turns into a secret.
func MigrateTaskConfig(ctx context.Context, old v1.TaskConfig) (TaskConfig, error) {
	cfg, err := autoMigrateTaskConfig(ctx, old)
	if err != nil {
		return TaskConfig{}, err
	}
	if old.RoutingKey != "" {
		cfg.PlainRoutingKey = &old.RoutingKey
	}
	return cfg, nil
}
