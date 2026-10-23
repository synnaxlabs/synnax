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
`

const aliasHead = `
import "schemas/base"

@ts output "client/ts/src/test"

Labeled = base.Labeled
Box<T> = base.Box<T>
Valve = base.Valve
Credentials = base.Credentials
`

var _ = Describe("TS aliases", func() {
	var (
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader().Add("schemas/base", aliasBase)
		typesPlugin = types.New(types.DefaultOptions())
	})

	generate := func(ctx SpecContext, use string) *ContentExpectation {
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
})
