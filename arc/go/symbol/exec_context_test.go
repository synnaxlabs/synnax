// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
)

var _ = Describe("ExecContext", func() {
	Describe("Compatible", func() {
		DescribeTable("truth table",
			func(e, target symbol.ExecContext, expected bool) {
				Expect(e.Compatible(target)).To(Equal(expected))
			},
			Entry("untagged is never visible (WASM filter)", symbol.ExecContext(0), symbol.ExecWASM, false),
			Entry("untagged is never visible (Flow filter)", symbol.ExecContext(0), symbol.ExecFlow, false),
			Entry("untagged is never visible (no filter)", symbol.ExecContext(0), symbol.ExecContext(0), false),
			Entry("WASM visible when no filter", symbol.ExecWASM, symbol.ExecContext(0), true),
			Entry("Flow visible when no filter", symbol.ExecFlow, symbol.ExecContext(0), true),
			Entry("Both visible when no filter", symbol.ExecBoth, symbol.ExecContext(0), true),
			Entry("WASM compatible with WASM", symbol.ExecWASM, symbol.ExecWASM, true),
			Entry("Flow compatible with Flow", symbol.ExecFlow, symbol.ExecFlow, true),
			Entry("Both compatible with Both", symbol.ExecBoth, symbol.ExecBoth, true),
			Entry("WASM not compatible with Flow", symbol.ExecWASM, symbol.ExecFlow, false),
			Entry("Flow not compatible with WASM", symbol.ExecFlow, symbol.ExecWASM, false),
			Entry("Both compatible with WASM", symbol.ExecBoth, symbol.ExecWASM, true),
			Entry("Both compatible with Flow", symbol.ExecBoth, symbol.ExecFlow, true),
			Entry("WASM compatible with Both", symbol.ExecWASM, symbol.ExecBoth, true),
			Entry("Flow compatible with Both", symbol.ExecFlow, symbol.ExecBoth, true),
		)
	})
})
