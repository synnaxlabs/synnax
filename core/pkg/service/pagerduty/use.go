// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/set"
)

var _ secret.Finder = FindSecretUses

// FindSecretUses is a secret.Finder that returns each alert task whose routing key is
// one of the given secrets. A use gives no rack access to the key, because the Core
// sends the alerts.
func FindSecretUses(
	ctx context.Context,
	tx gorp.Tx,
	keys []secret.Key,
) ([]secret.Use, error) {
	wanted := set.New(keys...)
	var configs []TaskConfig
	if err := gorp.NewRetrieve[uuid.UUID, TaskConfig]().
		Where(gorp.Match(func(_ gorp.Context, c *TaskConfig) (bool, error) {
			return c.RoutingKey != nil && wanted.Contains(*c.RoutingKey), nil
		})).
		Entries(&configs).
		Exec(ctx, tx); err != nil || len(configs) == 0 {
		return nil, err
	}
	var tasks []task.Task
	if err := gorp.NewRetrieve[task.Key, task.Task]().
		Where(gorp.MatchKeys[task.Key, task.Task](
			lo.Map(configs, func(c TaskConfig, _ int) task.Key { return c.Key })...,
		)).
		Entries(&tasks).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	byKey := lo.KeyBy(tasks, func(t task.Task) task.Key { return t.Key })
	uses := make([]secret.Use, len(configs))
	for i, c := range configs {
		t := byKey[c.Key]
		uses[i] = secret.Use{
			Secret:   *c.RoutingKey,
			Resource: t.OntologyID(),
			Name:     t.Name,
		}
	}
	return uses, nil
}
