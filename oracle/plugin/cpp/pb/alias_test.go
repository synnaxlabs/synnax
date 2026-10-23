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
	"github.com/synnaxlabs/oracle/plugin/cpp/pb"
	. "github.com/synnaxlabs/oracle/testutil"
)

const aliasBase = `
@cpp output "client/cpp/base"
@go output "core/pkg/service/base"
@pb

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

Edge union on variant {
	pipe Labeled
}
`

const aliasHead = `
import "schemas/base"

@cpp output "client/cpp/test"
@go output "core/pkg/service/test"
@pb

Size = base.Size
Labeled = base.Labeled
Valve = base.Valve
Credentials = base.Credentials
Code = base.Code
Edge = base.Edge
`

var _ = Describe("C++ PB aliases", func() {
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
			ExpectContent(resp, "test/proto.gen.h").ToContain(expected...)
		},
		Entry("an enum", `
			Panel struct {
				size Size
			}
		`, "::synnax::base::size_to_pb(this->size)"),
		Entry("an enum array", `
			Panel struct {
				sizes Size[]
			}
		`, "::synnax::base::size_to_pb(item)"),
		Entry("a union", `
			Panel struct {
				edge Edge
			}
		`,
			"::synnax::base::to_proto(this->edge)",
			"::synnax::base::edge_from_proto(pb.edge())",
		),
		Entry("a struct array", `
			Panel struct {
				valves Valve[]
			}
		`, "auto [v, err] = item.to_proto();"),
		Entry("a distinct type", `
			Panel struct {
				code Code
			}
		`, "cpp.code = ::synnax::base::Code(pb.code());"),
		Entry("a variant payload", `
			Node union on variant {
				valve Valve
			}
		`,
			"std::pair<::service::base::pb::Valve, x::errors::Error> "+
				"ValveNode::to_proto() const {",
			"pb.set_open(this->open);",
		),
		Entry("a union base", `
			Shape union on kind extends Labeled {
				circle Credentials
			}
		`, "static_cast<const ::synnax::base::Labeled&>(v).to_proto()"),
	)
})
