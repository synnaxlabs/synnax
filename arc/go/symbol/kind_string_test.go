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

var _ = Describe("Kind.String", func() {
	DescribeTable("Should return the canonical name for each Kind",
		func(kind symbol.Kind, expected string) {
			Expect(kind.String()).To(Equal(expected))
		},
		Entry("KindVariable", symbol.KindVariable, "KindVariable"),
		Entry("KindStatefulVariable", symbol.KindStatefulVariable, "KindStatefulVariable"),
		Entry("KindChannel", symbol.KindChannel, "KindChannel"),
		Entry("KindFunction", symbol.KindFunction, "KindFunction"),
		Entry("KindBlock", symbol.KindBlock, "KindBlock"),
		Entry("KindConfig", symbol.KindConfig, "KindConfig"),
		Entry("KindInput", symbol.KindInput, "KindInput"),
		Entry("KindOutput", symbol.KindOutput, "KindOutput"),
		Entry("KindSequence", symbol.KindSequence, "KindSequence"),
		Entry("KindStage", symbol.KindStage, "KindStage"),
		Entry("KindConstant", symbol.KindConstant, "KindConstant"),
		Entry("KindLoop", symbol.KindLoop, "KindLoop"),
		Entry("KindLoopVariable", symbol.KindLoopVariable, "KindLoopVariable"),
		Entry("KindModule", symbol.KindModule, "KindModule"),
		Entry("KindModuleAlias", symbol.KindModuleAlias, "KindModuleAlias"),
		Entry("KindAmbient", symbol.KindAmbient, "KindAmbient"),
	)

	It("Should fall back to a numeric representation for unknown Kinds", func() {
		Expect(symbol.Kind(99).String()).To(Equal("Kind(99)"))
	})
})
