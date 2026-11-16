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
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v1 "github.com/synnaxlabs/synnax/pkg/service/pagerduty/versions/v1"
	"github.com/synnaxlabs/x/errors"
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

// PlainRoutingKeyKVPrefix is the KV key prefix under which Migration stages the plain
// routing key of each config. The rest of the key is the config key as a string, and
// the value is the routing key.
const PlainRoutingKeyKVPrefix = "sy_pagerduty_plain_routing_key/"

// PlainRoutingKeyKVKey returns the staging KV key of the plain routing key of the
// config with the given key.
func PlainRoutingKeyKVKey(key uuid.UUID) []byte {
	return []byte(PlainRoutingKeyKVPrefix + key.String())
}

// Migration lifts each stored v1 config to v2. The v2 shape stores no plain routing
// key, so Migration stages each one under PlainRoutingKeyKVPrefix. Its key is the one
// that Core v0.62.0-rc.1 recorded for this lift, so a store of that candidate skips it.
var Migration = migrate.New(
	"v61_pagerduty_routing_key_secret",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var olds []v1.TaskConfig
		if err := gorp.NewRetrieve[uuid.UUID, v1.TaskConfig]().
			Entries(&olds).
			Exec(ctx, tx); err != nil || len(olds) == 0 {
			return err
		}
		news := make([]TaskConfig, len(olds))
		for i, old := range olds {
			cfg, err := MigrateTaskConfig(ctx, old)
			if err != nil {
				return errors.Wrapf(err, "config %s", old.Key)
			}
			if cfg.PlainRoutingKey != nil {
				if err = tx.Set(
					ctx,
					PlainRoutingKeyKVKey(cfg.Key),
					[]byte(*cfg.PlainRoutingKey),
				); err != nil {
					return err
				}
			}
			news[i] = cfg
		}
		return gorp.WrapWriter[uuid.UUID, TaskConfig](tx).Set(ctx, news...)
	},
)
