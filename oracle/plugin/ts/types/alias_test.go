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
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

const aliasBase = `
@ts output "client/ts/src/base"

Labeled struct {
	label  string
	hidden bool
}

Box struct<T> {
	item T
}

Valve struct extends Labeled {
	open bool
}

Credentials struct {
	username string
}

Size enum {
	small = "small"
	large = "large"
}

Sized struct {
	size Size
}

Edge union on variant {
	pipe Labeled
}
`

const aliasHead = `
import "schemas/base"

@ts output "client/ts/src/test"

Labeled = base.Labeled
Box<T> = base.Box<T>
Valve = base.Valve
Credentials = base.Credentials
Size = base.Size
Sized = base.Sized
Edge = base.Edge
`

const aliasStatus = `
@ts output "client/ts/src/status"

Status struct<Details?> {
	key     string = create { @key }
	details Details?
	@ts concrete_types
	@create
}
`

var _ = Describe("TS aliases", func() {
	var (
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader().
			Add("schemas/base", aliasBase).
			Add("schemas/status", aliasStatus)
		typesPlugin = types.New(types.DefaultOptions())
	})

	generate := func(ctx SpecContext, use string) *ContentExpectation {
		GinkgoHelper()
		resp := MustGenerate(ctx, aliasHead+use, "test", loader, typesPlugin)
		return ExpectContent(resp, "test/types.gen.ts")
	}

	It("Should extend the schema of an aliased parent", func(ctx SpecContext) {
		generate(ctx, `
			Panel struct extends Labeled {
				width float64
			}
		`).ToContain(
			"export const panelZ = labeledZ\n  .extend({\n    width: z.number(),",
		)
	})

	It("Should extend the schema of a generic aliased parent", func(ctx SpecContext) {
		generate(ctx, `
			Crate struct<T> extends Box<T> {
				count int32
			}
		`).ToContain(
			"export const crateZ = <T extends z.ZodType>(t: T) =>\n" +
				"  boxZ(t)\n    .extend({",
		)
	})

	It("Should extend a variant from an aliased payload", func(ctx SpecContext) {
		generate(ctx, `
			Node union on variant {
				valve Valve
			}
		`).ToContain("export const valveNodeZ = valveZ.extend({")
	})

	It("Should extend a variant from an aliased union base", func(ctx SpecContext) {
		generate(ctx, `
			Shape union on kind extends Labeled {
				circle Credentials
			}
		`).ToContain(
			"export const circleShapeZ = labeledZ.extend(credentialsZ.shape).extend({",
		)
	})

	It(
		"Should omit a field a variant drops from an aliased base",
		func(ctx SpecContext) {
			generate(ctx, `
			Node union on variant {
				tank extends Labeled {
					-hidden
					depth float64
				}
			}
		`).ToContain(
				"export const tankNodeZ = labeledZ.omit({ hidden: true }).extend({",
			)
		},
	)

	It("Should keep an aliased optional union nullable", func(ctx SpecContext) {
		generate(ctx, `
			Panel struct {
				edge Edge?
			}
		`).ToContain("edge: zod.nullToUndefined(edgeZ),")
	})

	It("Should type the fields of an aliased struct default", func(ctx SpecContext) {
		generate(ctx, `
			Panel struct {
				sized Sized = { size = large }
			}
		`).ToContain(`sized: sizedZ.prefault({ size: "large" }),`)
	})

	It("Should constrain a type parameter by an aliased enum", func(ctx SpecContext) {
		generate(ctx, `
			Tray struct<T extends Size> {
				item T
			}
		`).ToContain("T extends z.ZodType<Size>")
	})

	DescribeTable("Should project a create field typed through an alias to its New",
		func(ctx SpecContext, decls, expected string) {
			resp := MustGenerate(ctx, `
			import "schemas/status"

			@ts output "client/ts/src/device"

			DeviceStatusDetails struct {
				rack int64
			}
			`+decls+`
			Device struct<Properties extends record = record> {
				key        string = create { @key }
				properties Properties
				status     Health?
				@ts concrete_types
				@create
			}
			`, "device", loader, typesPlugin)
			ExpectContent(resp, "device/types.gen.ts").ToContain(expected)
		},
		Entry("an alias chain", `
			DeviceStatus = status.Status<DeviceStatusDetails>
			Health = DeviceStatus
		`, "status?: status.New<typeof deviceStatusDetailsZ>;"),
		Entry("an aliased details type", `
			Info = DeviceStatusDetails
			Health = status.Status<Info>
		`, "status?: status.New<typeof infoZ>;"),
	)
})
