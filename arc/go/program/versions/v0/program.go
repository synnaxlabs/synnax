// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/x/encoding/orc"

// EncodeOrc writes the value to w in the Orc binary format. The layout covers only the
// compiled output (WASM and memory bases), never the IR, and is frozen: stored arc v0
// records embed it.
func (p Program) EncodeOrc(w *orc.Writer) error {
	w.Bool(p.WASM != nil)
	if p.WASM != nil {
		w.WriteWithLen(p.WASM)
	}
	w.Bool(p.OutputMemoryBases != nil)
	if p.OutputMemoryBases != nil {
		w.Uint32(uint32(len(p.OutputMemoryBases)))
		for key, val := range p.OutputMemoryBases {
			w.String(key)
			w.Uint32(val)
		}
	}
	return nil
}

// DecodeOrc reads the value from r in the Orc binary format.
func (p *Program) DecodeOrc(r *orc.Reader) error {
	{
		present, err := r.Bool()
		if err != nil {
			return err
		}
		if present {
			p.WASM, err = r.ReadWithLen()
			if err != nil {
				return err
			}
		}
	}
	{
		present, err := r.Bool()
		if err != nil {
			return err
		}
		if present {
			n, err := r.CollectionLen()
			if err != nil {
				return err
			}
			p.OutputMemoryBases = make(map[string]uint32, n)
			for range n {
				var key string
				var val uint32
				if key, err = r.String(); err != nil {
					return err
				}
				if val, err = r.Uint32(); err != nil {
					return err
				}
				p.OutputMemoryBases[key] = val
			}
		}
	}
	return nil
}
