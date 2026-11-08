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

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
)

// UniqueChannelName generates a random channel name that should be unique. Channel name
// uniqueness itself is enforced by the channel service during creation.
func UniqueChannelName() string {
	return fmt.Sprintf("test_ch_%09d", rand.IntN(999999999))
}

// CreateVirtual creates count virtual channels with unique names through svc and
// returns their keys in creation order.
func CreateVirtual(
	ctx context.Context,
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
	if err := svc.NewWriter(nil).CreateMany(ctx, &channels); err != nil {
		return nil, err
	}
	return channel.KeysFromChannels(channels), nil
}
