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

var _ = Describe("@ref", func() {
	var (
		loader *MockFileLoader
		p      *query.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = query.New(query.DefaultOptions())
		loader.Add("schemas/rack", `
			@go output "core/pkg/service/rack"

			Key = uint32

			Rack struct {
				key Key {
					@key
				}
				@resource type "rack"
			}
		`)
		loader.Add("schemas/label", `
			@go output "core/pkg/service/label"

			Key = uuid

			Label struct {
				key Key {
					@key
				}
				@resource type "label"
			}
		`)
		loader.Add("schemas/resource", `
			@go output "core/pkg/service/resource"

			Type = string

			ID struct {
				type Type
				key  string
			}
		`)
	})

	device := func(fields string) string {
		return `
			import "schemas/rack"
			import "schemas/label"
			import "schemas/resource"

			@go output "core/pkg/service/device"

			Key = string

			Device struct {
				key Key {
					@key
				}
				home  rack.Key
				tag   label.Key?
				owned resource.ID?
				` + fields + `
				@resource type "device"
				@retrieve
			}
		`
	}

	It(
		"Should register a reference to the key of another resource",
		func(ctx SpecContext) {
			resp := MustGenerate(ctx, device(`
			rack rack.Key @ref on_delete restrict
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					"rack *gorp.Reference[Key, Device]",
					"rack: gorp.NewReference(gorp.ReferenceConfig[Key, Device]{",
					`Name:     "rack",`,
					`Tables:   []string{"rack"},`,
					"OnDelete: gorp.Restrict,",
					`return []gorp.Target{gorp.To("rack", e.Rack)}`,
					"func (i indexes) references() []*gorp.Reference[Key, Device]",
					`const tableName = "device"`,
					`"github.com/synnaxlabs/synnax/core/pkg/service/rack"`,
					"func MatchRacks(vals ...rack.Key) Filter",
					`targets[i] = gorp.To("rack", v)`,
					"return r.indexes.rack.Filter(targets...)",
				).
				ToNotContain("Detach:", "resourceTarget", `"slices"`)
		},
	)

	It("Should register an optional reference to its own type", func(ctx SpecContext) {
		resp := MustGenerate(ctx, device(`
			parent Key? {
				@ref {
					on_delete cascade
					acyclic
				}
			}
		`), "device", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToContain(
				`Tables:   []string{"device"},`,
				"OnDelete: gorp.Cascade,",
				"Acyclic:  true,",
				"if e.Parent == nil {",
				`return []gorp.Target{gorp.To("device", *e.Parent)}`,
			)
	})

	It("Should detach a deleted target from an array", func(ctx SpecContext) {
		resp := MustGenerate(ctx, device(`
			labels label.Key[] @ref on_delete detach
		`), "device", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToContain(
				`"slices"`,
				"targets := make([]gorp.Target, len(e.Labels))",
				"Detach: func(e *Device, t gorp.Target) {",
				"e.Labels = slices.DeleteFunc(e.Labels, func(v label.Key) bool {",
				`return gorp.To("label", v) == t`,
				"func MatchLabel(v label.Key) Filter",
				`return r.indexes.labels.Filter(gorp.To("label", v))`,
			)
	})

	It(
		"Should register a reference to resource IDs of named types",
		func(ctx SpecContext) {
			resp := MustGenerate(ctx, device(`
			owner resource.ID? {
				@ref {
					on_delete cascade
					types     rack device
					unique
				}
			}
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					`Tables:   []string{"rack", "device"},`,
					"Unique:   true,",
					"return []gorp.Target{resourceTarget(*e.Owner)}",
					"func MatchOwners(vals ...resource.ID) Filter",
					"targets[i] = resourceTarget(v)",
					"func resourceTarget(id resource.ID) gorp.Target {",
					"return gorp.Target{Table: string(id.Type), Key: id.Key}",
				)
		},
	)

	DescribeTable(
		"Should reject an invalid reference",
		func(ctx SpecContext, fields, message string) {
			req := MustGenerateRequest(ctx, device(fields), "device", loader)
			Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(message)))
		},
		Entry("without on_delete", `rack rack.Key @ref`,
			"@ref field rack needs on_delete cascade, restrict, or detach"),
		Entry("detaching a required field", `rack rack.Key @ref on_delete detach`,
			"@ref field rack needs to be optional or an array to detach"),
		Entry("to a key that no resource holds", `count uint32 @ref on_delete restrict`,
			"@ref field count needs the key type of a resource, not uint32"),
		Entry("naming types on a key", `rack rack.Key {
				@ref {
					on_delete restrict
					types     rack
				}
			}`,
			"@ref field rack can name types only when it holds resource IDs"),
		Entry(
			"to resource IDs without types",
			`owner resource.ID @ref on_delete cascade`,
			"@ref field owner needs types to name the resource types it may reference",
		),
		Entry("to an unknown resource type", `owner resource.ID {
				@ref {
					on_delete cascade
					types     rock
				}
			}`,
			"@ref field owner names unknown resource type rock"),
		Entry("acyclic across types", `rack rack.Key? {
				@ref {
					on_delete cascade
					acyclic
				}
			}`,
			"@ref field rack can be acyclic only when it references its own type"),
		Entry("with an index", `rack rack.Key {
				@ref on_delete restrict
				@index lookup
			}`,
			"@ref field rack has its own index and filter, so it cannot carry @index"),
		Entry("with a filter", `rack rack.Key {
				@ref on_delete restrict
				@filter
			}`,
			"@ref field rack has its own index and filter, so it cannot carry @index"),
		Entry("with an unknown expression", `rack rack.Key? {
				@ref {
					on_delete cascade
					uniqe
				}
			}`,
			"@ref field rack has unknown expression uniqe"),
		Entry("named search", `search rack.Key {
				@ref on_delete restrict
			}
			name string @search`,
			"@ref field search collides with the search index"),
	)

	It(
		"Should reject a key type that identifies two resources",
		func(ctx SpecContext) {
			loader.Add("schemas/shelf", `
				import "schemas/rack"

				@go output "core/pkg/service/shelf"

				Shelf struct {
					key rack.Key {
						@key
					}
					@resource type "shelf"
				}
			`)
			req := MustGenerateRequest(ctx, `
				import "schemas/rack"
				import "schemas/shelf"

				@go output "core/pkg/service/device"

				Device struct {
					key string {
						@key
					}
					rack  rack.Key @ref on_delete restrict
					shelf shelf.Shelf?
					@resource type "device"
					@retrieve
				}
			`, "device", loader)
			Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
				"@ref field rack has key type rack.Key, which identifies both rack and shelf",
			)))
		},
	)

	It(
		"Should reject a reference on a type that no table stores",
		func(ctx SpecContext) {
			req := MustGenerateRequest(ctx, `
			import "schemas/rack"

			@go output "core/pkg/service/device"

			Config struct {
				rack rack.Key @ref on_delete restrict
			}
		`, "device", loader)
			Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
				"device.Config: @ref field rack needs a @retrieve type to store it",
			)))
		},
	)

	Describe("@retrieve indexes", func() {
		status := func(fields, retrieve string) string {
			return `
			import "schemas/label"
			import "schemas/resource"

			@go output "core/pkg/service/status"

			Key = string

			Status struct<Details?> {
				key Key {
					@key
				}
				details Details
				tag     label.Key?
				source  resource.ID?
				` + fields + `
				` + retrieve + `
			}
		`
		}

		It(
			"Should generate only the indexes of a generic type over any",
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, status(`
				labels label.Key[] @ref on_delete detach
				owner resource.ID? {
					@ref {
						on_delete cascade
						types     label
						unique
					}
				}
			`, "@retrieve indexes"), "status", loader, p)
				ExpectContent(resp, "retrieve.gen.go").
					ToContain(
						"labels *gorp.Reference[Key, Status[any]]",
						"owner *gorp.Reference[Key, Status[any]]",
						"Targets: func(e *Status[any]) []gorp.Target {",
						"Detach: func(e *Status[any], t gorp.Target) {",
						"func (i indexes) references() []*gorp.Reference[Key, Status[any]]",
						"func resourceTarget(id resource.ID) gorp.Target {",
					).
					ToNotContain(
						"type Retrieve struct",
						"type Filter",
						"func MatchKeys",
						`"context"`,
					)
			},
		)

		It("Should accept a primitive key type", func(ctx SpecContext) {
			resp := MustGenerate(ctx, `
				import "schemas/label"

				@go output "core/pkg/service/alias"

				Alias struct {
					tag  label.Key @ref on_delete cascade
					name string

					@retrieve indexes
					@retrieve key "string"
				}
			`, "alias", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain("tag *gorp.Reference[string, Alias]").
				ToNotContain(`"github.com/synnaxlabs/synnax/core/pkg/service/label"`)
		})

		DescribeTable(
			"Should reject",
			func(ctx SpecContext, fields, retrieve, message string) {
				req := MustGenerateRequest(
					ctx,
					status(fields, retrieve),
					"status",
					loader,
				)
				Expect(
					p.Generate(req),
				).Error().
					To(MatchError(ContainSubstring(message)))
			},
			Entry(
				"a generic type with a generated Retrieve",
				`labels label.Key[] @ref on_delete detach`,
				"@retrieve",
				"core/pkg/service/status: status.Status: a generic @retrieve type needs @retrieve indexes",
			),
			Entry(
				"a filter with no generated Retrieve",
				`name string @filter`,
				"@retrieve indexes",
				"@filter field name needs a generated Retrieve, not @retrieve indexes",
			),
			Entry(
				"a type with no index",
				`name string`,
				"@retrieve indexes",
				"@retrieve indexes needs a field with @index, @ref, or @search",
			),
		)
	})
})
