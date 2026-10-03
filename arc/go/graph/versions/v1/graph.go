// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"maps"

	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/vmihailenco/msgpack/v5"
)

// DecodeMsgpack implements msgpack.CustomDecoder. It also reads graphs that Consoles
// before v0.57 sent with each node's type and config inline and no inputs map.
func (g *Graph) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Graph
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(g)); err != nil {
		return err
	}
	if g.Inputs == nil {
		var legacy struct {
			Nodes []struct {
				Key    string               `msgpack:"key"`
				Type   string               `msgpack:"type"`
				Config xmsgpack.EncodedJSON `msgpack:"config"`
			} `msgpack:"nodes"`
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		if len(legacy.Nodes) > 0 {
			g.Inputs = make(map[string]xmsgpack.EncodedJSON, len(legacy.Nodes))
			for _, ln := range legacy.Nodes {
				inputs := xmsgpack.EncodedJSON{}
				maps.Copy(inputs, ln.Config)
				inputs["type"] = ln.Type
				g.Inputs[ln.Key] = inputs
			}
		}
	}
	return nil
}
