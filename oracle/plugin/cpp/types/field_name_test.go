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
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin/cpp/types"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("FieldName", func() {
	DescribeTable("Should return the C++ name of a field",
		func(ctx SpecContext, field, expected string) {
			table, diag := Analyze(ctx, `
				@cpp output "client/cpp/thing"

				Thing struct {
					`+field+`
				}
			`, "thing", NewMockFileLoader())
			Expect(diag.Ok()).To(BeTrue())
			thing := lo.Must(table.Get("thing.Thing"))
			form := thing.Form.(resolution.StructForm)
			Expect(types.FieldName(form.Fields[0])).To(Equal(expected))
		},
		Entry("a snake case name", "data_type uint8", "data_type"),
		Entry("a camel case name", "dataType uint8", "data_type"),
		Entry("an explicit name", "value int32 { @cpp name raw }", "raw"),
		Entry("a keyword", "virtual bool", "virtual_"),
		Entry("an explicit keyword", "value int32 { @cpp name class }", "class_"),
	)
})
