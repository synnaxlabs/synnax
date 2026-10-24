// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/oracle/plugin/go/pb"
	. "github.com/synnaxlabs/oracle/testutil"
)

const aliasBase = `
@go output "core/pkg/service/base"
@pb

Key = uuid

Size enum {
	small = "small"
	large = "large"
}

Labeled struct {
	label string
}

Valve struct extends Labeled {
	open bool
}

Credentials struct {
	username string
}

Code uint32

Details struct {
	label string
}

Member struct {
	key string
}

Edge union on variant {
	pipe Labeled
}
`

const aliasHead = `
import "schemas/base"

@go output "core/pkg/service/test"
@pb

Key = base.Key
Size = base.Size
Labeled = base.Labeled
Valve = base.Valve
Gate = Valve
Credentials = base.Credentials
Details = base.Details
Member = base.Member
Code = base.Code
Edge = base.Edge
`

var _ = Describe("Go PB aliases", func() {
	DescribeTable("Should translate a use of an alias as a use of its target",
		func(ctx SpecContext, use string, expected ...string) {
			loader := NewMockFileLoader().Add("schemas/base", aliasBase)
			resp := MustGenerate(
				ctx,
				aliasHead+use,
				"test",
				loader,
				pb.New(pb.DefaultOptions()),
			)
			ExpectContent(resp, "test/pb/translator.gen.go").ToContain(expected...)
		},
		Entry("an alias of a primitive alias", `
			Panel struct {
				key Key
			}
		`, "Key: r.Key.String(),", "r.Key = test.Key(parsedKey)"),
		Entry("an enum", `
			Panel struct {
				size Size
			}
		`, "basepb.SizeToPB(r.Size)"),
		Entry("an enum array", `
			Panel struct {
				sizes Size[]
			}
		`, "func(v base.Size, _ int) (basepb.Size, error) { "+
			"return basepb.SizeToPB(v) }"),
		Entry("a union", `
			Panel struct {
				edge Edge
			}
		`, "basepb.EdgeToPB(r.Edge)"),
		Entry("a struct array", `
			Panel struct {
				valves Valve[]
			}
		`, "basepb.ValvesToPB(r.Valves)"),
		Entry("a distinct type", `
			Panel struct {
				code Code
			}
		`, "Code: uint32(r.Code),", "r.Code = base.Code(pb.Code)"),
		Entry("a variant payload", `
			Node union on variant {
				valve Valve
			}
		`, "inner, err := basepb.ValveToPB(v.Valve)"),
		Entry("a union base", `
			Shape union on kind extends Labeled {
				circle Credentials
			}
		`, "basepb.LabeledToPB(v.Labeled)"),
		Entry("an optional union", `
			Panel struct {
				edge Edge?
			}
		`, "val, err := basepb.EdgeFromPB(pb.Edge)", "r.Edge = &val"),
		Entry("an optional alias of an alias", `
			Panel struct {
				gate Gate?
			}
		`, "val, err := basepb.ValveFromPB(pb.Gate)", "r.Gate = &val"),
		Entry("an optional distinct type", `
			Panel struct {
				code Code?
			}
		`, "v := uint32(*r.Code)", "v := base.Code(*pb.Code)"),
		Entry("a map of unions", `
			Panel struct {
				edges map<string, Edge>
			}
		`, "converted, err := basepb.EdgeToPB(v)", "converted, err := basepb.EdgeFromPB(v)"),
		Entry("a generic struct's type argument", `
			Status struct<D> {
				key     string
				details D
			}

			Panel struct {
				status Status<Details>
			}
		`,
			"StatusToPB(r.Status, DetailsToPBAny)",
			"func DetailsToPBAny(v base.Details) (*anypb.Any, error) {",
			"pb, err := basepb.DetailsToPB(v)",
			"var pb basepb.Details",
		),
		Entry("a nested array's element", `
			Members = Member[]

			Strata Members[]

			Layers = Strata

			Panel struct {
				layers Layers
			}
		`, "vals, err := basepb.MembersToPB(inner)", "func() (test.Layers, error) {"),
	)
})
