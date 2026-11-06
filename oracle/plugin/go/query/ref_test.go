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
			rack rack.Key @ref on_delete restrict
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
				`Table:    "device",`,
				"OnDelete: gorp.Cascade,",
				"Acyclic:  true,",
				"if e.Parent == nil {",
				"return []Key{*e.Parent}",
			)
	})

	It("Should detach a deleted target from an array", func(ctx SpecContext) {
		resp := MustGenerate(ctx, device(`
			labels label.Key[] @ref on_delete detach
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
			shelf rack.Key? @ref on_delete detach
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
		Entry("an integer key", `spare rack.Key = 0 @ref on_delete detach`, "0"),
		Entry("a string key", `spare Key = "" @ref on_delete detach`, `""`),
		Entry(
			"a UUID key",
			`spare label.Key = "" @ref on_delete detach`,
			"(label.Key{})",
		),
	)

	It("Should always hold the target of a required field", func(ctx SpecContext) {
		resp := MustGenerate(ctx, device(`
			rack rack.Key @ref on_delete cascade
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
		Entry(
			"detaching a required field",
			`rack rack.Key @ref on_delete detach`,
			"@ref field rack needs to be optional, an array, or zero by default to detach",
		),
		Entry(
			"detaching a field with a default that is not zero",
			`rack rack.Key = 4 @ref on_delete detach`,
			"@ref field rack needs to be optional, an array, or zero by default to detach",
		),
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
					tag  label.Key @ref on_delete cascade
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
				rack rack.Key = 0 @ref on_delete restrict
			`).
				ToContain(
					"func ReadConfigReferences() []gorp.Reference[uuid.UUID, ReadConfig]",
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

		It("Should reset a zero string key", func(ctx SpecContext) {
			generate(ctx, "", `
				entry kv.Key = "" @ref on_delete restrict
			`).
				ToContain(`if e.Entry != "" {`, `e.Entry = ""`)
		})

		It("Should reset a zero UUID key to its composite zero", func(ctx SpecContext) {
			generate(ctx, "", `
				tag label.Key = "" @ref on_delete restrict
			`).
				ToContain("if e.Tag != (label.Key{}) {", "e.Tag = label.Key{}")
		})

		It("Should reach a key through a nested struct", func(ctx SpecContext) {
			generate(ctx, `
				Inner struct {
					channel channel.Key = 0 @ref on_delete restrict
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
					channel channel.Key = 0 @ref on_delete restrict
				}
			`, `
				inner Inner?
			`).
				ToContain("if e.Inner != nil {", "if e.Inner.Channel != 0 {")
		})

		It("Should range over an array of structs", func(ctx SpecContext) {
			generate(ctx, `
				Entry struct {
					channel channel.Key = 0 @ref on_delete restrict
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
					channel channel.Key = 0 @ref on_delete restrict
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
						channel channel.Key = 0 @ref on_delete restrict
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
					tag label.Key @ref on_delete detach
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
					y1 channel.Key[] @ref on_delete detach
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
							types     rack
						}
					}
				}
			`, `
				inner Inner
				rack  rack.Key = 0 @ref on_delete restrict
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
				rack rack.Key @ref on_delete restrict
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
				rack rack.Key = 0 @ref on_delete restrict
			`).
					ToContain(`Name:     "rack",`).
					ToNotContain("tree.")
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
					rack rack.Key = 0 @ref on_delete restrict
					@retrieve references
				}
			`, `
				rack rack.Key = 0 @ref on_delete restrict
			`), "modbus", loader, p)
				ExpectContent(resp, "retrieve.gen.go").ToContain(
					"func ReadConfigReferences()",
					"func WriteConfigReferences()",
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
					rack     rack.Key = 0 @ref on_delete restrict
					children Node[]
				}`,
				`tree Node`,
				"@ref fields inside the recursive type Node are not supported",
			),
			Entry(
				"detaching a required key outside an array",
				`Inner struct {
					rack rack.Key @ref on_delete detach
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
						acyclic
					}
				}`,
				"@ref field rack can be acyclic only when it references its own type",
			),
			Entry("a type with no reference", "", `name string`,
				"@retrieve references needs a @ref field"),
		)

		It(
			"Should reject a nested reference under a type that generates a Retrieve",
			func(ctx SpecContext) {
				req := MustGenerateRequest(ctx, device(`
				inner Inner
			`)+`
				Inner struct {
					rack rack.Key = 0 @ref on_delete restrict
				}
			`, "device", loader)
				Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
					"device.Inner: @ref field rack needs a @retrieve type to store it",
				)))
			},
		)
	})
})
