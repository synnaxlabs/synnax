// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	ir "github.com/synnaxlabs/arc/ir/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	vmsgpack "github.com/vmihailenco/msgpack/v5"
)

// legacyXY is a point as Cores before v0.54 stored it, under its Go field names.
type legacyXY struct{ X, Y float64 }

func (p legacyXY) xy() spatial.XY { return spatial.XY{X: p.X, Y: p.Y} }

// DecodeMsgpack implements msgpack.CustomDecoder. It also reads graphs that Cores
// before v0.54 stored under their uppercase Go field names.
func (g *Graph) DecodeMsgpack(dec *vmsgpack.Decoder) error {
	type alias Graph
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = vmsgpack.Unmarshal(raw, (*alias)(g)); err != nil {
		return err
	}
	if g.Nodes != nil {
		return nil
	}
	var legacy struct {
		Viewport  Viewport
		Functions ir.Functions
		Edges     ir.Edges
		Nodes     Nodes
	}
	if err = vmsgpack.Unmarshal(raw, &legacy); err != nil {
		return err
	}
	g.Viewport = legacy.Viewport
	g.Functions = legacy.Functions
	g.Edges = legacy.Edges
	g.Nodes = legacy.Nodes
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder. It also reads nodes that Cores
// before v0.54 stored under their uppercase Go field names.
func (n *Node) DecodeMsgpack(dec *vmsgpack.Decoder) error {
	type alias Node
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = vmsgpack.Unmarshal(raw, (*alias)(n)); err != nil {
		return err
	}
	if n.Key != "" {
		return nil
	}
	var legacy struct {
		Key      string
		Type     string
		Config   msgpack.EncodedJSON
		Position legacyXY
	}
	if err = vmsgpack.Unmarshal(raw, &legacy); err != nil {
		return err
	}
	n.Key = legacy.Key
	n.Type = legacy.Type
	n.Config = legacy.Config
	n.Position = legacy.Position.xy()
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder. It also reads viewports that Cores
// before v0.54 stored under their uppercase Go field names.
func (v *Viewport) DecodeMsgpack(dec *vmsgpack.Decoder) error {
	type alias Viewport
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = vmsgpack.Unmarshal(raw, (*alias)(v)); err != nil {
		return err
	}
	if *v != (Viewport{}) {
		return nil
	}
	var legacy struct {
		Position legacyXY
		Zoom     float64
	}
	if err = vmsgpack.Unmarshal(raw, &legacy); err != nil {
		return err
	}
	v.Position = legacy.Position.xy()
	v.Zoom = legacy.Zoom
	return nil
}
