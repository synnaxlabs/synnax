// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import (
	"github.com/vmihailenco/msgpack/v5"
)

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (o *Output) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Output
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(o)); err != nil {
		return err
	}
	if o.WASM == nil {
		var legacy struct {
			WASM              []byte
			OutputMemoryBases map[string]uint32
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		o.WASM = legacy.WASM
		o.OutputMemoryBases = legacy.OutputMemoryBases
	}
	return nil
}
