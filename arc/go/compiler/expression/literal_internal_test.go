// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("writeNumericConst", func() {
	DescribeTable("returns an error when the boxed value type does not match the kind",
		func(parsed literal.ParsedValue) {
			Expect(writeNumericConst(wasm.NewWriter(), parsed)).
				Error().To(MatchError(ContainSubstring("unexpected value type")))
		},
		Entry("i64 kind, string box", literal.ParsedValue{Type: types.I64(), Value: "x"}),
		Entry("i64 kind, float64 box", literal.ParsedValue{Type: types.I64(), Value: float64(1)}),
		Entry("i32 kind, string box", literal.ParsedValue{Type: types.I32(), Value: "x"}),
		Entry("u8 kind, int64 box", literal.ParsedValue{Type: types.U8(), Value: int64(1)}),
	)

	It("returns an error for a non-numeric kind", func() {
		Expect(writeNumericConst(wasm.NewWriter(), literal.ParsedValue{Type: types.String(), Value: "x"})).
			Error().To(MatchError(ContainSubstring("unsupported numeric type")))
	})
})
