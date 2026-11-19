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

	"github.com/synnaxlabs/gorp"
	secret "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v0"
)

// CreateSecret creates a stored secret with the given key, name, and value through tx.
type CreateSecret func(
	ctx context.Context,
	tx gorp.Tx,
	key secret.Key,
	name string,
	value []byte,
) error

// ResolveTaskConfig points RoutingKey at a new secret that holds the plain routing key
// of cfg, and clears the plain key. It returns the write that creates the secret,
// named "<name> routing key", where name is the name of the task that owns cfg. It
// returns nil for a config without a plain routing key.
func ResolveTaskConfig(
	create CreateSecret,
	name string,
	cfg *TaskConfig,
) func(context.Context, gorp.Tx) error {
	if cfg.PlainRoutingKey == nil {
		return nil
	}
	key, value := uuid.New(), []byte(*cfg.PlainRoutingKey)
	cfg.RoutingKey, cfg.PlainRoutingKey = &key, nil
	return func(ctx context.Context, tx gorp.Tx) error {
		return create(ctx, tx, key, name+" routing key", value)
	}
}
