// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

import (
	"maps"

	"github.com/google/uuid"
	"github.com/synnaxlabs/arc/ir"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	"github.com/vmihailenco/msgpack/v5"
)

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (n *Node) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Node
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(n)); err != nil {
		return err
	}
	if len(n.Key) == 0 {
		var legacy struct {
			Key      string
			Position spatial.XY
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		n.Key = legacy.Key
		n.Position = legacy.Position
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (g *Graph) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Graph
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(g)); err != nil {
		return err
	}
	if g.Nodes == nil {
		var legacy struct {
			Functions ir.Functions
			Edges     ir.Edges
			Nodes     Nodes
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		g.Functions = legacy.Functions
		g.Edges = make(Edges, len(legacy.Edges))
		for i, e := range legacy.Edges {
			g.Edges[i] = Edge{Edge: e, Key: uuid.NewString()}
		}
		g.Nodes = legacy.Nodes
	}
	// Legacy graphs stored the function type and config inline on each node with
	// no inputs map; lift them into Inputs keyed by node key.
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
				cfg := xmsgpack.EncodedJSON{}
				maps.Copy(cfg, ln.Config)
				cfg["type"] = ln.Type
				g.Inputs[ln.Key] = cfg
			}
		}
	}
	return nil
}
