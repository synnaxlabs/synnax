// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"github.com/synnaxlabs/arc/types"
	"github.com/vmihailenco/msgpack/v5"
)

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (h *Handle) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Handle
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(h)); err != nil {
		return err
	}
	if len(h.Node) == 0 {
		var legacy struct {
			Node  string
			Param string
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		h.Node = legacy.Node
		h.Param = legacy.Param
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (e *Edge) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Edge
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(e)); err != nil {
		return err
	}
	if len(e.Source.Node) == 0 {
		var legacy struct {
			Source Handle
			Target Handle
			Kind   EdgeKind
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		e.Source = legacy.Source
		e.Target = legacy.Target
		e.Kind = legacy.Kind
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (s *Stage) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Stage
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(s)); err != nil {
		return err
	}
	if len(s.Key) == 0 {
		var legacy struct {
			Key    string
			Nodes  []string
			Strata Strata
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		s.Key = legacy.Key
		s.Nodes = legacy.Nodes
		s.Strata = legacy.Strata
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (s *Sequence) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Sequence
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(s)); err != nil {
		return err
	}
	if len(s.Key) == 0 {
		var legacy struct {
			Key    string
			Stages []Stage
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		s.Key = legacy.Key
		s.Stages = legacy.Stages
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (f *Function) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Function
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(f)); err != nil {
		return err
	}
	if len(f.Key) == 0 {
		var legacy struct {
			Key      string
			Body     Body
			Config   types.Params
			Inputs   types.Params
			Outputs  types.Params
			Channels types.Channels
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		f.Key = legacy.Key
		f.Body = legacy.Body
		f.Config = legacy.Config
		f.Inputs = legacy.Inputs
		f.Outputs = legacy.Outputs
		f.Channels = legacy.Channels
	}
	return nil
}

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
			Config   types.Params
			Inputs   types.Params
			Outputs  types.Params
			Channels types.Channels
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		n.Key = legacy.Key
		n.Type = legacy.Type
		n.Config = legacy.Config
		n.Inputs = legacy.Inputs
		n.Outputs = legacy.Outputs
		n.Channels = legacy.Channels
	}
	return nil
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (ir *IR) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias IR
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(ir)); err != nil {
		return err
	}
	if ir.Functions == nil && ir.Nodes == nil {
		var legacy struct {
			Functions Functions
			Nodes     Nodes
			Edges     Edges
			Strata    Strata
			Sequences Sequences
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		ir.Functions = legacy.Functions
		ir.Nodes = legacy.Nodes
		ir.Edges = legacy.Edges
		ir.Strata = legacy.Strata
		ir.Sequences = legacy.Sequences
	}
	return nil
}
