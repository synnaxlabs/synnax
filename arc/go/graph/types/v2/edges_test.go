// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/arc/graph/types/v2"
	"github.com/synnaxlabs/arc/ir"
)

var _ = Describe("Edges", func() {
	Describe("IR", func() {
		It("Should project graph edges into keyless ir edges", func() {
			edges := v2.Edges{
				{
					Key: "e1",
					Edge: ir.Edge{
						Source: ir.Handle{Node: "a", Param: "out"},
						Target: ir.Handle{Node: "b", Param: "in"},
						Kind:   ir.EdgeKindContinuous,
					},
				},
				{
					Key: "e2",
					Edge: ir.Edge{
						Source: ir.Handle{Node: "b", Param: "out"},
						Target: ir.Handle{Node: "c", Param: "in"},
						Kind:   ir.EdgeKindConditional,
					},
				},
			}
			Expect(edges.IR()).To(Equal(ir.Edges{
				{
					Source: ir.Handle{Node: "a", Param: "out"},
					Target: ir.Handle{Node: "b", Param: "in"},
					Kind:   ir.EdgeKindContinuous,
				},
				{
					Source: ir.Handle{Node: "b", Param: "out"},
					Target: ir.Handle{Node: "c", Param: "in"},
					Kind:   ir.EdgeKindConditional,
				},
			}))
		})

		It("Should return an empty collection for no edges", func() {
			Expect(v2.Edges{}.IR()).To(BeEmpty())
		})
	})
})
