// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("UnitsAssignable", func() {
	ns := &types.Unit{Dimensions: types.DimTime, Scale: 1, Name: "ns"}
	psi := &types.Unit{Name: "psi", Scale: 1}
	bar := &types.Unit{Name: "bar", Scale: 1}

	DescribeTable("unit compatibility",
		func(a, b *types.Unit, want bool) {
			Expect(types.UnitsAssignable(a, b)).To(Equal(want))
		},
		Entry("nil + nil", (*types.Unit)(nil), (*types.Unit)(nil), true),
		Entry("nil + ns (wildcard)", (*types.Unit)(nil), ns, true),
		Entry("ns + nil (wildcard)", ns, (*types.Unit)(nil), true),
		Entry("ns + ns (match)", ns, ns, true),
		Entry("psi + bar (mismatch)", psi, bar, false),
	)
})
