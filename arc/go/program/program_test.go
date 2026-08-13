// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package program_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	program "github.com/synnaxlabs/arc/program"
	v0 "github.com/synnaxlabs/arc/program/versions/v0"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/orc"
)

var _ = Describe("Program", func() {
	Describe("IsZero", func() {
		It("Should return true for an empty program", func() {
			m := program.Program{}
			Expect(m.IsZero()).To(BeTrue())
		})

		It("Should return false for a program with WASM", func() {
			m := program.Program{
				Output: compiler.Output{
					WASM: []byte{0x00, 0x61, 0x73, 0x6d},
				},
			}
			Expect(m.IsZero()).To(BeFalse())
		})
	})

	Describe("String", func() {
		It("Should include WASM summary with no content", func() {
			m := program.Program{}
			s := m.String()
			Expect(s).To(ContainSubstring("Arc Program"))
			Expect(s).To(ContainSubstring("WASM: (none)"))
		})

		It("Should include WASM size and hash when bytecode is present", func() {
			m := program.Program{
				Output: compiler.Output{
					WASM: []byte{0x00, 0x61, 0x73, 0x6d},
				},
			}
			s := m.String()
			Expect(s).To(ContainSubstring("WASM: 4 bytes (sha256:"))
		})

		It("Should include IR content when nodes are present", func() {
			m := program.Program{
				IR: ir.IR{
					Nodes: ir.Nodes{{
						Key:  "node1",
						Type: "add",
						Inputs: types.Params{
							{Name: "a", Type: types.I64()},
						},
					}},
				},
			}
			s := m.String()
			Expect(s).To(ContainSubstring("Arc Program"))
			Expect(s).To(ContainSubstring("node1"))
		})
	})

	Describe("Orc", func() {
		It("Should round-trip the compiled output", func() {
			original := v0.Program{
				Output: compiler.Output{
					WASM:              []byte{0xDE, 0xAD, 0xBE, 0xEF},
					OutputMemoryBases: map[string]uint32{"main": 7},
				},
			}
			w := orc.NewWriter(0)
			Expect(original.EncodeOrc(w)).To(Succeed())
			var decoded v0.Program
			r := orc.NewReader(nil)
			r.ResetBytes(w.Bytes())
			Expect(decoded.DecodeOrc(r)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})

		// Stored arc v0 records embed this layout; a change here corrupts released
		// rows. The fixture uses a single memory base because map iteration makes
		// multi-entry encodings order-dependent.
		It("Should keep the frozen byte layout stored arc v0 records embed", func() {
			p := v0.Program{Output: compiler.Output{
				WASM:              []byte{0xDE, 0xAD, 0xBE, 0xEF},
				OutputMemoryBases: map[string]uint32{"main": 7},
			}}
			w := orc.NewWriter(0)
			Expect(p.EncodeOrc(w)).To(Succeed())
			Expect(w.Bytes()).To(Equal([]byte{
				0x01, 0x00, 0x00, 0x00, 0x04, 0xDE, 0xAD, 0xBE, 0xEF,
				0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x04,
				0x6D, 0x61, 0x69, 0x6E, 0x00, 0x00, 0x00, 0x07,
			}))
		})
	})
})
