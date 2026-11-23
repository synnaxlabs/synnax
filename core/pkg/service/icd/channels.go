// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
)

// ReadChannels returns the v0.Channels that reads channels from svc.
func ReadChannels(svc *channel.Service) v0.Channels {
	return func(
		ctx context.Context,
		snap gorp.Snapshot,
		keys []channel.Key,
	) (map[channel.Key]v0.Channel, error) {
		var found []channel.Channel
		if err := svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&found).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		return lo.SliceToMap(
			found,
			func(c channel.Channel) (channel.Key, v0.Channel) {
				return c.Key(), v0.Channel{
					Name:     c.Name,
					DataType: c.DataType,
					Index:    c.IsIndex,
				}
			},
		), nil
	}
}
