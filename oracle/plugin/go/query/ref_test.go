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
})
