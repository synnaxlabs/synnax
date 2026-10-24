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
	"github.com/synnaxlabs/oracle/plugin/py/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

const aliasBase = `
@py output "client/py/synnax/base"

Labeled struct {
	label  string
	hidden bool
}

Box struct<T> {
	item   T
	hidden bool
}

Code uint32
`

const aliasHead = `
import "schemas/base"

@py output "client/py/synnax/test"

Labeled = base.Labeled
Box<T> = base.Box<T>
TextBox = base.Box<string>
Code = base.Code
`

var _ = Describe("Python aliases", func() {
	DescribeTable("Should generate a use of an alias as a use of its target",
		func(ctx SpecContext, use, expected string) {
			loader := NewMockFileLoader().Add("schemas/base", aliasBase)
			resp := MustGenerate(
				ctx,
				aliasHead+use,
				"test",
				loader,
				types.New(types.DefaultOptions()),
			)
			ExpectContent(resp, "test/types_gen.py").ToContain(expected)
		},
		Entry("a distinct default", `
			Panel struct {
				limit Code = 5
			}
		`, "limit: Code = Field(default=Code(5)"),
		Entry("a parent whose field the child redeclares", `
			Panel struct extends Labeled {
				hidden bool?
				width  float64
			}
		`, "class Panel(BaseModel):"),
		Entry("a generic parent's type argument", `
			Panel struct extends Box<string> {
				count int32
			}
		`, "class Panel(Box[str]):"),
		Entry("a generic alias's type variable", `
			Panel struct {
				count int32
			}
		`, "T = TypeVar(\"T\")\n"),
		Entry("a generic alias's declaration", `
			Panel struct {
				count int32
			}
		`, "Box: TypeAlias = base.Box[T]\n"),
		Entry("a generic alias's type arguments", `
			Panel struct {
				box Box<string>
			}
		`, "box: Box[str]\n"),
		Entry("a generic parent inlined for a redeclared field", `
			Panel struct extends Box<string> {
				hidden bool?
			}
		`, "class Panel(BaseModel):\n    item: str\n"),
		Entry("an instance of a generic parent inlined for a redeclared field", `
			Panel struct extends TextBox {
				hidden bool?
			}
		`, "class Panel(BaseModel):\n    item: str\n"),
	)
})
