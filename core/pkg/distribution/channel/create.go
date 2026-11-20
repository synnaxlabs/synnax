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

// Issued reports whether no later Allocate can return key k. A key leased to the host
// node or free counts once storage holds its counter at or past its local key. A key of
// another node counts, since Allocate issues only host and free keys. A key of a
// deleted channel counts.
func (s *Service) Issued(k Key) bool {
	switch {
	case k.Free():
		return s.freeCounter.issued(k.LocalKey())
	case k.Lease() == s.cfg.HostKey:
		return s.leasedCounter.issued(k.LocalKey())
	}
	return true
}

// IssuedAtOpen reports whether Issued(k) held when the service opened, before any
// Allocate of this run.
func (s *Service) IssuedAtOpen(k Key) bool {
	switch {
	case k.Free():
		return s.freeCounter.issuedAtOpen(k.LocalKey())
	case k.Lease() == s.cfg.HostKey:
		return s.leasedCounter.issuedAtOpen(k.LocalKey())
	}
	return true
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
