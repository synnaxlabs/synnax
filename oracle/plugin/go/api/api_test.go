// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/api"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Go API Filter Plugin", func() {
	var (
		loader *MockFileLoader
		p      *api.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = api.New(api.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(p.Name()).To(Equal("go/api"))
		})

		It("Should filter on go domain", func() {
			Expect(p.Domains()).To(Equal([]string{"go"}))
		})

		It("Should require go/query", func() {
			Expect(p.Requires()).To(Equal([]string{"go/query"}))
		})
	})

	Describe("Generate", func() {
		Context("struct with string and bool filters", func() {
			It("Should generate a FilterNode with typed slots and toFilter interpreter", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/rack"
					@api output "core/pkg/api/rack"

					Rack struct {
						key uint32 {
							@key
						}
						name string {
							@filter
						}
						embedded bool {
							@filter
						}
						@ontology type "rack"
						@retrieve custom
					}
				`
				resp := MustGenerate(ctx, source, "rack", loader, p)
				Expect(resp.Files).To(HaveLen(1))

				ExpectContent(resp, "filter.gen.go").
					ToContain(
						"package rack",
						"type RackFilterNode struct",
						"Name *filter.StringFilter",
						"Embedded *filter.BoolFilter",
						"And []RackFilterNode",
						"Or  []RackFilterNode",
						"Not *RackFilterNode",
						"func (n RackFilterNode) toFilter()",
						"func (n *RackFilterNode) ToFilter()",
					)
			})
		})

		Context("struct with no @api output", func() {
			It("Should not generate any files", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/foo"

					Foo struct {
						key uuid {
							@key
						}
						name string {
							@filter
						}
						@ontology type "foo"
						@retrieve
					}
				`
				resp := MustGenerate(ctx, source, "foo", loader, p)
				Expect(resp.Files).To(HaveLen(0))
			})
		})

		Context("struct with no @filter fields", func() {
			It("Should not generate any files", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/foo"
					@api output "core/pkg/api/foo"

					Foo struct {
						key uuid {
							@key
						}
						name string
						@ontology type "foo"
						@retrieve
					}
				`
				resp := MustGenerate(ctx, source, "foo", loader, p)
				Expect(resp.Files).To(HaveLen(0))
			})
		})

		Context("bool filter routing", func() {
			It("Should route bool eq to MatchX(bool)", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/task"
					@api output "core/pkg/api/task"

					Task struct {
						key uuid {
							@key
						}
						internal bool {
							@filter
						}
						@ontology type "task"
						@retrieve custom
					}
				`
				resp := MustGenerate(ctx, source, "task", loader, p)

				ExpectContent(resp, "filter.gen.go").
					ToContain(
						"Internal *filter.BoolFilter",
						"MatchInternal(*n.Internal.Eq)",
					)
			})
		})

		Context("string filter with contains and regex", func() {
			It("Should generate inline Match closures for contains and regex", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/rack"
					@api output "core/pkg/api/rack"

					Rack struct {
						key uint32 {
							@key
						}
						name string {
							@filter
						}
						@ontology type "rack"
						@retrieve custom
					}
				`
				resp := MustGenerate(ctx, source, "rack", loader, p)

				ExpectContent(resp, "filter.gen.go").
					ToContain(
						"strings.Contains(",
						"regexp.Compile(",
						"rx.MatchString(",
					)
			})
		})

		Context("scalar filter", func() {
			It("Should generate MatchX (singular) for scalar filters", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/workspace"
					@api output "core/pkg/api/workspace"

					Workspace struct {
						key uuid {
							@key
						}
						author uuid {
							@filter scalar
						}
						@ontology type "workspace"
						@retrieve custom
					}
				`
				resp := MustGenerate(ctx, source, "workspace", loader, p)

				ExpectContent(resp, "filter.gen.go").
					ToContain(
						"MatchAuthor(",
					).
					ToNotContain(
						"MatchAuthors(",
					)
			})
		})

		Context("And/Or/Not composition", func() {
			It("Should generate recursive And/Or/Not handling", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/rack"
					@api output "core/pkg/api/rack"

					Rack struct {
						key uint32 {
							@key
						}
						name string {
							@filter
						}
						@ontology type "rack"
						@retrieve custom
					}
				`
				resp := MustGenerate(ctx, source, "rack", loader, p)

				ExpectContent(resp, "filter.gen.go").
					ToContain(
						"for _, child := range n.And",
						"child.toFilter()",
						"Or(orFilters...)",
						"Not(n.Not.toFilter())",
					)
			})
		})

		Context("sorted index generates OrderBy", func() {
			It("Should generate OrderBy struct and ToOrderBy method", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/event"
					@api output "core/pkg/api/event"

					Event struct {
						key uuid {
							@key
						}
						created_at int64 {
							@filter scalar
							@index sorted
						}
						@ontology type "event"
						@retrieve
					}
				`
				resp := MustGenerate(ctx, source, "event", loader, p)

				ExpectContent(resp, "filter.gen.go").
					ToContain(
						"type EventOrderBy struct",
						"CreatedAt *filter.NumericOrderBy[int64]",
						"func (o *EventOrderBy) ToOrderBy()",
						"filter.ParseDirection(",
						"OrderByCreatedAt(dir",
					)
			})
		})

		Context("cross-namespace filter type", func() {
			It("Should handle a filter field with a cross-namespace type", func(ctx SpecContext) {
				loader.Add("schemas/rack", `
					@go output "core/pkg/service/rack"

					Key = uint32
				`)

				source := `
					import "schemas/rack"

					@go output "core/pkg/service/device"
					@api output "core/pkg/api/device"

					Device struct {
						key string {
							@key
						}
						rack rack.Key {
							@filter
						}
						@ontology type "device"
						@retrieve custom
					}
				`
				resp := MustGenerate(ctx, source, "device", loader, p)

				ExpectContent(resp, "filter.gen.go").
					ToContain(
						"type DeviceFilterNode struct",
						"Rack *filter.NumericFilter[uint32]",
					)
			})
		})

		Context("entity with no sorted index", func() {
			It("Should not generate OrderBy types", func(ctx SpecContext) {
				source := `
					@go output "core/pkg/service/rack"
					@api output "core/pkg/api/rack"

					Rack struct {
						key uint32 {
							@key
						}
						name string {
							@filter
						}
						@ontology type "rack"
						@retrieve custom
					}
				`
				resp := MustGenerate(ctx, source, "rack", loader, p)

				ExpectContent(resp, "filter.gen.go").
					ToNotContain(
						"OrderBy",
						"ToOrderBy",
						"ParseDirection",
					)
			})
		})
	})
})
