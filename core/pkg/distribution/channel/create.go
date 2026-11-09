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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Allocate assigns local keys to the provided new channels (those with a zero
// LocalKey). The returned channels carry their assigned keys, in the same order as the
// input. Free channels draw their keys from a separate counter. A channel with an
// unspecified (zero) leaseholder defaults to the host node. Allocate returns a
// query.ErrNotFound error for a leaseholder that is neither the host node nor
// node.KeyFree. It creates no storage channels.
func (s *Service) Allocate(ctx context.Context, channels []Channel) ([]Channel, error) {
	out := make([]Channel, len(channels))
	copy(out, channels)
	host := s.cfg.HostKey
	for i := range out {
		if out[i].Leaseholder == 0 {
			out[i].Leaseholder = host
		}
	}
	indicesByTarget := lo.GroupBy(
		lo.Range(len(out)),
		func(i int) node.Key { return out[i].Lease() },
	)
	for target, indices := range indicesByTarget {
		chs := lo.Map(indices, func(i, _ int) Channel { return out[i] })
		var err error
		switch {
		case target == host:
			err = assignKeys(ctx, s.leasedCounter, chs)
		case target.Free():
			err = assignKeys(ctx, s.freeCounter, chs)
		default:
			err = errors.Wrapf(query.ErrNotFound, "node %d", target)
		}
		if err != nil {
			return nil, err
		}
		for j, i := range indices {
			out[i] = chs[j]
		}
	}
	return out, nil
}

// Create creates the storage channels for the provided channels, which carry the keys
// that Allocate assigned. Free channels have no storage registration and are skipped.
func (s *Service) Create(ctx context.Context, channels []Channel) error {
	stored := lo.Reject(channels, func(c Channel, _ int) bool { return c.Key().Free() })
	return s.cfg.TS.CreateChannel(
		ctx,
		lo.Map(stored, func(c Channel, _ int) ts.Channel { return c.Storage() })...,
	)
}

// assignKeys draws len(channels with a zero LocalKey) values from the provided counter
// and assigns them sequentially. Index channels have their LocalIndex set to their own
// LocalKey, whether that key was just assigned or already present.
func assignKeys(ctx context.Context, c *counter, channels []Channel) error {
	var toAssign LocalKey
	for _, ch := range channels {
		if ch.LocalKey == 0 {
			toAssign++
		}
	}
	next, err := c.add(ctx, toAssign)
	if err != nil {
		return err
	}
	next -= toAssign
	for i := range channels {
		if channels[i].LocalKey == 0 {
			next++
			channels[i].LocalKey = next
		}
		if channels[i].IsIndex {
			channels[i].LocalIndex = channels[i].LocalKey
		}
	}
	return nil
}
