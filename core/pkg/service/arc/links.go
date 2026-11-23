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

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/zyn"
)

// prepareExport replaces each channel input of a graph Arc's nodes with its channel
// key, so that the channel link options of an export write it as a path.
func (s *Service) prepareExport(_ context.Context, _ gorp.Snapshot, a *Arc) error {
	return s.mapChannelInputs(a, func(v any) (any, error) {
		var k uint32
		if err := zyn.Uint32().Coerce().Parse(v, &k); err != nil {
			return nil, err
		}
		return channel.Key(k), nil
	})
}

// readChannelInputs decodes each channel input of a graph Arc's nodes with opts, so
// that the channel link options of an import read a path or a raw key.
func (s *Service) readChannelInputs(a *Arc, opts []json.Options) error {
	return s.mapChannelInputs(a, func(v any) (any, error) {
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
// value. A node's function is a function of the graph or a standard flow function, as
// the channel references read it, so a channel name never shadows a function.
func (s *Service) mapChannelInputs(a *Arc, f func(any) (any, error)) error {
	b := a.Base()
	src, ok := b.Source.Variant.(GraphSource)
	if !ok {
		return nil
	}
	var (
		inputs = src.Graph.Inputs
		cloned = make(map[string]bool)
		err    error
	)
	s.params().eachNodeParam(
		src.Graph,
		types.ChanDirectionReadWrite,
		func(node string, values msgpack.EncodedJSON, param string) {
			if err != nil {
				return
			}
			if !cloned[node] {
				if len(cloned) == 0 {
					inputs = maps.Clone(inputs)
				}
				values = maps.Clone(values)
				inputs[node] = values
				cloned[node] = true
			} else {
				values = inputs[node]
			}
			if values[param], err = f(values[param]); err != nil {
				err = errors.Wrapf(err, "input %s of node %s", param, node)
			}
		},
	)
	if err != nil || len(cloned) == 0 {
		return err
	}
	src.Graph.Inputs = inputs
	b.Source.Variant = src
	a.SetBase(b)
	return nil
}
