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
	"github.com/synnaxlabs/oracle/plugin/cpp/types"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Indirect", func() {
	DescribeTable("Should report whether a field holds its parent through an indirect",
		func(ctx SpecContext, field string, expected bool) {
			req := MustGenerateRequest(ctx, `
				@cpp output "out"

				Leaf struct {
					size int32
				}
				Node struct {
					parent   Node?
					children Node[]?
					leaf     Leaf?
					name     string
				}
			`, "test", NewMockFileLoader())
			node := req.Resolutions.MustGet("test.Node")
			f := MustBeOk(node.Form.(resolution.StructForm).Field(field))
			Expect(types.Indirect(f, node, req.Resolutions)).To(Equal(expected))
		},
		Entry("an optional field that holds the parent", "parent", true),
		Entry("a list of the parent", "children", false),
		Entry("an optional field of another type", "leaf", false),
		Entry("a required field", "name", false),
	)
})
