// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("DecodeMsgpack", func() {
	Describe("Output", func() {
		It("Should decode new lowercase msgpack fields", func() {
			original := compiler.Output{
				WASM:              []byte{0x00, 0x61, 0x73, 0x6d},
				OutputMemoryBases: map[string]uint32{"fn1": 16},
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded compiler.Output
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.WASM).To(Equal(original.WASM))
			Expect(decoded.OutputMemoryBases).To(Equal(original.OutputMemoryBases))
		})
		It("Should decode legacy uppercase Go field names", func() {
			legacy := struct {
				WASM              []byte
				OutputMemoryBases map[string]uint32
			}{
				WASM:              []byte{0x00, 0x61, 0x73, 0x6d},
				OutputMemoryBases: map[string]uint32{"fn1": 32},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var decoded compiler.Output
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.WASM).To(Equal([]byte{0x00, 0x61, 0x73, 0x6d}))
			Expect(decoded.OutputMemoryBases).To(Equal(map[string]uint32{"fn1": 32}))
		})
	})
})
