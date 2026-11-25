// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides test helpers for working with channels.
package testutil

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
)

// UniqueChannelName generates a random channel name that should be unique. Channel name
// uniqueness itself is enforced by the channel service during creation.
func UniqueChannelName() string {
	return fmt.Sprintf("test_ch_%016x", rand.Uint64())
}

// CreateVirtual creates count virtual channels with unique names through svc in db and
// returns their keys in creation order.
func CreateVirtual(
	ctx context.Context,
	db *gorp.DB,
	svc *channel.Service,
	count int,
) (channel.Keys, error) {
	channels := make([]channel.Channel, count)
	for i := range channels {
		channels[i] = channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float64T,
			Virtual:  true,
		}
	}
	if err := db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.NewWriter(tx).CreateMany(ctx, &channels)
	}); err != nil {
		return nil, err
	}
	return channel.KeysFromChannels(channels), nil
}

// Delete deletes the channels with keys through svc in tx, as a delete with no picks
// does: it plans with svc.PlanDelete, then deletes with DeletePlanned. A built-in
// channel or a record that restricts a channel fails it.
func Delete(
	ctx context.Context,
	svc *channel.Service,
	tx gorp.Tx,
	keys ...channel.Key,
) error {
	removed, effects, err := svc.PlanDelete(ctx, tx, keys)
	if err != nil {
		return err
	}
	return svc.NewWriter(tx).DeletePlanned(ctx, removed, effects)
}
