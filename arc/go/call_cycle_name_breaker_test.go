// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
)

var _ = Describe("Call cycle breaker", func() {
	It(
		"Should not report a cycle for a func that calls a module member of its name",
		func(
			ctx SpecContext,
		) {
			src := "import math\n\nfunc is_nan(x f64) bool {\n    y := math.is_nan(x)\n" +
				"    return y\n}\n"
			_, err := arc.CompileText(
				ctx,
				arc.Text{Raw: src},
				symbol.NewRoot(nil, stl.NewSymbols()),
			)
			Expect(err).ToNot(HaveOccurred())
		},
	)
})
