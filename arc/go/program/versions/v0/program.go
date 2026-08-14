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

// DecodeMsgpack implements msgpack.CustomDecoder for rows stored before Orc
// encoding. Released rows nest the compiled output under the embedded
// compiler.Output field name "Output"; the oldest rows key its fields by Go
// field name instead of msgpack tag.
func (p *Program) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Program
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(p)); err != nil {
		return err
	}
	if p.WASM != nil {
		return nil
	}
	var nested struct {
		Output struct {
			WASM              []byte            `msgpack:"wasm"`
			OutputMemoryBases map[string]uint32 `msgpack:"output_memory_bases"`
		} `msgpack:"Output"`
	}
	if err = msgpack.Unmarshal(raw, &nested); err != nil {
		return err
	}
	if nested.Output.WASM == nil {
		var legacy struct {
			Output struct {
				WASM              []byte
				OutputMemoryBases map[string]uint32
			} `msgpack:"Output"`
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		nested.Output.WASM = legacy.Output.WASM
		nested.Output.OutputMemoryBases = legacy.Output.OutputMemoryBases
	}
	p.WASM = nested.Output.WASM
	if p.OutputMemoryBases == nil {
		p.OutputMemoryBases = nested.Output.OutputMemoryBases
	}
	return nil
}
