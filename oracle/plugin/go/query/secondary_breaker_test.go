// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package query_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/query"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Search secondary breaker", func() {
	var (
		loader *MockFileLoader
		p      *query.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = query.New(query.DefaultOptions())
	})

	DescribeTable("Should refuse a @search expression it does not know",
		func(expr string) {
			source := `
				@go output "core/pkg/service/foo"

				Foo struct {
					key uuid @key
					name string @search
					description string {
						@search ` + expr + `
					}
					@resource type "foo"
					@retrieve
				}
			`
			req := MustGenerateRequest(source, "foo", loader)
			Expect(p.Generate(req)).Error().To(HaveOccurred())
		},
		Entry("a misspelled secondary", "secondry"),
		Entry("secondary with a value", "secondary false"),
	)

	DescribeTable("Should refuse @search secondary on a field that is not a string",
		func(field string) {
			source := `
				@go output "core/pkg/service/foo"

				Foo struct {
					key uuid @key
					name string @search
					` + field + ` {
						@search secondary
					}
					@retrieve
				}
			`
			req := MustGenerateRequest(source, "foo", loader)
			Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
				"foo.Foo: @search field text must be a string that is not optional",
			)))
		},
		Entry("optional", "text string?"),
		Entry("number", "text int32"),
		Entry("array", "text string[]"),
	)

	It("Should weigh an inherited primary field above a secondary field", func() {
		source := `
			@go output "core/pkg/service/foo"

			Base struct {
				key uuid @key
				name string @search
			}

			Foo struct extends Base {
				note string {
					@search secondary
				}
				@resource type "foo"
				@retrieve
			}
		`
		resp := MustGenerate(source, "foo", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToContain("return []string{e.Name}").
			ToContain(`if e.Note == "" {`).
			ToContain("return []string{e.Note}")
	})

	It("Should read a shared secondary field through the base method", func() {
		source := `
			@go output "core/pkg/service/view"

			Base struct {
				key uuid @key
				name string @search
				note string {
					@search secondary
				}
			}

			View union on type extends Base {
				first {
					size int32
				}
				@retrieve
			}
		`
		resp := MustGenerate(source, "view", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToContain("return []string{e.Base().Name}").
			ToContain(`if e.Base().Note == "" {`).
			ToContain("return []string{e.Base().Note}")
	})
})
