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
	"github.com/synnaxlabs/arc/program"
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
})
