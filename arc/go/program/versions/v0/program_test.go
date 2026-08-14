// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/arc/program/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

// Released rows nest the compiled output under the embedded compiler.Output field
// name "Output", because its custom decoder blocks msgpack inlining. The mirrors
// below reproduce those bytes with explicit named fields.
var _ = Describe("Legacy msgpack decode", func() {
	It("Should decode a main-era row nesting tagged output", func() {
		type output struct {
			WASM              []byte            `msgpack:"wasm"`
			OutputMemoryBases map[string]uint32 `msgpack:"output_memory_bases"`
		}
		type program struct {
			Output output `msgpack:"Output"`
		}
		data := MustSucceed(msgpack.Marshal(program{output{
			WASM:              []byte{0xDE, 0xAD},
			OutputMemoryBases: map[string]uint32{"main": 7},
		}}))
		var p v0.Program
		Expect(msgpack.Unmarshal(data, &p)).To(Succeed())
		Expect(p.WASM).To(Equal([]byte{0xDE, 0xAD}))
		Expect(p.OutputMemoryBases).To(Equal(map[string]uint32{"main": 7}))
	})

	It("Should decode a pre-tag row keying output by Go field name", func() {
		type output struct {
			WASM              []byte
			OutputMemoryBases map[string]uint32
		}
		type program struct {
			Output output `msgpack:"Output"`
		}
		data := MustSucceed(msgpack.Marshal(program{output{
			WASM:              []byte{0xBE, 0xEF},
			OutputMemoryBases: map[string]uint32{"main": 3},
		}}))
		var p v0.Program
		Expect(msgpack.Unmarshal(data, &p)).To(Succeed())
		Expect(p.WASM).To(Equal([]byte{0xBE, 0xEF}))
		Expect(p.OutputMemoryBases).To(Equal(map[string]uint32{"main": 3}))
	})

	It("Should decode the flat tagged shape", func() {
		data := MustSucceed(msgpack.Marshal(map[string]any{
			"wasm":                []byte{0x01},
			"output_memory_bases": map[string]uint32{"main": 1},
		}))
		var p v0.Program
		Expect(msgpack.Unmarshal(data, &p)).To(Succeed())
		Expect(p.WASM).To(Equal([]byte{0x01}))
		Expect(p.OutputMemoryBases).To(Equal(map[string]uint32{"main": 1}))
	})
})
