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

const aliasBase = `
@go output "core/pkg/base"

Size enum {
	small = "small"
	large = "large"
}

Method enum {
	password = "password"
}

Credentials struct {
	username string
}

Labeled struct {
	label string
}

Data = record

Blob = bytes

Labels = string[]

Index = map<string, int32>

Signal union on kind {
	none {}
	level {
		value float64
	}
}

Pair struct<T, U = string> {
	first  T
	second U
}
`

const aliasHead = `
import "schemas/base"

@go output "core/pkg/test"

Size = base.Size
Method = base.Method
Credentials = base.Credentials
Labeled = base.Labeled
Signal = base.Signal
Data = base.Data
Blob = base.Blob
Labels = base.Labels
Index = base.Index
Names Labels
Table Index
Doc Data
Payload Blob
Quad float64[4]
Couple<T, U = string> = base.Pair<T, U>
`

var _ = Describe("Go aliases", func() {
	DescribeTable("Should generate a use of an alias as a use of its target",
		func(ctx SpecContext, use string, expected ...string) {
			loader := NewMockFileLoader().Add("schemas/base", aliasBase)
			resp := MustGenerate(
				ctx,
				aliasHead+use,
				"test",
				loader,
				types.New(types.DefaultOptions()),
			)
			ExpectContent(resp, "test/types.gen.go").ToContain(expected...)
		},
		Entry("a borrowed enum's members", `
			Proof union on method Method {
				password Credentials
			}
		`, "case base.MethodPassword:"),
		Entry("an enum check", `
			Panel struct {
				size Size
			}
		`, `v.Ternaryf("size", !p.Size.IsValid(), "invalid size: %v", p.Size)`),
		Entry("an enum default", `
			Panel struct {
				size Size = large
			}
		`, "p.Size = base.SizeLarge"),
		Entry("a union default", `
			Panel struct {
				signal Signal = none
			}
		`, "p.Signal.Variant = base.NoneSignal{}"),
		Entry("a struct default", `
			Panel struct {
				labeled Labeled = { label = "x" }
			}
		`, `p.Labeled.Label = "x"`),
		Entry("a union literal default", `
			Panel struct {
				signal Signal = { kind = "level", value = 1 }
			}
		`, "v0 := base.LevelSignal{}", "v0.Value = 1"),
		Entry("an optional alias of record", `
			Panel struct {
				data Data?
			}
		`, "Data Data `"),
		Entry("an optional alias of bytes", `
			Panel struct {
				blob Blob?
			}
		`, "Blob Blob `"),
		Entry("an optional alias of an array", `
			Panel struct {
				labels Labels?
			}
		`, "Labels Labels `"),
		Entry("an optional alias of a map", `
			Panel struct {
				index Index?
			}
		`, "Index Index `"),
		Entry("an optional alias of a struct", `
			Panel struct {
				labeled Labeled?
			}
		`, "Labeled *Labeled `"),
		Entry("an optional distinct type over an alias of an array", `
			Panel struct {
				names Names?
			}
		`, "Names Names `"),
		Entry("an optional distinct type over an alias of a map", `
			Panel struct {
				table Table?
			}
		`, "Table Table `"),
		Entry("an optional distinct type over an alias of record", `
			Panel struct {
				doc Doc?
			}
		`, "Doc Doc `"),
		Entry("an optional distinct type over an alias of bytes", `
			Panel struct {
				payload Payload?
			}
		`, "Payload Payload `"),
		Entry("an optional fixed-size array", `
			Panel struct {
				corners Quad?
			}
		`, "Corners *Quad `"),
		Entry("a parent alias passed a defaulted type argument", `
			Child struct extends Couple<int32, string> {
				depth float64
			}
		`, "\tCouple[int32]\n"),
	)
})
