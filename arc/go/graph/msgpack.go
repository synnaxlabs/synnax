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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/binary"
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
			Type     string
			Config   binary.MsgpackEncodedJSON
			Position spatial.XY
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		n.Key = legacy.Key
		n.Type = legacy.Type
		n.Config = legacy.Config
		n.Position = legacy.Position
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (v *Viewport) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Viewport
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(v)); err != nil {
		return err
	}
	// Always try legacy since Zoom=0 could indicate failed decode or valid value,
	// but in practice zoom is never 0 for new data (defaults to 1.0).
	var legacy struct {
		Position spatial.XY
		Zoom     float64
	}
	if err = msgpack.Unmarshal(raw, &legacy); err != nil {
		return err
	}
	if legacy.Zoom != 0 && v.Zoom == 0 {
		v.Position = legacy.Position
		v.Zoom = legacy.Zoom
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
			Viewport  Viewport
			Functions ir.Functions
			Edges     ir.Edges
			Nodes     Nodes
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		g.Viewport = legacy.Viewport
		g.Functions = legacy.Functions
		g.Edges = legacy.Edges
		g.Nodes = legacy.Nodes
	}
	return nil
}
