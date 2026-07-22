// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"github.com/synnaxlabs/arc/types"
	"github.com/vmihailenco/msgpack/v5"
)

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
			Inputs   types.Params
			Outputs  types.Params
			Channels types.Channels
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		f.Key = legacy.Key
		f.Body = legacy.Body
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
			Inputs   types.Params
			Outputs  types.Params
			Channels types.Channels
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		n.Key = legacy.Key
		n.Type = legacy.Type
		n.Inputs = legacy.Inputs
		n.Outputs = legacy.Outputs
		n.Channels = legacy.Channels
	}
	return nil
}
