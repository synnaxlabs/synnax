// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
)

const updatePatchSchema = `
@cpp output "x/cpp/patch"
@go output  "x/go/patch"

Set struct { path string[] }
Remove struct { path string[] }
Insert struct { path string[] }
Move struct { path string[] }
Change struct<T> {
    value T?
    @presence
}
Struct struct<U, T> {
    merge U?
    replace T?
}
List struct<E, K, U> {
    set E[]?
    remove K[]?
    update U[]?
}
ValueList struct<E> {
    set E[]?
}
`

func ref(name string, args ...resolution.TypeRef) resolution.TypeRef {
	return resolution.TypeRef{Name: name, TypeArgs: args}
}

func fieldOf(table *resolution.Table, typ, name string) resolution.Field {
	GinkgoHelper()
	form := table.MustGet(typ).Form.(resolution.StructForm)
	return findField(form.Fields, name)
}

var _ = Describe("Update types", func() {
	var loader *MockFileLoader

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/patch", updatePatchSchema)
	})

	analyze := func(ctx SpecContext, source string) *resolution.Table {
		GinkgoHelper()
		table, diag := Analyze(ctx, source, "test", loader)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
		return table
	}

	Describe("Record updates", func() {
		const record = `
			import "schemas/x/patch"

			@go output "core/pkg/service/test"

			Point struct {
				x float64
				y float64
			}

			Entry struct {
				key  uuid { @key }
				port int32
			}

			Plot struct<Data extends record = record, Extra extends record = record> {
				key      uuid
				name     string            { @patch }
				note     string?           { @patch }
				origin   Point             { @patch }
				anchor   Point?            { @patch }
				entries  Entry[]           { @patch }
				tags     string[]          { @patch }
				data     Data              { @patch }
				layout   record            { @patch }
				hash     string            { @patch core }
				internal string
				extra    Extra

				@actions
			}
		`

		It("Should derive Update from the @patch fields of the record", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, record)
			update := table.MustGet("test.Update")
			Expect(update.Updates).To(Equal("test.Plot"))
			Expect(doc.Get(update.Domains)).To(Equal(
				"holds changes to the fields of the plot. An absent field does not change.",
			))
			form := update.Form.(resolution.StructForm)
			Expect(fieldNames(form)).To(Equal([]string{
				"name", "note", "origin", "anchor", "entries", "tags", "data", "layout",
				"hash",
			}))
			for _, f := range form.Fields {
				Expect(f.Optional).To(BeTrue(), f.Name)
			}
		})

		DescribeTable("Should map each field to the type that changes it",
			func(ctx SpecContext, name string, want resolution.TypeRef) {
				table := analyze(ctx, record)
				got := fieldOf(table, "test.Update", name).Type
				Expect(got.Name).To(Equal(want.Name))
				Expect(got.TypeArgs).To(HaveLen(len(want.TypeArgs)))
				for i, arg := range want.TypeArgs {
					Expect(got.TypeArgs[i].Name).To(Equal(arg.Name))
					for j, inner := range arg.TypeArgs {
						Expect(got.TypeArgs[i].TypeArgs[j].Name).To(Equal(inner.Name))
					}
				}
			},
			Entry("a required scalar to itself", "name", ref("string")),
			Entry("an optional scalar to a change", "note",
				ref("patch.Change", ref("string"))),
			Entry("a required struct to a merge", "origin",
				ref("patch.Struct", ref("test.PointUpdate"), ref("test.Point"))),
			Entry("an optional struct to a change of a merge", "anchor",
				ref("patch.Change", ref("patch.Struct",
					ref("test.PointUpdate"), ref("test.Point")))),
			Entry("a keyed list to list operators", "entries",
				ref("patch.List", ref("test.Entry"), ref("uuid"),
					ref("test.EntryUpdate"))),
			Entry("a list of values to value list operators", "tags",
				ref("patch.ValueList", ref("string"))),
			Entry("a record to itself", "layout", ref("record")),
		)

		It("Should keep only the type parameters that the update holds", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, record)
			form := table.MustGet("test.Update").Form.(resolution.StructForm)
			Expect(form.TypeParams).To(HaveLen(1))
			Expect(form.TypeParams[0].Name).To(Equal("Data"))
			data := findField(form.Fields, "data")
			Expect(data.Type.TypeParam).To(BeIdenticalTo(&form.TypeParams[0]))
		})

		It("Should keep the key of a list entry required", func(ctx SpecContext) {
			table := analyze(ctx, record)
			key := fieldOf(table, "test.EntryUpdate", "key")
			Expect(key.Optional).To(BeFalse())
			Expect(key.Type.Name).To(Equal("uuid"))
			Expect(fieldOf(table, "test.EntryUpdate", "port").Optional).To(BeTrue())
		})

		It("Should leave a Core-only field out of Python and C++", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, record)
			hash := fieldOf(table, "test.Update", "hash")
			Expect(hash.Domains).To(HaveKey("py"))
			Expect(hash.Domains).To(HaveKey("cpp"))
			Expect(hash.Domains).ToNot(HaveKey("ts"))
		})

		It("Should derive UpdateInput for TypeScript without Core-only fields", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, record)
			input := table.MustGet("test.UpdateInput")
			form := input.Form.(resolution.StructForm)
			Expect(form.Extends).To(HaveLen(1))
			Expect(form.Extends[0].Name).To(Equal("test.Update"))
			Expect(form.OmittedFields).To(Equal([]string{"hash"}))
			for _, lang := range []string{"go", "py", "cpp", "pb"} {
				Expect(omit.IsAbsent(input, lang)).To(BeTrue(), lang)
			}
			Expect(domainExprNames(input.Domains["ts"])).To(
				ContainElements("use_input", "type_only"),
			)
		})

		It("Should derive no UpdateInput for a record without Core-only fields", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, `
				import "schemas/x/patch"
				Plot struct {
					key  uuid
					name string { @patch }
					@actions
				}
			`)
			_, ok := table.Get("test.UpdateInput")
			Expect(ok).To(BeFalse())
		})

		It("Should render each update as a TypedDict in Python", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, record)
			for _, name := range []string{"test.Update", "test.PointUpdate"} {
				Expect(domainExprNames(table.MustGet(name).Domains["py"])).To(
					ContainElement("dict"), name,
				)
			}
		})

		It("Should derive no update for a record without @patch fields", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, `
				import "schemas/x/patch"
				Plot struct {
					key uuid
					@actions
				}
			`)
			_, ok := table.Get("test.Update")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Struct updates", func() {
		It("Should derive an update from every wire field of a type-level @patch", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, `
				import "schemas/x/patch"

				Config struct {
					key    uuid { @key }
					rate   float64
					secret string { @go wire omit }
					@patch
				}
			`)
			form := table.MustGet("test.ConfigUpdate").Form.(resolution.StructForm)
			Expect(fieldNames(form)).To(Equal([]string{"key", "rate"}))
			Expect(findField(form.Fields, "key").Optional).To(BeTrue())
		})

		It("Should replace a struct with a hand-written Go decoder whole", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, `
				import "schemas/x/patch"

				Color struct {
					r uint8
					@go json hand
				}

				Config struct {
					color Color
					@patch
				}
			`)
			Expect(fieldOf(table, "test.ConfigUpdate", "color").Type.Name).
				To(Equal("test.Color"))
			_, ok := table.Get("test.ColorUpdate")
			Expect(ok).To(BeFalse())
		})

		It("Should leave out a field whose type a hand-written owner lacks", func(
			ctx SpecContext,
		) {
			table := analyze(ctx, `
				import "schemas/x/patch"

				@go output  "core/pkg/service/test"
				@cpp output "client/cpp/test"

				Group struct {
					key uuid
					@cpp omit
				}

				Plot struct {
					key   uuid
					name  string   { @patch }
					tags  string[] { @patch }
					group Group?   { @patch }

					@actions
					@cpp hand
				}
			`)
			Expect(fieldOf(table, "test.Update", "group").Domains).To(HaveKey("cpp"))
			Expect(fieldOf(table, "test.Update", "name").Domains).ToNot(HaveKey("cpp"))
			Expect(fieldOf(table, "test.Update", "tags").Domains).ToNot(HaveKey("cpp"))
		})
	})

	DescribeTable("Should reject", func(ctx SpecContext, source, wantErr string) {
		_, diag := Analyze(ctx, source, "test", loader)
		Expect(diag).To(MatchError(ContainSubstring(wantErr)))
	},
		Entry("a field with the name of an operator", `
import "schemas/x/patch"
Plot struct {
    key uuid
    set string { @patch }
    @actions
}`, "5:4 error: Plot.set: an update reads set as an operator, so no field of "+
			"Plot can take the name"),
		Entry("an operator name in a struct that an update reaches", `
import "schemas/x/patch"
Point struct {
    move float64
}
Plot struct {
    key uuid
    origin Point { @patch }
    @actions
}`, "4:4 error: Point.move: an update reads move as an operator"),
		Entry("a field option other than core", `
import "schemas/x/patch"
Plot struct {
    key uuid
    name string { @patch loud }
    @actions
}`, "5:25 error: @patch on Plot.name takes only core, not loud"),
		Entry("a type-level option", `
import "schemas/x/patch"
Config struct {
    rate float64
    @patch loud
}`, "5:11 error: @patch on Config takes no options, not loud"),
		Entry("a type-level @patch without the patch import", `
Config struct {
    rate float64
    @patch
}`, `4:4 error: @patch requires import "schemas/x/patch"`),
		Entry("a @patch field without a JSON form", `
import "schemas/x/patch"
Plot struct {
    key uuid
    secret string {
        @patch
        @go wire omit
    }
    @actions
}`, "5:4 error: Plot.secret: a field without a JSON form takes no update"),
		Entry("an update name that the schema declares", `
import "schemas/x/patch"
ConfigUpdate struct {
    rate float64
}
Config struct {
    rate float64
    @patch
}`, "6:0 error: Config derives the update type test.ConfigUpdate, which the "+
			"schema already declares"),
		Entry("an option on @presence", `
Box struct<T> {
    value T?
    @presence loud
}`, "4:14 error: @presence on Box takes no options, not loud"),
	)
})
