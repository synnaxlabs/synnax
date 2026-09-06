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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("CanEmbed", func() {
	// childForm resolves source and returns the form of its Child struct.
	childForm := func(
		ctx context.Context, source string,
	) (resolution.StructForm, *resolution.Table) {
		GinkgoHelper()
		table := MustGenerateRequest(ctx, source, "test", NewMockFileLoader()).
			Resolutions
		form, ok := MustBeOk(table.Get("test.Child")).Form.(resolution.StructForm)
		Expect(ok).To(BeTrue(), "Child is not a struct")
		return form, table
	}

	DescribeTable("Should report whether the parents can be embedded",
		func(ctx SpecContext, source string, expected bool) {
			form, table := childForm(ctx, source)
			Expect(types.CanEmbed(form, table)).To(Equal(expected))
		},
		Entry("no parents", `
			@go output "core/test"

			Child struct {
				c string
			}
		`, false),
		Entry("disjoint parents", `
			@go output "core/test"

			A struct {
				a string
			}

			B struct {
				b string
			}

			Child struct extends A, B {
				c string
			}
		`, true),
		Entry("an omitted field", `
			@go output "core/test"

			A struct {
				a string
				drop string
			}

			Child struct extends A {
				-drop
			}
		`, false),
		Entry("a field that drops an inherited domain", `
			@go output "core/test"

			A struct {
				a string @validate required
			}

			Child struct extends A {
				a -@validate
			}
		`, false),
		Entry("a field that restates an inherited type", `
			@go output "core/test"

			A struct {
				a string
			}

			Child struct extends A {
				a string?
			}
		`, false),
		Entry("a parent whose embed name a sibling parent's field shadows", `
			@go output "core/test"

			Base struct {
				x string
			}

			A struct extends Base {
				aa string
			}

			B struct {
				base string
			}

			Child struct extends A, B {
				c string
			}
		`, false),
		Entry("a flattened parent promoting a name another parent embeds", `
			@go output "core/test"

			Base struct {
				middle string
				drop   string
			}

			A struct extends Base {
				-drop
			}

			Middle struct {
				m string
			}

			B struct extends Middle {
				b string
			}

			Child struct extends A, B {
				c string
			}
		`, false),
	)
})
