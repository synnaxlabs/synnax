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
	"encoding/json/v2"
	"reflect"

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/zyn"
)

// prepareExport replaces each channel input of a graph Arc's nodes that holds a number
// with its channel key, so that the channel link options of an export write it as a
// path. An input that holds no number, such as text or null, exports as stored.
func (s *Service) prepareExport(a *Arc) error {
	return s.mapChannelInputs(a, func(v any) (any, error) {
		if !isNumber(v) {
			return v, nil
		}
		var k uint32
		if err := zyn.Uint32().Coerce().Parse(v, &k); err != nil {
			return nil, err
		}
		return channel.Key(k), nil
	})
}

// isNumber reports whether v is a Go number.
func isNumber(v any) bool {
	switch reflect.ValueOf(v).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
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
// value, in place, so a must own its node inputs. A node's function is a function of
// the graph or a standard flow function, as the channel references read it, so a
// channel name never shadows a function.
func (s *Service) mapChannelInputs(a *Arc, f func(any) (any, error)) error {
	src, ok := a.Base().Source.Variant.(GraphSource)
	if !ok {
		return nil
	}
	var err error
	s.params().eachNodeParam(
		src.Graph,
		types.ChanDirectionReadWrite,
		func(node string, inputs msgpack.EncodedJSON, param string) {
			if err != nil {
				return
			}
			if inputs[param], err = f(inputs[param]); err != nil {
				err = errors.Wrapf(err, "input %s of node %s", param, node)
			}
		},
	)
	return err
}
