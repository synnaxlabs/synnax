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

const aliasTelem = `
@py output "x/py/x/telem"

TimeStamp int64 {
	@py hand
}
`

const aliasBase = `
import "schemas/x/telem"

@py output "client/py/synnax/base"

Stamp = telem.TimeStamp

Key = uuid

Labeled struct {
	label  string
	hidden bool
}

Box struct<T> {
	item   T
	hidden bool
}

Pair struct<A, B> {
	first  A
	second B
}

Code uint32
`

const aliasHead = `
import "schemas/base"

@py output "client/py/synnax/test"

Labeled = base.Labeled
Box<T> = base.Box<T>
TextBox = base.Box<string>
Swapped<A, B> = base.Pair<B, A>
Unswapped<A, B> = Swapped<B, A>
Loose<T? = string> = base.Box<T>
Code = base.Code
Stamp = base.Stamp
Key = base.Key
Moment Stamp
Ident Key
`

var _ = Describe("Python aliases", func() {
	DescribeTable("Should generate a use of an alias as a use of its target",
		func(ctx SpecContext, use string, expected ...string) {
			loader := NewMockFileLoader().
				Add("schemas/x/telem", aliasTelem).
				Add("schemas/base", aliasBase)
			resp := MustGenerate(
				ctx,
				aliasHead+use,
				"test",
				loader,
				types.New(types.DefaultOptions()),
			)
			ExpectContent(resp, "test/types_gen.py").ToContain(expected...)
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
		Entry("a now default on an alias of a timestamp", `
			Panel struct {
				created Stamp = now
			}
		`,
			"from x import telem\n",
			"created: Stamp = Field(default_factory=telem.TimeStamp.now",
		),
		Entry("a now default on a distinct type over an alias of a timestamp", `
			Panel struct {
				created Moment = now
			}
		`,
			"from x import telem\n",
			"created: Moment = Field(default_factory=telem.TimeStamp.now",
		),
		Entry("a zero default on an alias of a uuid", `
			Panel struct {
				key Key = 0
			}
		`, "key: Key = UUID(int=0)\n"),
		Entry("a zero default on a distinct type over an alias of a uuid", `
			Panel struct {
				ident Ident = 0
			}
		`, "ident: Ident = UUID(int=0)\n"),
		Entry("a distinct type over an alias", `
			Panel struct {
				created Moment
			}
		`, "Moment: TypeAlias = Stamp\n"),
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
		`, "box: base.Box[str]\n"),
		Entry("a generic alias that reorders its type parameters", `
			Panel struct {
				pair Swapped<int32, string>
			}
		`, "pair: base.Pair[str, int]\n"),
		Entry("an alias of a generic alias that reorders its type parameters", `
			Panel struct {
				pair Unswapped<int32, string>
			}
		`, "pair: base.Pair[int, str]\n"),
		Entry("a generic alias whose type arguments bind only defaulted parameters", `
			Panel struct<D? = string> {
				loose Loose<D>
			}
		`, "loose: Loose\n"),
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
