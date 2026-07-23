// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/vmihailenco/msgpack/v5"

// String returns the string representation of the handle as "node.param".
func (h Handle) String() string { return h.Node + "." + h.Param }

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase Go
// field names and new lowercase msgpack tag names for backward compatibility.
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
