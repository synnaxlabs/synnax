// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"slices"
	"sync"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
)

// writerClocks returns the index channel of each group of the stored channels of keys,
// in key order, and the authority that the first write of each index that keys do not
// name raises it to: the authority of the first key on it. It leaves out an index that
// subject may not write.
func (s *Service) writerClocks(
	ctx context.Context,
	snap gorp.Snapshot,
	subject resource.ID,
	keys channel.Keys,
	authorities []control.Authority,
) (channel.Keys, map[channel.Key]control.Authority, error) {
	var channels []channel.Channel
	if err := s.channel.NewRetrieve().
		Entries(&channels).
		Where(channel.MatchKeys(keys...)).
		Exec(ctx, snap); err != nil {
		return nil, nil, err
	}
	allowed, err := s.access.NewEnforcer(snap).
		Allowed(ctx, subject, access.ActionCreate)
	if err != nil {
		return nil, nil, err
	}
	byKey := make(map[channel.Key]channel.Channel, len(channels))
	for _, ch := range channels {
		byKey[ch.Key()] = ch
	}
	var (
		clocks channel.Keys
		raises map[channel.Key]control.Authority
	)
	for i, key := range keys {
		ch, ok := byKey[key]
		index := ch.Index()
		if !ok || ch.Virtual || index == 0 || clocks.Contains(index) {
			continue
		}
		if !keys.Contains(index) {
			if !allowed(framer.ResourceIDs(channel.Keys{index})[0]) {
				continue
			}
			if raises == nil {
				raises = make(map[channel.Key]control.Authority)
			}
			raises[index] = authorityOf(authorities, i)
		}
		clocks = append(clocks, index)
	}
	return clocks, raises, nil
}

// authorityOf returns the authority that authorities give the key at position i.
func authorityOf(authorities []control.Authority, i int) control.Authority {
	switch len(authorities) {
	case 0:
		return control.AuthorityAbsolute
	case 1:
		return authorities[0]
	default:
		return authorities[i]
	}
}

// withRaises returns keys and authorities with each index of raises added at authority
// 0, in clock order, so the writer claims nothing on it against another writer.
func withRaises(
	keys channel.Keys,
	authorities []control.Authority,
	clocks channel.Keys,
	raises map[channel.Key]control.Authority,
) (channel.Keys, []control.Authority) {
	if len(raises) == 0 {
		return keys, authorities
	}
	all := make([]control.Authority, len(keys), len(keys)+len(raises))
	for i := range keys {
		all[i] = authorityOf(authorities, i)
	}
	keys = slices.Clone(keys)
	for _, clock := range clocks {
		if _, ok := raises[clock]; ok {
			keys = append(keys, clock)
			all = append(all, 0)
		}
	}
	return keys, all
}

// indexClaims raises each index that a writer open added, at the first write that holds
// it, to the authority of its group. The client never sees the response to a raise.
type indexClaims struct {
	confluence.AbstractLinear[framer.WriterRequest, framer.WriterRequest]
	// pending holds the authority of each added index that no write has held yet.
	pending map[channel.Key]control.Authority
	mu      struct {
		// raises holds, in request order, whether each SetAuthority request that has no
		// response yet is a raise.
		raises []bool
		sync.Mutex
	}
}

func newIndexClaims(raises map[channel.Key]control.Authority) *indexClaims {
	return &indexClaims{pending: raises}
}

// Flow implements confluence.Flow.
func (c *indexClaims) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(c.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-c.In.Outlet():
				if !ok {
					return nil
				}
				if err := c.forward(ctx, req); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

func (c *indexClaims) forward(ctx context.Context, req framer.WriterRequest) error {
	switch req.Command {
	case framer.WriterCommandSetAuthority:
		c.push(false)
	case framer.WriterCommandWrite:
		for key := range req.Frame.Keys() {
			authority, ok := c.pending[key]
			if !ok {
				continue
			}
			delete(c.pending, key)
			c.push(true)
			if err := signal.SendUnderContext(ctx, c.Out.Inlet(), framer.WriterRequest{
				Command: framer.WriterCommandSetAuthority,
				Config: framer.WriterConfig{
					Keys:        channel.Keys{key},
					Authorities: []control.Authority{authority},
				},
			}); err != nil {
				return err
			}
		}
	}
	return signal.SendUnderContext(ctx, c.Out.Inlet(), req)
}

func (c *indexClaims) push(raise bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mu.raises = append(c.mu.raises, raise)
}

// answers reports whether res answers a raise.
func (c *indexClaims) answers(res framer.WriterResponse) bool {
	if res.Command != framer.WriterCommandSetAuthority {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.mu.raises) == 0 {
		return false
	}
	raise := c.mu.raises[0]
	c.mu.raises = c.mu.raises[1:]
	return raise
}
