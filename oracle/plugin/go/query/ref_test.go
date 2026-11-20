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
	"strings"

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
		loader.Add("schemas/swatch", `
			import "schemas/resource"

			@go output "core/pkg/service/swatch"

			Key = uuid

			Swatch struct {
				key   Key {
					@key
				}
				value string
				@resource type "swatch"
			}

			Ref union on type {
				named {
					key Key @ref on_delete inline @ref kind use
				}
				fixed {
					value string
				}
			}

			Spot union on type {
				named {
					key Key? @ref on_delete inline @ref kind use
				}
			}

			Pin union on type {
				named {
					key Key @ref on_delete inline @ref kind use
				}
			}

			Band union on type {
				owned {
					owner resource.ID {
						@ref on_delete inline
						@ref kind      use
						@ref types     swatch
					}
				}
			}
		`)
		loader.Add("schemas/gauge", `
			@go output "core/pkg/service/gauge"

			Key = uint16

			Gauge struct<T> {
				key   Key {
					@key
				}
				value T?
				@resource type "gauge"
			}

			Ref union on type {
				named {
					key Key @ref on_delete inline @ref kind use
				}
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
		func() {
			resp := MustGenerate(device(`
			rack rack.Key @ref on_delete restrict @ref kind use
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					"rack *gorp.KeyReference[Key, Device, rack.Key]",
					"rack: gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, Device, rack.Key]{",
					`Name:     "rack",`,
					`Table:    "rack",`,
					"OnDelete: gorp.Restrict,",
					"Targets: func(e *Device) []rack.Key {",
					"return []rack.Key{e.Rack}",
					"func (i indexes) references() []gorp.Reference[Key, Device]",
					`const tableName = "device"`,
					`"github.com/synnaxlabs/synnax/core/pkg/service/rack"`,
					"func MatchRacks(vals ...rack.Key) Filter",
					"return r.indexes.rack.Filter(vals...)",
				).
				ToNotContain("Detach:", "resourceTarget", `"slices"`, "gorp.Target")
		},
	)

	DescribeTable(
		"Should declare the kind of a reference",
		func(field, kind string) {
			resp := MustGenerate(device(field), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain("Kind:     gorp." + kind + ",")
		},
		Entry("a place on a key", `rack rack.Key {
				@ref {
					on_delete restrict
					kind      place
				}
			}`, "Place"),
		Entry("a use on an array", `labels label.Key[] {
				@ref {
					on_delete detach
					kind      use
				}
			}`, "Use"),
		Entry("a use on resource IDs", `owner resource.ID? {
				@ref {
					on_delete cascade
					types     rack
					kind      use
				}
			}`, "Use"),
	)

	union := func(schema, field string) string {
		var imports strings.Builder
		for dep := range strings.SplitSeq(schema, ",") {
			imports.WriteString("import \"schemas/" + dep + "\"\n")
		}
		return `
			` + imports.String() + `

			@go output "core/pkg/service/device"

			Key = string

			Base struct {
				key Key @key
				` + field + `
			}

			Device union on model extends Base {
				t4 {}
				t7 {}
				@resource type "device"
				@retrieve
			}
		`
	}

	DescribeTable(
		"Should declare a reference on the base of a union once",
		func(schema, field, name, targets, match string) {
			resp := MustGenerate(union(schema, field), "device", loader, p)
			content := MustContentOf(resp, "retrieve.gen.go")
			Expect(strings.Count(content, `Name:     "`+name+`",`)).To(Equal(1))
			Expect(content).To(ContainSubstring(targets))
			Expect(content).To(ContainSubstring(match))
		},
		Entry(
			"a restrict key",
			"rack",
			`rack rack.Key @ref on_delete restrict @ref kind place`,
			"rack",
			"targets = append(targets, v1.Rack)",
			"func MatchRacks(",
		),
		Entry(
			"a cascade optional key",
			"label",
			`label label.Key? @ref on_delete cascade @ref kind use`,
			"label",
			"targets = append(targets, *v1.Label)",
			"func MatchLabels(",
		),
	)

	It("Should refuse to inline a key that a struct holds", func() {
		req := MustGenerateRequest(device(`
				spare rack.Key @ref on_delete inline @ref kind use
			`), "device", loader)
		Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
			"Device: @ref field spare needs a union variant to hold it to inline",
		)))
	})

	It("Should declare a resource target for an ID reference on a union", func() {
		resp := MustGenerate(union(
			"resource",
			`owner resource.ID? @ref on_delete cascade @ref types device @ref kind use`,
		), "device", loader, p)
		ExpectContent(resp, "retrieve.gen.go").ToContain(
			"func resourceTarget(id resource.ID) gorp.Target {",
			"targets[i] = resourceTarget(v)",
		)
	})

	It("Should name the index of a union reference after the field", func() {
		resp := MustGenerate(union(
			"rack",
			`rack rack.Key {
				@ref on_delete restrict
				@ref kind place
				@go name "RackKey"
			}`,
		), "device", loader, p)
		content := MustContentOf(resp, "retrieve.gen.go")
		Expect(content).To(ContainSubstring("r.indexes.rack.Filter("))
		Expect(content).ToNot(ContainSubstring("r.indexes.rackKey"))
	})

	It("Should refuse a detach reference on a union", func() {
		req := MustGenerateRequest(
			union("label", `labels label.Key[] @ref on_delete detach @ref kind use`),
			"device",
			loader,
		)
		Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
			"device.Device: @ref on_delete detach is not supported on the base " +
				"fields of a union (labels)",
		)))
	})

	It("Should register an optional reference to its own type", func() {
		resp := MustGenerate(device(`
			parent Key? {
				@ref {
					on_delete cascade
					kind      use
					acyclic
				}
			}
		`), "device", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToContain(
				`Table:    "device",`,
				"OnDelete: gorp.Cascade,",
				"Acyclic:  true,",
				"if e.Parent == nil {",
				"return []Key{*e.Parent}",
			)
	})

	Describe("union record", func() {
		union := func(dep, variant string) string {
			return `
				import "schemas/` + dep + `"

				@go output "core/pkg/service/device"

				Base struct {
					key uuid @key
				}

				Device union on kind extends Base {
					physical {
						` + variant + `
					}
					virtual {}
					@resource type "device"
					@retrieve
				}
			`
		}

		It(
			"Should register a reference that a variant of the union holds",
			func() {
				resp := MustGenerate(union("rack", `
					rack rack.Key? @ref on_delete restrict @ref kind use
				`), "device", loader, p)
				ExpectContent(resp, "retrieve.gen.go").
					ToContain(
						"func (i indexes) references() []gorp.Reference[uuid.UUID, Device]",
						`Name:     "rack",`,
						`Table:    "rack",`,
						"OnDelete: gorp.Restrict,",
						"switch v1 := e.Variant.(type) {",
						"case PhysicalDevice:",
						"if v1.Rack != nil {",
					).
					ToNotContain("case VirtualDevice:")
			},
		)

		It(
			"Should write the variant back after a detach clears its reference",
			func() {
				resp := MustGenerate(union("label", `
					tag label.Key? @ref on_delete detach @ref kind use
				`), "device", loader, p)
				ExpectContent(resp, "retrieve.gen.go").ToContain(
					"OnDelete: gorp.Detach,",
					"case PhysicalDevice:",
					"v2.Tag = nil",
					"e.Variant = v2",
				)
			},
		)

		It(
			"Should reject a variant reference that collides with the search index",
			func() {
				req := MustGenerateRequest(`
					import "schemas/rack"

					@go output "core/pkg/service/device"

					Base struct {
						key  uuid   @key
						name string @search
					}

					Device union on kind extends Base {
						physical {
							search rack.Key? @ref on_delete restrict @ref kind use
						}
						virtual {}
						@resource type "device"
						@retrieve
					}
				`, "device", loader)
				Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
					"@ref field search collides with the search index",
				)))
			},
		)
	})

	It("Should detach a deleted target from an array", func() {
		resp := MustGenerate(device(`
			labels label.Key[] @ref on_delete detach @ref kind use
		`), "device", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToContain(
				`"slices"`,
				"Targets: func(e *Device) []label.Key {",
				"return e.Labels",
				"Detach: func(e *Device, t label.Key) {",
				"e.Labels = slices.DeleteFunc(e.Labels, func(v label.Key) bool {",
				"return v == t",
				"func MatchLabels(vals ...label.Key) Filter",
				"return r.indexes.labels.Filter(vals...)",
				"func MatchHasLabels(has bool) Filter",
				"return r.indexes.labels.FilterHolding(has)",
			).
			ToNotContain("func MatchLabel(", "func setLabels")
	})

	It("Should detach an optional field and write it", func() {
		resp := MustGenerate(device(`
			shelf rack.Key? @ref on_delete detach @ref kind use
		`), "device", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToContain(
				"Detach: func(e *Device, _ rack.Key) { e.Shelf = nil },",
				"func MatchHasShelf(has bool) Filter",
				"func setShelf(",
				"table *gorp.Table[Key, Device],",
				"v *rack.Key,",
				"keys ...Key,",
				") gorp.Update[Key, Device] {",
				"Where(gorp.MatchKeys[Key, Device](keys...)).",
				"e.Shelf = v",
			)
	})

	DescribeTable(
		"Should hold no target while a field with a zero default is zero",
		func(field, zero string) {
			resp := MustGenerate(device(field), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					"if e.Spare == "+zero+" {",
					"return nil",
					"Detach: func(e *Device, _ ",
					"e.Spare = "+strings.Trim(zero, "()")+" },",
					"func MatchHasSpare(has bool) Filter",
					"func setSpare(",
				)
		},
		Entry(
			"an integer key",
			`spare rack.Key = 0 @ref on_delete detach @ref kind use`,
			"0",
		),
		Entry(
			"a string key",
			`spare Key = "" @ref on_delete detach @ref kind use`,
			`""`,
		),
		Entry(
			"a UUID key",
			`spare label.Key = "" @ref on_delete detach @ref kind use`,
			"(label.Key{})",
		),
	)

	It("Should always hold the target of a required field", func() {
		resp := MustGenerate(device(`
			rack rack.Key @ref on_delete cascade @ref kind use
		`), "device", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToNotContain("func MatchHasRack", "func setRack", "if e.Rack ==")
	})

	It(
		"Should register a reference to resource IDs of named types",
		func() {
			resp := MustGenerate(device(`
			owner resource.ID? {
				@ref {
					on_delete cascade
					kind      use
					types     rack device
					unique
				}
			}
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					"owner *gorp.TargetReference[Key, Device]",
					"owner: gorp.NewTargetReference(gorp.TargetReferenceConfig[Key, Device]{",
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
		func(fields, message string) {
			req := MustGenerateRequest(device(fields), "device", loader)
			Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(message)))
		},
		Entry(
			"without on_delete",
			`rack rack.Key @ref`,
			"@ref field rack needs on_delete cascade, restrict, detach, inline, or weak",
		),
		Entry("without a kind", `rack rack.Key @ref on_delete restrict`,
			"@ref field rack needs kind place or use"),
		Entry(
			"detaching a required field",
			`rack rack.Key @ref on_delete detach @ref kind use`,
			"@ref field rack needs to be optional, an array, or zero by default to detach",
		),
		Entry(
			"detaching a field with a default that is not zero",
			`rack rack.Key = 4 @ref on_delete detach @ref kind use`,
			"@ref field rack needs to be optional, an array, or zero by default to detach",
		),
		Entry(
			"to a key that no resource holds",
			`count uint32 @ref on_delete restrict @ref kind use`,
			"@ref field count needs the key type of a resource, not uint32",
		),
		Entry("naming types on a key", `rack rack.Key {
				@ref {
					on_delete restrict
					kind      use
					types     rack
				}
			}`,
			"@ref field rack can name types only when it holds resource IDs"),
		Entry(
			"to resource IDs without types",
			`owner resource.ID @ref on_delete cascade @ref kind use`,
			"@ref field owner needs types to name the resource types it may reference",
		),
		Entry("to an unknown resource type", `owner resource.ID {
				@ref {
					on_delete cascade
					kind      use
					types     rock
				}
			}`,
			"@ref field owner names unknown resource type rock"),
		Entry("acyclic across types", `rack rack.Key? {
				@ref {
					on_delete cascade
					kind      use
					acyclic
				}
			}`,
			"@ref field rack can be acyclic only when it references its own type"),
		Entry("with an index", `rack rack.Key {
				@ref on_delete restrict
				@ref kind use
				@index lookup
			}`,
			"@ref field rack has its own index and filter, so it cannot carry @index"),
		Entry("with a filter", `rack rack.Key {
				@ref on_delete restrict
				@ref kind use
				@filter
			}`,
			"@ref field rack has its own index and filter, so it cannot carry @index"),
		Entry("with an unknown expression", `rack rack.Key? {
				@ref {
					on_delete cascade
					kind      use
					uniqe
				}
			}`,
			"@ref field rack has unknown expression uniqe"),
		Entry("with an unknown kind", `rack rack.Key {
				@ref {
					on_delete restrict
					kind      spot
				}
			}`,
			"@ref field rack needs kind place or use"),
		Entry(
			"with a kind on the on_delete line",
			`rack rack.Key @ref on_delete restrict kind use`,
			"@ref field rack needs one value for on_delete, not 3",
		),
		Entry("with two kinds", `rack rack.Key {
				@ref {
					on_delete restrict
					kind      use place
				}
			}`,
			"@ref field rack needs one value for kind, not 2"),
		Entry(
			"with a flag before another flag",
			`rack rack.Key @ref acyclic unique`,
			"@ref field rack needs no value for acyclic, not 1",
		),
		Entry("named search", `search rack.Key {
				@ref on_delete restrict
				@ref kind use
			}
			name string @search`,
			"@ref field search collides with the search index"),
	)

	It(
		"Should reject a key type that identifies two resources",
		func() {
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
			req := MustGenerateRequest(`
				import "schemas/rack"
				import "schemas/shelf"

				@go output "core/pkg/service/device"

				Device struct {
					key string {
						@key
					}
					rack  rack.Key @ref on_delete restrict @ref kind use
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
		func() {
			req := MustGenerateRequest(`
			import "schemas/rack"

			@go output "core/pkg/service/device"

			Config struct {
				rack rack.Key @ref on_delete restrict @ref kind use
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
			func() {
				resp := MustGenerate(status(`
				labels label.Key[] @ref on_delete detach @ref kind use
				owner resource.ID? {
					@ref {
						on_delete cascade
						kind      use
						types     label
						unique
					}
				}
			`, "@retrieve indexes"), "status", loader, p)
				ExpectContent(resp, "retrieve.gen.go").
					ToContain(
						"labels *gorp.KeyReference[Key, Status[any], label.Key]",
						"owner *gorp.TargetReference[Key, Status[any]]",
						"Targets: func(e *Status[any]) []label.Key {",
						"Detach: func(e *Status[any], t label.Key) {",
						"Targets: func(e *Status[any]) []gorp.Target {",
						"func (i indexes) references() []gorp.Reference[Key, Status[any]]",
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

		It("Should import context for a reference that inlines", func() {
			resp := MustGenerate(`
				import "schemas/swatch"

				@go output "core/pkg/service/look"

				Look struct {
					key  uuid {
						@key
					}
					fill swatch.Ref?

					@retrieve indexes
				}
			`, "look", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(`"context"`, "ctx context.Context,").
				ToNotContain("type Retrieve struct")
		})

		It("Should accept a primitive key type", func() {
			resp := MustGenerate(`
				import "schemas/label"

				@go output "core/pkg/service/alias"

				Alias struct {
					tag  label.Key @ref on_delete cascade @ref kind use
					name string

					@retrieve indexes
					@retrieve key "string"
				}
			`, "alias", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					"tag *gorp.KeyReference[string, Alias, label.Key]",
					`"github.com/synnaxlabs/synnax/core/pkg/service/label"`,
				)
		})

		DescribeTable(
			"Should reject",
			func(fields, retrieve, message string) {
				req := MustGenerateRequest(
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
				`labels label.Key[] @ref on_delete detach @ref kind use`,
				"@retrieve",
				"core/pkg/service/status: status.Status: a generic @retrieve type needs @retrieve indexes",
			),
			Entry(
				"a filter with no generated Retrieve",
				`name string @filter`,
				"@retrieve indexes",
				"@retrieve indexes generates no matchers, so field name cannot take "+
					"@filter",
			),
			Entry(
				"a type with no index",
				`name string`,
				"@retrieve indexes",
				"@retrieve indexes needs a field with @index, @ref, or @search",
			),
		)
	})
	Describe("@retrieve references", func() {
		BeforeEach(func() {
			loader.Add("schemas/channel", `
				@go output "core/pkg/service/channel"

				Key = uint32

				Channel struct {
					key Key {
						@key
					}
					@resource type "channel"
				}
			`)
			loader.Add("schemas/kv", `
				@go output "core/pkg/service/kv"

				Key = string

				Entry struct {
					key Key {
						@key
					}
					@resource type "kv"
				}
			`)
		})

		config := func(types, fields string) string {
			var imports strings.Builder
			namespaces := []string{
				"rack", "label", "channel", "kv", "resource", "swatch", "gauge",
			}
			for _, ns := range namespaces {
				if strings.Contains(types+fields, ns+".") {
					imports.WriteString(`import "schemas/` + ns + `"` + "\n")
				}
			}
			return imports.String() + `
			@go output "core/pkg/service/modbus"

			` + types + `

			ReadConfig struct {
				key uuid {
					@key
				}
				` + fields + `
				@retrieve references
			}
		`
		}

		generate := func(types, fields string) *ContentExpectation {
			GinkgoHelper()
			resp := MustGenerate(config(types, fields), "modbus", loader, p)
			return ExpectContent(resp, "retrieve.gen.go")
		}

		It("Should skip a zero key and reset it on detach", func() {
			generate("", `
				rack rack.Key = 0 @ref on_delete restrict @ref kind use
			`).
				ToContain(
					"func readConfigReferences() []gorp.Reference[uuid.UUID, ReadConfig]",
					`Name:     "rack",`,
					`Table:    "rack",`,
					"OnDelete: gorp.Restrict,",
					"if e.Rack != 0 {",
					`targets = append(targets, e.Rack)`,
					"Detach: func(e *ReadConfig, t rack.Key) {",
					`if e.Rack == t {`,
					"e.Rack = 0",
				).
				ToNotContain(
					"type Retrieve struct",
					"type indexes struct",
					`"context"`,
					`"slices"`,
				)
		})

		It(
			"Should inline a deleted target in an optional union and detach a missing one",
			func() {
				generate("", `
				fill swatch.Ref?
			`).
					ToContain(
						`"context"`,
						`Name:     "fill.key",`,
						`Table:    "swatch",`,
						"OnDelete: gorp.Inline,",
						"ctx context.Context,",
						"tx gorp.Tx,",
						"e *ReadConfig,",
						"t swatch.Key,",
						"var target swatch.Swatch",
						"if err := gorp.NewRetrieve[swatch.Key, swatch.Swatch]().",
						"Where(gorp.MatchKeys[swatch.Key, swatch.Swatch](t)).",
						"Entry(&target).",
						"if e.Fill != nil {",
						"if v2, ok := e.Fill.Variant.(swatch.NamedRef); ok && v2.Key == t {",
						"e.Fill.Variant = target.Inline()",
						"return nil",
						"Detach: func(e *ReadConfig, t swatch.Key) {",
						"e.Fill = nil",
					).
					ToNotContain("FixedRef")
			},
		)

		It("Should inline a deleted target in each union of an array", func() {
			generate("", `
				fills swatch.Ref[]
			`).
				ToContain(
					"for i3 := range e.Fills {",
					"if v4, ok := e.Fills[i3].Variant.(swatch.NamedRef); ok && v4.Key == t {",
					"e.Fills[i3].Variant = target.Inline()",
				).
				ToNotContain("Detach:", `"slices"`)
		})

		It(
			"Should write back the variant that holds a union it inlines",
			func() {
				generate(`
				Marker union on type {
					spot {
						fill swatch.Ref?
					}
					plain {}
				}
			`, `
				marker Marker
			`).
					ToContain(
						"switch v3 := e.Marker.Variant.(type) {",
						"case SpotMarker:",
						"if v3.Fill != nil {",
						"if v4, ok := v3.Fill.Variant.(swatch.NamedRef); ok && v4.Key == t {",
						"v3.Fill.Variant = target.Inline()",
						"e.Marker.Variant = v3",
					).
					ToNotContain("case PlainMarker:")
			},
		)

		It("Should declare the kind of a nested reference", func() {
			generate(`
				Inner struct {
					channel channel.Key = 0 {
						@ref {
							on_delete restrict
							kind      use
						}
					}
				}
			`, `
				inner Inner
			`).
				ToContain(`Name:     "inner.channel",`, "Kind:     gorp.Use,")
		})

		It("Should reset a zero string key", func() {
			generate("", `
				entry kv.Key = "" @ref on_delete restrict @ref kind use
			`).
				ToContain(`if e.Entry != "" {`, `e.Entry = ""`)
		})

		It("Should reset a zero UUID key to its composite zero", func() {
			generate("", `
				tag label.Key = "" @ref on_delete restrict @ref kind use
			`).
				ToContain("if e.Tag != (label.Key{}) {", "e.Tag = label.Key{}")
		})

		It("Should reach a key through a nested struct", func() {
			generate(`
				Inner struct {
					channel channel.Key = 0 @ref on_delete restrict @ref kind use
				}
			`, `
				inner Inner
			`).
				ToContain(
					`Name:     "inner.channel",`,
					`Table:    "channel",`,
					"if e.Inner.Channel != 0 {",
					`if e.Inner.Channel == t {`,
					"e.Inner.Channel = 0",
				)
		})

		It("Should reach a key through an optional struct", func() {
			generate(`
				Inner struct {
					channel channel.Key = 0 @ref on_delete restrict @ref kind use
				}
			`, `
				inner Inner?
			`).
				ToContain("if e.Inner != nil {", "if e.Inner.Channel != 0 {")
		})

		It("Should range over an array of structs", func() {
			generate(`
				Entry struct {
					channel channel.Key = 0 @ref on_delete restrict @ref kind use
				}
			`, `
				entries Entry[]
			`).
				ToContain(
					`Name:     "entries.channel",`,
					"var targets []channel.Key",
					"for _, v1 := range e.Entries {",
					"if targets == nil {",
					"targets = make([]channel.Key, 0, len(e.Entries))",
					"if v1.Channel != 0 {",
					"for i2 := range e.Entries {",
					`if e.Entries[i2].Channel == t {`,
					"e.Entries[i2].Channel = 0",
				)
		})

		It(
			"Should switch over every variant of a union that holds the key",
			func() {
				generate(`
				Base struct {
					channel channel.Key = 0 @ref on_delete restrict @ref kind use
				}
				Channel union on type extends Base {
					coil {}
					register {
						address uint16
					}
				}
			`, `
				channels Channel[]
			`).
					ToContain(
						`Name:     "channels.channel",`,
						"switch v2 := v1.Variant.(type) {",
						"case CoilChannel:",
						"case RegisterChannel:",
						"if v2.Channel != 0 {",
						"switch v4 := e.Channels[i3].Variant.(type) {",
						"e.Channels[i3].Variant = v4",
					)
			},
		)

		It(
			"Should reach a key that only some variants of a union declare",
			func() {
				generate(`
				Entry union on type {
					plain {
						topic string
					}
					sparkplug {
						channel channel.Key = 0 @ref on_delete restrict @ref kind use
					}
				}
			`, `
				entries Entry[]
			`).
					ToContain("case SparkplugEntry:", "if v2.Channel != 0 {").
					ToNotContain("case PlainEntry:")
			},
		)

		It(
			"Should drop the array element that holds a required key",
			func() {
				generate(`
				Alert struct {
					tag label.Key @ref on_delete detach @ref kind use
					component string
				}
			`, `
				alerts Alert[]
			`).
					ToContain(
						`"slices"`,
						`Name:     "alerts.tag",`,
						"OnDelete: gorp.Detach,",
						`targets = append(targets, v1.Tag)`,
						"e.Alerts = slices.DeleteFunc(e.Alerts, func(v2 Alert) bool {",
						`if v2.Tag == t {`,
						"return true",
						"return false",
					)
			},
		)

		It("Should detach a key from a nested array of keys", func() {
			generate(`
				Axes struct {
					y1 channel.Key[] @ref on_delete detach @ref kind use
				}
			`, `
				axes Axes
			`).
				ToContain(
					"var targets []channel.Key",
					"for _, k1 := range e.Axes.Y1 {",
					"e.Axes.Y1 = slices.DeleteFunc(e.Axes.Y1, func(k2 channel.Key) bool {",
					`"github.com/synnaxlabs/synnax/core/pkg/service/channel"`,
				)
		})

		It("Should clear a nested optional key", func() {
			generate(`
				Inner struct {
					owner resource.ID? {
						@ref {
							on_delete detach
							kind      use
							types     rack
						}
					}
				}
			`, `
				inner Inner
				rack  rack.Key = 0 @ref on_delete restrict @ref kind use
			`).
				ToContain(
					`Name:     "inner.owner",`,
					`Tables:   []string{"rack"},`,
					"if e.Inner.Owner != nil {",
					"Key: (*e.Inner.Owner).Key}",
					"e.Inner.Owner = nil",
				).
				ToNotContain("resourceTarget")
		})

		It(
			"Should leave out Detach when a required key has no array",
			func() {
				generate("", `
				rack rack.Key @ref on_delete restrict @ref kind use
			`).
					ToContain(`targets = append(targets, e.Rack)`).
					ToNotContain("Detach:")
			},
		)

		It(
			"Should skip a recursive type that holds no reference",
			func() {
				generate(`
				Node struct {
					children Node[]
				}
			`, `
				tree Node
				rack rack.Key = 0 @ref on_delete restrict @ref kind use
			`).
					ToContain(`Name:     "rack",`).
					ToNotContain("tree.")
			},
		)

		It(
			"Should generate the references that the variants of a union hold",
			func() {
				resp := MustGenerate(`
					import "schemas/rack"

					@go output "core/pkg/service/modbus"

					Base struct {
						key uuid @key
					}

					ReadConfig union on kind extends Base {
						serial {
							rack rack.Key = 0 @ref on_delete restrict @ref kind use
						}
						tcp {}
						@retrieve references
					}
				`, "modbus", loader, p)
				ExpectContent(resp, "retrieve.gen.go").ToContain(
					"func readConfigReferences() []gorp.Reference[uuid.UUID, ReadConfig]",
					`Name:     "rack",`,
					"case SerialReadConfig:",
				)
			},
		)

		It(
			"Should generate the references of several types in one package",
			func() {
				resp := MustGenerate(config(`
				WriteConfig struct {
					key uuid {
						@key
					}
					rack rack.Key = 0 @ref on_delete restrict @ref kind use
					@retrieve references
				}
			`, `
				rack rack.Key = 0 @ref on_delete restrict @ref kind use
			`), "modbus", loader, p)
				ExpectContent(resp, "retrieve.gen.go").ToContain(
					"func readConfigReferences()",
					"func writeConfigReferences()",
				)
			},
		)

		DescribeTable(
			"Should reject",
			func(types, fields, message string) {
				req := MustGenerateRequest(config(types, fields), "modbus", loader)
				Expect(
					p.Generate(req),
				).Error().
					To(MatchError(ContainSubstring(message)))
			},
			Entry(
				"a recursive type that holds a reference",
				`Node struct {
					rack     rack.Key = 0 @ref on_delete restrict @ref kind use
					children Node[]
				}`,
				`tree Node`,
				"@ref fields inside the recursive type Node are not supported",
			),
			Entry(
				"detaching a required key outside an array",
				`Inner struct {
					rack rack.Key @ref on_delete detach @ref kind use
				}`,
				`inner Inner`,
				"@ref field inner.rack needs to be optional, an array, or zero by "+
					"default to detach",
			),
			Entry(
				"an acyclic reference",
				"",
				`rack rack.Key? {
					@ref {
						on_delete cascade
						kind      use
						acyclic
					}
				}`,
				"@ref field rack can be acyclic only when it references its own type",
			),
			Entry("a type with no reference", "", `name string`,
				"@retrieve references needs a @ref field"),
			Entry(
				"inlining a key that no union variant holds",
				"",
				`swatch swatch.Key @ref on_delete inline @ref kind use`,
				"@ref field swatch is not a field of a union variant that a field "+
					"holds, so it cannot inline",
			),
			Entry(
				"inlining one key path through different unions",
				`Marker union on type {
					spot {
						fill swatch.Ref
					}
					pin {
						fill swatch.Pin
					}
				}`,
				`marker Marker`,
				"@ref field marker.fill.key sits in different unions, so it cannot inline",
			),
			Entry(
				"inlining a key of a generic type",
				"",
				`fill gauge.Ref`,
				"@ref field fill.key cannot inline the generic type Gauge",
			),
			Entry(
				"inlining an optional key",
				"",
				`spots swatch.Spot[]`,
				"@ref field spots.key is not required, so it cannot inline",
			),
			Entry(
				"inlining a resource ID",
				"",
				`band swatch.Band`,
				"@ref field band.owner needs the key type of one resource to inline",
			),
			Entry(
				"union variants that give one field different references",
				`Channel union on type {
					coil {
						channel channel.Key = 0 @ref on_delete restrict @ref kind use
					}
					register {
						channel channel.Key = 0 @ref on_delete detach @ref kind use
					}
				}`,
				`channels Channel[]`,
				"@ref field channels.channel declares different references in its "+
					"variants",
			),
			Entry(
				"union variants that give one field different kinds",
				`Channel union on type {
					coil {
						channel channel.Key = 0 {
							@ref {
								on_delete restrict
								kind      use
							}
						}
					}
					register {
						channel channel.Key = 0 {
							@ref {
								on_delete restrict
								kind      place
							}
						}
					}
				}`,
				`channels Channel[]`,
				"@ref field channels.channel declares different references in its "+
					"variants",
			),
		)

		It(
			"Should check cycles through a nested reference to its own type",
			func() {
				resp := MustGenerate(device(`
				inner Inner
			`)+`
				Inner struct {
					parent Key? {
						@ref {
							on_delete cascade
							kind      use
							acyclic
						}
					}
				}
			`, "device", loader, p)
				ExpectContent(resp, "retrieve.gen.go").ToContain(
					`Name:     "inner.parent",`,
					`Table:    "device",`,
					"Acyclic:  true,",
				)
			},
		)

		It(
			"Should store a nested reference on a type that generates a Retrieve",
			func() {
				resp := MustGenerate(device(`
				inner Inner[]
			`)+`
				Inner struct {
					rack rack.Key @ref on_delete detach @ref kind use
				}
			`, "device", loader, p)
				ExpectContent(resp, "retrieve.gen.go").ToContain(
					"innerRack *gorp.KeyReference[Key, Device, rack.Key]",
					"innerRack: gorp.NewKeyReference(",
					`Name:     "inner.rack",`,
					"e.Inner = slices.DeleteFunc(",
					"i.innerRack,",
				)
			},
		)

		Describe("weak and direction", func() {
			It(
				"Should reset a weak key that holds none at zero in its Detach",
				func() {
					generate(`
				Chan struct {
					channel channel.Key = 0 @ref on_delete weak @ref kind use
				}
			`, `
				channels Chan[]
			`).
						ToContain(
							`Name:     "channels.channel",`,
							"OnDelete: gorp.Weak,",
							"if v1.Channel != 0 {",
							"Detach: func(e *ReadConfig, t channel.Key) {",
							"if e.Channels[i2].Channel == t {",
							"e.Channels[i2].Channel = 0",
						).
						ToNotContain(`"slices"`, "Tag:")
				},
			)

			It("Should drop the element of a required weak key in its Detach", func() {
				generate(`
				Binding struct {
					name    string
					channel channel.Key @ref on_delete weak @ref kind use
				}
			`, `
				bindings Binding[]
			`).
					ToContain(
						"OnDelete: gorp.Weak,",
						"e.Bindings = slices.DeleteFunc(e.Bindings, func(v2 Binding) bool {",
						"if v2.Channel == t {",
					)
			})

			It("Should write the direction of a nested reference into its tag", func() {
				generate(`
				Chan struct {
					channel channel.Key = 0 {
						@ref on_delete restrict
						@ref kind      use
						@ref direction write
					}
				}
			`, `
				channels Chan[]
			`).
					ToContain(
						`Name:     "channels.channel",`,
						"Tag:      2, // direction write",
					)
			})

			It(
				"Should give the direction of a field to every reference below it",
				func() {
					content := MustContentOf(MustGenerate(config(`
				Chan struct {
					state   channel.Key = 0 @ref on_delete weak @ref kind use
					command channel.Key = 0 @ref on_delete weak @ref kind use
				}
				Group struct {
					channels Chan[]
				}
			`, `
				group Group @ref direction read
			`), "modbus", loader, p), "retrieve.gen.go")
					Expect(strings.Count(content, "Tag:      1, // direction read")).
						To(Equal(2))
					Expect(
						content,
					).To(ContainSubstring(`Name:     "group.channels.state",`))
					Expect(
						content,
					).To(ContainSubstring(`Name:     "group.channels.command",`))
				},
			)

			DescribeTable(
				"Should reject",
				func(types, fields, message string) {
					req := MustGenerateRequest(config(types, fields), "modbus", loader)
					Expect(
						p.Generate(req),
					).Error().
						To(MatchError(ContainSubstring(message)))
				},
				Entry(
					"a direction under a field that sets one",
					`Chan struct {
					channel channel.Key = 0 {
						@ref on_delete weak
						@ref kind      use
						@ref direction write
					}
				}`,
					`channels Chan[] @ref direction read`,
					"modbus.ReadConfig: @ref fields channels and channels.channel both "+
						"set a direction on one path",
				),
				Entry(
					"two fields above a reference that set a direction",
					`Chan struct {
					channel channel.Key = 0 @ref on_delete weak @ref kind use
				}
				Group struct {
					channels Chan[] @ref direction write
				}`,
					`group Group @ref direction write`,
					"modbus.ReadConfig: @ref fields group and group.channels both set a "+
						"direction on one path",
				),
				Entry(
					"variants that give one reference two directions",
					`Port union on type {
					input {
						channel channel.Key = 0 {
							@ref on_delete weak
							@ref kind      use
							@ref direction read
						}
					}
					output {
						channel channel.Key = 0 {
							@ref on_delete weak
							@ref kind      use
							@ref direction write
						}
					}
				}`,
					`port Port`,
					"@ref field port.channel declares different references in its "+
						"variants",
				),
				Entry(
					"a direction on a field that holds no reference",
					`Chan struct {
					name string
				}`,
					`channels Chan[] @ref direction read
				rack rack.Key = 0 @ref on_delete restrict @ref kind use`,
					"modbus.ReadConfig: @ref field channels sets a direction but holds "+
						"no @ref field",
				),
				Entry(
					"a direction other than read or write",
					`Chan struct {
					channel channel.Key = 0 @ref on_delete weak @ref kind use
				}`,
					`channels Chan[] @ref direction both`,
					"modbus.ReadConfig: @ref field channels needs direction read or write",
				),
				Entry(
					"a reference with a direction other than read or write",
					"",
					`rack rack.Key = 0 {
					@ref on_delete restrict
					@ref kind      use
					@ref direction both
				}`,
					"modbus.ReadConfig: @ref field rack needs direction read or write",
				),
			)
		})

		Describe("Retarget", func() {
			It("Should move a zero key in place", func() {
				generate("", `
				rack rack.Key = 0 @ref on_delete restrict @ref kind use
			`).
					ToContain(
						"Retarget: func(e *ReadConfig, to map[rack.Key]rack.Key) {",
						"if t, ok := to[e.Rack]; ok {",
						"e.Rack = t",
					).
					ToBeValidGoSource()
			})

			It("Should move the key of each element of an array in place", func() {
				generate(`
				Binding struct {
					name    string
					channel channel.Key @ref on_delete weak @ref kind use
				}
			`, `
				bindings Binding[]
			`).
					ToContain(
						"Retarget: func(e *ReadConfig, to map[channel.Key]channel.Key) {",
						"for i3 := range e.Bindings {",
						"if t, ok := to[e.Bindings[i3].Channel]; ok {",
						"e.Bindings[i3].Channel = t",
					).
					ToBeValidGoSource()
			})

			It("Should keep the position of each key of a nested array", func() {
				generate(`
				Group struct {
					channels channel.Key[] @ref on_delete weak @ref kind use
				}
			`, `
				groups Group[]
			`).
					ToContain(
						"for i5 := range e.Groups {",
						"for i6, k7 := range e.Groups[i5].Channels {",
						"if t, ok := to[k7]; ok {",
						"e.Groups[i5].Channels[i6] = t",
					).
					ToBeValidGoSource()
			})

			It("Should give a moved optional key a new pointer", func() {
				generate(`
				Port struct {
					spare rack.Key? @ref on_delete detach @ref kind use
				}
			`, `
				port Port?
			`).
					ToContain(
						"if e.Port != nil {",
						"if e.Port.Spare != nil {",
						"if t, ok := to[*e.Port.Spare]; ok {",
						"e.Port.Spare = &t",
					).
					ToBeValidGoSource()
			})

			It("Should write back each union variant that holds a key", func() {
				generate("", `
				fills swatch.Ref[]
			`).
					ToContain(
						"Retarget: func(e *ReadConfig, to map[swatch.Key]swatch.Key) {",
						"for i5 := range e.Fills {",
						"switch v6 := e.Fills[i5].Variant.(type) {",
						"case swatch.NamedRef:",
						"if t, ok := to[v6.Key]; ok {",
						"v6.Key = t",
						"e.Fills[i5].Variant = v6",
					).
					ToBeValidGoSource()
			})
		})
	})

	Describe("weak and direction on a retrieved type", func() {
		It("Should register a weak reference with no Detach", func() {
			resp := MustGenerate(device(`
			spare rack.Key @ref on_delete weak @ref kind use
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					`Name:     "spare",`,
					"OnDelete: gorp.Weak,",
					"return []rack.Key{e.Spare}",
				).
				ToNotContain("Detach:", "Tag:")
		})

		It("Should clear a weak reference that can hold none in its Detach", func() {
			resp := MustGenerate(device(`
			spare rack.Key? @ref on_delete weak @ref kind use
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					"OnDelete: gorp.Weak,",
					"Detach: func(e *Device, _ rack.Key) { e.Spare = nil },",
				).
				ToBeValidGoSource()
		})

		It("Should write the direction of a reference into its tag", func() {
			resp := MustGenerate(device(`
			spare rack.Key {
				@ref on_delete restrict
				@ref kind      use
				@ref direction read
			}
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(`Name:     "spare",`, "Tag:      1, // direction read").
				ToBeValidGoSource()
		})

		It(
			"Should give the direction of a top-level field to the references below it",
			func() {
				resp := MustGenerate(device(`
			inner Inner[] @ref direction write
		`)+`
			Inner struct {
				rack rack.Key @ref on_delete weak @ref kind use
			}
		`, "device", loader, p)
				ExpectContent(resp, "retrieve.gen.go").
					ToContain(
						`Name:     "inner.rack",`,
						"OnDelete: gorp.Weak,",
						"Tag:      2, // direction write",
					).
					ToNotContain("func MatchInners(").
					ToBeValidGoSource()
			},
		)
	})

	Describe("Retarget on a retrieved type", func() {
		DescribeTable(
			"Should move a scalar key in place whatever its on_delete",
			func(field string) {
				resp := MustGenerate(device(field), "device", loader, p)
				ExpectContent(resp, "retrieve.gen.go").
					ToContain(
						"Retarget: func(e *Device, to map[rack.Key]rack.Key) {",
						"if t, ok := to[e.Spare]; ok {",
						"e.Spare = t",
					).
					ToBeValidGoSource()
			},
			Entry("restrict", `spare rack.Key @ref on_delete restrict @ref kind use`),
			Entry("cascade", `spare rack.Key @ref on_delete cascade @ref kind place`),
			Entry("weak", `spare rack.Key @ref on_delete weak @ref kind use`),
			Entry(
				"a zero default",
				`spare rack.Key = 0 @ref on_delete detach @ref kind use`,
			),
		)

		It("Should move each key of an array in place", func() {
			resp := MustGenerate(device(`
			labels label.Key[] @ref on_delete detach @ref kind use
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					"Retarget: func(e *Device, to map[label.Key]label.Key) {",
					"for i, k := range e.Labels {",
					"if t, ok := to[k]; ok {",
					"e.Labels[i] = t",
				).
				ToBeValidGoSource()
		})

		It("Should give a moved optional key a new pointer", func() {
			resp := MustGenerate(device(`
			shelf rack.Key? @ref on_delete detach @ref kind use
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain(
					"if e.Shelf != nil {",
					"if t, ok := to[*e.Shelf]; ok {",
					"e.Shelf = &t",
				).
				ToBeValidGoSource()
		})

		It("Should not retarget a reference that holds resource IDs", func() {
			resp := MustGenerate(device(`
			owner resource.ID {
				@ref on_delete restrict
				@ref kind      use
				@ref types     rack
			}
		`), "device", loader, p)
			ExpectContent(resp, "retrieve.gen.go").
				ToContain("gorp.NewTargetReference(").
				ToNotContain("Retarget:")
		})
	})
})
