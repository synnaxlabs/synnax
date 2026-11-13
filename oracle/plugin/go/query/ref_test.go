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
		func(ctx SpecContext, field, kind string) {
			resp := MustGenerate(ctx, device(field), "device", loader, p)
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
		imports := ""
		for dep := range strings.SplitSeq(schema, ",") {
			imports += "import \"schemas/" + dep + "\"\n"
		}
		return `
			` + imports + `

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
		func(ctx SpecContext, schema, field, name, targets, match string) {
			resp := MustGenerate(ctx, union(schema, field), "device", loader, p)
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

	It("Should declare a resource target for an ID reference on a union", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, union(
			"resource",
			`owner resource.ID? @ref on_delete cascade @ref types device @ref kind use`,
		), "device", loader, p)
		ExpectContent(resp, "retrieve.gen.go").ToContain(
			"func resourceTarget(id resource.ID) gorp.Target {",
			"targets[i] = resourceTarget(v)",
		)
	})

	It("Should name the index of a union reference after the field", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, union(
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

	It("Should refuse a detach reference on a union", func(ctx SpecContext) {
		req := MustGenerateRequest(
			ctx,
			union("label", `labels label.Key[] @ref on_delete detach @ref kind use`),
			"device",
			loader,
		)
		Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
			"device.Device: @ref on_delete detach is not supported on the base " +
				"fields of a union (labels)",
		)))
	})

	It("Should register an optional reference to its own type", func(ctx SpecContext) {
		resp := MustGenerate(ctx, device(`
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
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, union("rack", `
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
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, union("label", `
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
			func(ctx SpecContext) {
				req := MustGenerateRequest(ctx, `
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

	It("Should detach a deleted target from an array", func(ctx SpecContext) {
		resp := MustGenerate(ctx, device(`
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

	It("Should detach an optional field and write it", func(ctx SpecContext) {
		resp := MustGenerate(ctx, device(`
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
		func(ctx SpecContext, field, zero string) {
			resp := MustGenerate(ctx, device(field), "device", loader, p)
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

	It("Should always hold the target of a required field", func(ctx SpecContext) {
		resp := MustGenerate(ctx, device(`
			rack rack.Key @ref on_delete cascade @ref kind use
		`), "device", loader, p)
		ExpectContent(resp, "retrieve.gen.go").
			ToNotContain("func MatchHasRack", "func setRack", "if e.Rack ==")
	})

	It(
		"Should register a reference to resource IDs of named types",
		func(ctx SpecContext) {
			resp := MustGenerate(ctx, device(`
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
		func(ctx SpecContext, fields, message string) {
			req := MustGenerateRequest(ctx, device(fields), "device", loader)
			Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(message)))
		},
		Entry("without on_delete", `rack rack.Key @ref`,
			"@ref field rack needs on_delete cascade, restrict, or detach"),
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
		func(ctx SpecContext) {
			req := MustGenerateRequest(ctx, `
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
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, status(`
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

		It("Should accept a primitive key type", func(ctx SpecContext) {
			resp := MustGenerate(ctx, `
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
			for _, ns := range []string{"rack", "label", "channel", "kv", "resource"} {
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

		generate := func(ctx SpecContext, types, fields string) *ContentExpectation {
			GinkgoHelper()
			resp := MustGenerate(ctx, config(types, fields), "modbus", loader, p)
			return ExpectContent(resp, "retrieve.gen.go")
		}

		It("Should skip a zero key and reset it on detach", func(ctx SpecContext) {
			generate(ctx, "", `
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

		It("Should declare the kind of a nested reference", func(ctx SpecContext) {
			generate(ctx, `
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

		It("Should reset a zero string key", func(ctx SpecContext) {
			generate(ctx, "", `
				entry kv.Key = "" @ref on_delete restrict @ref kind use
			`).
				ToContain(`if e.Entry != "" {`, `e.Entry = ""`)
		})

		It("Should reset a zero UUID key to its composite zero", func(ctx SpecContext) {
			generate(ctx, "", `
				tag label.Key = "" @ref on_delete restrict @ref kind use
			`).
				ToContain("if e.Tag != (label.Key{}) {", "e.Tag = label.Key{}")
		})

		It("Should reach a key through a nested struct", func(ctx SpecContext) {
			generate(ctx, `
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

		It("Should reach a key through an optional struct", func(ctx SpecContext) {
			generate(ctx, `
				Inner struct {
					channel channel.Key = 0 @ref on_delete restrict @ref kind use
				}
			`, `
				inner Inner?
			`).
				ToContain("if e.Inner != nil {", "if e.Inner.Channel != 0 {")
		})

		It("Should range over an array of structs", func(ctx SpecContext) {
			generate(ctx, `
				Entry struct {
					channel channel.Key = 0 @ref on_delete restrict @ref kind use
				}
			`, `
				entries Entry[]
			`).
				ToContain(
					`Name:     "entries.channel",`,
					"for _, v1 := range e.Entries {",
					"if v1.Channel != 0 {",
					"for i2 := range e.Entries {",
					`if e.Entries[i2].Channel == t {`,
					"e.Entries[i2].Channel = 0",
				)
		})

		It(
			"Should switch over every variant of a union that holds the key",
			func(ctx SpecContext) {
				generate(ctx, `
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
			func(ctx SpecContext) {
				generate(ctx, `
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
			func(ctx SpecContext) {
				generate(ctx, `
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

		It("Should detach a key from a nested array of keys", func(ctx SpecContext) {
			generate(ctx, `
				Axes struct {
					y1 channel.Key[] @ref on_delete detach @ref kind use
				}
			`, `
				axes Axes
			`).
				ToContain(
					"for _, k1 := range e.Axes.Y1 {",
					"e.Axes.Y1 = slices.DeleteFunc(e.Axes.Y1, func(k2 channel.Key) bool {",
					`"github.com/synnaxlabs/synnax/core/pkg/service/channel"`,
				)
		})

		It("Should clear a nested optional key", func(ctx SpecContext) {
			generate(ctx, `
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
			func(ctx SpecContext) {
				generate(ctx, "", `
				rack rack.Key @ref on_delete restrict @ref kind use
			`).
					ToContain(`targets = append(targets, e.Rack)`).
					ToNotContain("Detach:")
			},
		)

		It(
			"Should skip a recursive type that holds no reference",
			func(ctx SpecContext) {
				generate(ctx, `
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
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, `
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
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, config(`
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
			func(ctx SpecContext, types, fields, message string) {
				req := MustGenerateRequest(ctx, config(types, fields), "modbus", loader)
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
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, device(`
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
			func(ctx SpecContext) {
				resp := MustGenerate(ctx, device(`
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
	})
})
