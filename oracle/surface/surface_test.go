// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package surface_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/surface"
	"github.com/synnaxlabs/oracle/testutil"
)

func report(ctx context.Context, source string) string {
	GinkgoHelper()
	table, diag := testutil.Analyze(
		ctx, source, "widget", testutil.NewMockFileLoader(),
	)
	Expect(diag.Ok()).To(BeTrue(), diag.String())
	var b strings.Builder
	Expect(surface.Write(&b, table)).To(Succeed())
	return b.String()
}

var _ = Describe("Write", func() {
	It("Should write structs, fields, enums, and unions", func(ctx SpecContext) {
		Expect(report(ctx, `
Size enum {
	small = "small"
	large = "large"
}

Labeled struct {
	label string = "valve"
	tags string[]?
}

Valve struct extends Labeled {
	size Size
}

Part union on kind {
	valve Valve
}
`)).To(Equal(`widget.Labeled struct
widget.Labeled.label string = "valve"
widget.Labeled.tags? Array<string>
widget.Part union on kind
widget.Part|valve widget.Valve
widget.Size enum
widget.Size.small = "small"
widget.Size.large = "large"
widget.Valve struct extends widget.Labeled
widget.Valve.size widget.Size
`))
	})

	It(
		"Should write aliases, distinct types, actions, and defaults",
		func(ctx SpecContext) {
			Expect(report(ctx, `
Key uint32

ID = Key

Level enum {
	low = 1
}

Valve struct {
	key Key
	tags string[] = []
	ratio float64 = 0.0
	open bool = false

	action Rename {
		name string
	}
}
`)).To(Equal(`widget.ID = widget.Key
widget.Key distinct uint32
widget.Level enum
widget.Level.low = 1
widget.Valve struct
widget.Valve.key widget.Key
widget.Valve.tags Array<string> = []
widget.Valve.ratio float64 = 0
widget.Valve.open bool = false
widget.Valve action Rename
widget.Valve!Rename.name string
`))
		},
	)

	It("Should write omitted fields and borrowed enums", func(ctx SpecContext) {
		Expect(report(ctx, `
Method enum {
	password = "password"
}

Parent struct {
	name string
	age int32
}

Child struct extends Parent {
	-age
}

Proof union on method Method {
	password Parent
}
`)).To(Equal(`widget.Child struct extends widget.Parent
widget.Child omits age
widget.Method enum
widget.Method.password = "password"
widget.Parent struct
widget.Parent.name string
widget.Parent.age int32
widget.Proof union on method widget.Method
widget.Proof|password widget.Parent
`))
	})

	It("Should write type parameters and struct defaults", func(ctx SpecContext) {
		Expect(report(ctx, `
Kind enum {
	a = "a"
}

Point struct {
	x int32
	y int32
}

Box struct<T?, K extends Kind = Kind> {
	item T
	kind K
	at Point = {x = 0, y = 0}
}

Boxes = Box<string>[]
`)).To(Equal(`widget.Box<T?, K extends widget.Kind = widget.Kind> struct
widget.Box.item T
widget.Box.kind K
widget.Box.at widget.Point = {x = 0, y = 0}
widget.Boxes = Array<widget.Box<string>>
widget.Kind enum
widget.Kind.a = "a"
widget.Point struct
widget.Point.x int32
widget.Point.y int32
`))
	})

	It("Should ignore docs and formatting", func(ctx SpecContext) {
		Expect(report(ctx, `
// A thing that holds a name.
Valve struct {
	// The name of the valve.
	name   string
}
`)).To(Equal(report(ctx, "Valve struct { name string }")))
	})

	It("Should change when a field changes", func(ctx SpecContext) {
		Expect(report(ctx, "Valve struct { name string }")).
			ToNot(Equal(report(ctx, "Valve struct { name string? }")))
	})
})
