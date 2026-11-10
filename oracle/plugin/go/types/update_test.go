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
	"github.com/synnaxlabs/oracle/plugin/go/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Update types", func() {
	It("Should decode an update without the type as base fields only", func(
		ctx SpecContext,
	) {
		loader := NewMockFileLoader()
		loader.Add("schemas/x/patch", `
			@go output "x/go/patch"

			Set struct { path string[] }
			Remove struct { path string[] }
			Insert struct { path string[] }
			Move struct { path string[] }
			Change struct<T> {
				value T?
				@presence
			}
		`)
		resp := MustGenerate(ctx, `
			import "schemas/x/patch"

			@go output "core/pkg/service/view"

			Base struct {
				key  uuid
				name string { @patch }
			}

			View union on type extends Base {
				range {
					where string? { @patch }
				}
				status {
					level int32 { @patch }
				}
				@actions
			}
		`, "view", loader, types.New(types.DefaultOptions()))
		ExpectContent(resp, "core/pkg/service/view/types.gen.go").ToContain(
			"func (BaseUpdate) isUpdateVariant() {}",
			"\tcase BaseUpdate:\n\t\treturn json.MarshalEncode(enc, v)\n",
			"\tcase \"\":\n\t\tvar v BaseUpdate\n\t\tif err := json.Unmarshal(\n"+
				"\t\t\tdata, &v, opts, json.RejectUnknownMembers(true),\n",
		)
	})

	Describe("Apply", func() {
		var loader *MockFileLoader
		BeforeEach(func() {
			loader = NewMockFileLoader()
			loader.Add("schemas/x/patch", `
				@go output "x/go/patch"

				Set struct { path string[] }
				Remove struct { path string[] }
				Insert struct { path string[] }
				Move struct { path string[] }
				Change struct<T> {
					value T?
					@presence
				}
				Struct struct<U, T> { merge U? }
				List struct<E, K, U> { set E[]? }
				ValueList struct<E> { set E[]? }
				Update struct<V> { value V }
			`)
		})
		generate := func(ctx SpecContext, source string) *ContentExpectation {
			GinkgoHelper()
			resp := MustGenerate(ctx, `
				import "schemas/x/patch"

				@go output "core/pkg/service/plot"
			`+source, "plot", loader, types.New(types.DefaultOptions()))
			return ExpectContent(resp, "core/pkg/service/plot/types.gen.go").
				ToBeValidGoSource()
		}

		It("Should declare the task type of a config beside its update", func(
			ctx SpecContext,
		) {
			generate(ctx, `
				AnalogReadConfig struct {
					key  uuid
					rate float64

					@patch
					@task type "plot_analog_read"
				}
			`).ToContain(
				"\t// AnalogReadTaskType is the task type of AnalogReadConfig.\n"+
					"\tAnalogReadTaskType = \"plot_analog_read\"\n",
			)
		})

		It("Should apply each field kind of a struct update", func(ctx SpecContext) {
			content := generate(ctx, `
				Point struct {
					x float64
					y float64
				}

				Entry struct {
					key  string { @key }
					port int32
				}

				Region struct {
					tags string[]
				}

				Plot struct {
					key     uuid
					name    string   { @patch }
					note    string?  { @patch }
					origin  Point    { @patch }
					anchor  Point?   { @patch }
					entries Entry[]  { @patch }
					tags    string[] { @patch }
					regions Region[] { @patch }
					layout  record   { @patch }

					@actions
				}
			`)
			content.ToContain(
				"func (u Update) ApplyTo(x *Plot) error {\n" +
					"\tif u.Name != nil {\n\t\tx.Name = *u.Name\n\t}\n" +
					"\tu.Note.ApplyOptional(&x.Note)\n",
			)
			content.ToContain(
				"if err := u.Origin.ApplyTo(&x.Origin, " +
					"func(v *Point, w PointUpdate) error { return w.ApplyTo(v) }); " +
					"err != nil {",
			)
			content.ToContain("PointUpdate.Defaults,")
			content.ToContain(
				"func (u PointUpdate) Defaults() (Point, error) {",
			)
			content.ToContain(
				"func(e Entry) string { return e.Key },\n" +
					"\t\t\tfunc(w EntryUpdate) string { return w.Key },",
			)
			content.ToContain(
				"patch.ApplyValueList(&x.Tags, *u.Tags)",
			)
			content.ToContain(
				"return errors.Wrap(validate.ErrValidation, " +
					"\"regions takes only set\")",
			)
			content.ToContain(
				"\tif u.Layout != nil {\n\t\tx.Layout = u.Layout\n\t}\n",
			)
		})

		It("Should name each embed on the way to a field that an embed shadows", func(
			ctx SpecContext,
		) {
			content := generate(ctx, `
				Scale struct {
					scale float64
				}

				Channel struct extends Scale {
					name string
				}

				Plot struct {
					key     uuid
					channel Channel { @patch }
					@actions
				}
			`)
			content.ToContain("x.Scale.Scale = *u.Scale")
			content.ToContain("x.Name = *u.Name")
		})

		It("Should apply an update without the type through the shared base", func(
			ctx SpecContext,
		) {
			content := generate(ctx, `
				Base struct {
					key  uuid
					name string { @patch }
				}

				View union on type extends Base {
					range {
						where string? { @patch }
					}
					status {
						level int32 { @patch }
					}
					@actions
				}
			`)
			content.ToContain(
				"\tcase BaseUpdate:\n" +
					"\t\tif x.Variant == nil {\n" +
					"\t\t\treturn errors.Wrap(validate.ErrValidation, " +
					"\"type is required\")\n\t\t}\n" +
					"\t\tbase := x.Base()\n",
			)
			content.ToContain(
				"\tcase RangeUpdate:\n" +
					"\t\tnext, same := x.Variant.(RangeView)\n" +
					"\t\tif !same {\n" +
					"\t\t\tvar err error\n" +
					"\t\t\tif next, err = v.Defaults(); err != nil {\n" +
					"\t\t\t\treturn err\n\t\t\t}\n" +
					"\t\t\tif x.Variant != nil {\n" +
					"\t\t\t\tnext.Base = x.Base()\n",
			)
		})

		It("Should declare base helpers when a variant drops a base field", func(
			ctx SpecContext,
		) {
			content := generate(ctx, `
				Base struct {
					key  string { @key }
					name string
					port int32
				}

				Channel union on type extends Base {
					voltage {}
					builtin {
						-port
					}
				}

				Config struct {
					key      uuid
					channels Channel[] { @patch }
					@actions
				}
			`)
			content.ToContain(
				"func(e Channel) string { return baseOfChannel(&e).Key },",
			)
			content.ToContain(
				"func baseOfChannel(x *Channel) Base {",
			)
			content.ToContain(
				"\tcase BuiltinChannel:\n\t\tbase.Key = v.Key\n" +
					"\t\tbase.Name = v.Name\n\t}\n\treturn base\n}",
			)
			content.ToContain(
				"func setBaseOfChannel(x *Channel, base Base) {",
			)
			content.ToContain(
				"func (u ChannelUpdate) Key() string {",
			)
		})

		It("Should carry only the fields every variant holds to a new variant", func(
			ctx SpecContext,
		) {
			generate(ctx, `
				Base struct {
					key  string { @key }
					name string
					port int32
				}

				Channel union on type extends Base {
					voltage {}
					builtin {
						-port
					}
				}

				Config struct {
					key      uuid
					channels Channel[] { @patch }
					@actions
				}
			`).ToContain(
				"\t\t\t\tbase := baseOfChannel(x)\n"+
					"\t\t\t\tnext.Key = base.Key\n"+
					"\t\t\t\tnext.Name = base.Name\n\t\t\t}\n",
				`errors.Wrap(validate.ErrValidation, "port is required")`,
			).ToNotContain("next.Port = base.Port")
		})

		It("Should check each field with no default when the variant changes", func(
			ctx SpecContext,
		) {
			content := generate(ctx, `
				Base struct {
					key uuid
				}

				Reading union on type extends Base {
					coil {}
					text {
						length uint16 { @patch }
					}
					@actions
				}
			`)
			content.ToContain(
				"func (u TextUpdate) Defaults() (TextReading, error) {\n" +
					"\tif u.Length == nil {\n" +
					"\t\treturn TextReading{}, " +
					"errors.Wrap(validate.ErrValidation, \"length is required\")\n",
			)
		})
	})
})
