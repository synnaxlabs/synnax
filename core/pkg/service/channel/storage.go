// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

// createStorage creates the storage channels for the provided channels, which carry
// the keys that the allocator assigned. Free channels have no storage registration and
// are skipped.
func (s *Service) createStorage(ctx context.Context, channels []channel.Channel) error {
	stored := lo.Reject(channels, func(c channel.Channel, _ int) bool {
		return c.Key().Free()
	})
	return s.cfg.TS.CreateChannel(
		ctx,
		lo.Map(stored, func(c channel.Channel, _ int) cesium.Channel {
			return c.Storage()
		})...,
	)
}

// deleteStorage deletes the storage channels for the provided keys. Free channels have
// no storage registration and are skipped. It does not touch channel metadata.
func (s *Service) deleteStorage(keys Keys) error {
	stored := lo.Reject(keys, func(key Key, _ int) bool { return key.Free() })
	return s.cfg.TS.DeleteChannels(stored.Uint32())
}

// renameStorage renames each storage channel keyed in renames to its corresponding new
// name. Free channels have no storage registration and are skipped.
func (s *Service) renameStorage(ctx context.Context, renames map[Key]string) error {
	stored := make(map[cesium.ChannelKey]string, len(renames))
	for key, name := range renames {
		if !key.Free() {
			stored[cesium.ChannelKey(key)] = name
		}
	}
	return s.cfg.TS.RenameChannels(ctx, stored)
}
