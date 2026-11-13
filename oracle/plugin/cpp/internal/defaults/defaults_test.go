// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package defaults_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/defaults"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

const source = `
	@cpp output "client/cpp/task"

	Level int64

	Mode enum {
		off = 0
		on  = 1
	}

	Color enum {
		deep_red = "deep_red"
		blue     = "blue"
	}

	Point struct {
		x int32
		y int32 = 2
	}

	Task struct {
		name   string  = "task"
		count  int32   = 3
		ratio  float64 = 0.5
		active bool    = false
		level  Level   = 4
		mode   Mode    = off
		color  Color   = blue
		point  Point   = { y = 2, x = 1 }
		empty  Point   = {}
		tags   string[] = ["a", "b"]
		key    uuid     = create
	}
`

var _ = Describe("Renderer", func() {
	var (
		renderer defaults.Renderer
		task     resolution.Type
	)

	literal := func(name string) string {
		GinkgoHelper()
		form := task.Form.(resolution.StructForm)
		field := MustBeOk(lo.Find(
			form.Fields,
			func(f resolution.Field) bool { return f.Name == name },
		))
		return renderer.Literal(field.Type, *field.Default)
	}

	BeforeEach(func(ctx SpecContext) {
		loader := testutil.NewMockFileLoader()
		table := testutil.MustGenerateRequest(ctx, source, "task", loader).Resolutions
		task = MustBeOk(table.Lookup("task", "Task"))
		renderer = defaults.Renderer{
			Table:     table,
			Namespace: "task",
			TypeName: func(typeRef resolution.TypeRef) string {
				return typeRef.Name
			},
		}
	})

	Describe("Literal", func() {
		DescribeTable("Should render a default",
			func(field, expected string) {
				Expect(literal(field)).To(Equal(expected))
			},
			Entry("a string", "name", `"task"`),
			Entry("an integer", "count", "3"),
			Entry("a float", "ratio", "0.500000"),
			Entry("a boolean", "active", "false"),
			Entry("a distinct integer", "level", "task.Level(4)"),
			Entry("an int enum", "mode", "Mode::Off"),
			Entry("a string enum", "color", "COLOR_BLUE"),
			Entry("a struct in declaration order", "point", "{.x = 1, .y = 2}"),
			Entry("an empty struct", "empty", "{}"),
			Entry("an array", "tags", `{"a", "b"}`),
		)

		It("Should render nothing for an unsupported default", func() {
			Expect(literal("key")).To(BeEmpty())
		})
	})

	Describe("EnumVariant", func() {
		variant := func(enumName, member string) validation.EnumVariant {
			GinkgoHelper()
			typ := MustBeOk(renderer.Table.Lookup("task", enumName))
			values := typ.Form.(resolution.EnumForm).Values
			value := MustBeOk(lo.Find(
				values,
				func(v resolution.EnumValue) bool { return v.Name == member },
			))
			return validation.EnumVariant{Type: typ, Variant: value}
		}

		It("Should reference an int enum through its class", func() {
			Expect(renderer.EnumVariant(variant("Mode", "on"))).To(Equal("Mode::On"))
		})

		It("Should reference a string enum through its constant", func() {
			Expect(renderer.EnumVariant(variant("Color", "deep_red"))).
				To(Equal("COLOR_DEEP_RED"))
		})

		It("Should qualify a variant from another namespace", func() {
			renderer.Namespace = "other"
			Expect(renderer.EnumVariant(variant("Mode", "on"))).
				To(Equal("::synnax::task::Mode::On"))
		})

		It("Should include the header of a variant from another namespace", func() {
			var included []string
			renderer.Namespace = "other"
			renderer.Include = func(path string) { included = append(included, path) }
			renderer.EnumVariant(variant("Color", "deep_red"))
			Expect(included).To(Equal([]string{"client/cpp/task/types.gen.h"}))
		})

		It("Should not include a header for a variant in the same namespace", func() {
			var included []string
			renderer.Include = func(path string) { included = append(included, path) }
			renderer.EnumVariant(variant("Color", "deep_red"))
			Expect(included).To(BeEmpty())
		})
	})
})
