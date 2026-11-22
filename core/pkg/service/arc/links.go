// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"encoding/json/v2"
	"maps"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/resolver"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/zyn"
)

// prepareExport replaces each channel input of a graph Arc's nodes with its channel
// key, so that the channel link options of an export write it as a path.
func (s *Service) prepareExport(ctx context.Context, snap gorp.Snapshot, a *Arc) error {
	return s.mapChannelInputs(ctx, snap, a, func(v any) (any, error) {
		var k uint32
		if err := zyn.Uint32().Coerce().Parse(v, &k); err != nil {
			return nil, err
		}
		return channel.Key(k), nil
	})
}

// readChannelInputs decodes each channel input of a graph Arc's nodes with opts, so
// that the channel link options of an import read a path or a raw key.
func (s *Service) readChannelInputs(
	ctx context.Context,
	a *Arc,
	opts []json.Options,
) error {
	return s.mapChannelInputs(ctx, s.cfg.DB, a, func(v any) (any, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var k channel.Key
		if err = json.Unmarshal(b, &k, opts...); err != nil {
			return nil, err
		}
		return k, nil
	})
}

// mapChannelInputs replaces each channel input of a graph Arc's nodes with f of its
// value, resolving the node functions as snap sees them.
func (s *Service) mapChannelInputs(
	ctx context.Context,
	snap gorp.Snapshot,
	a *Arc,
	f func(any) (any, error),
) error {
	b := a.Base()
	src, ok := b.Source.Variant.(GraphSource)
	if !ok {
		return nil
	}
	channels, err := graph.ChannelInputs(
		ctx,
		src.Graph,
		s.NewRoot(snap, &resolver.Lookups{}),
	)
	if err != nil || len(channels) == 0 {
		return err
	}
	inputs := maps.Clone(src.Graph.Inputs)
	for node, names := range channels {
		values := maps.Clone(inputs[node])
		for _, name := range names {
			v, ok := values[name]
			if !ok {
				continue
			}
			if values[name], err = f(v); err != nil {
				return errors.Wrapf(err, "input %s of node %s", name, node)
			}
		}
		inputs[node] = values
	}
	src.Graph.Inputs = inputs
	b.Source.Variant = src
	a.SetBase(b)
	return nil
}
