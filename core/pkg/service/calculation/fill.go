// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

// FillOf returns the fill of the channel that c writes. c must be valid.
func FillOf(c Calculation) channel.Fill {
	return channel.Fill{
		Expression: c.Expression,
		Operations: ToChannelOperations(c.Operations),
	}
}

// Fills returns the fill of each channel among keys that a calculation writes, keyed by
// that channel, as snap sees them.
func (s *Service) Fills(
	ctx context.Context,
	snap gorp.Snapshot,
	keys ...channel.Key,
) (map[channel.Key]channel.Fill, error) {
	calcs, err := s.Outputs(ctx, snap, keys...)
	if err != nil || len(calcs) == 0 {
		return nil, err
	}
	fills := make(map[channel.Key]channel.Fill, len(calcs))
	for key, c := range calcs {
		fills[key] = FillOf(c)
	}
	return fills, nil
}
