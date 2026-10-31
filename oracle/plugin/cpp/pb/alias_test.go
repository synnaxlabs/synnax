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

const aliasTelem = `
@cpp output "x/cpp/telem"
@go output "x/go/telem"

TimeStamp int64 {
	@cpp hand
}
`

const aliasBase = `
import "schemas/x/telem"

@cpp output "client/cpp/base"
@go output "core/pkg/service/base"
@pb

Key = uuid

Labels = string[]

Hash uint8[32]

Raw = uint8[16]

Stamp = telem.TimeStamp

Box struct<T> {
	item T
}

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
Key = base.Key
Ident Key
Labels = base.Labels
Names Labels
Tags Labels {
	@pb name "Tags"
}
Raw = base.Raw
Digest Raw
Hash = base.Hash
Stamp = base.Stamp
Moment Stamp
Grade Size
Fitting Valve
Box<T> = base.Box<T>
Crate<T> Box<T>
Valves Valve[] {
	@pb name "Valves"
}
Doc = record
`

var _ = Describe("C++ PB aliases", func() {
	DescribeTable("Should translate a use of an alias as a use of its target",
		func(ctx SpecContext, use string, expected ...string) {
			loader := NewMockFileLoader().
				Add("schemas/x/telem", aliasTelem).
				Add("schemas/base", aliasBase)
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
		Entry("a map of a struct", `
			Panel struct {
				byName map<string, Valve>
			}
		`, "auto [pb_v, err] = v.to_proto();", "Valve::from_proto(v)"),
		Entry("a map of a record", `
			Panel struct {
				docs map<string, Doc>
			}
		`, "x::json::to_struct(v)", "x::json::from_struct(v)"),
		Entry("a nested array of a struct", `
			Panel struct {
				grid Valves[]
			}
		`,
			"auto [v_pb, err] = v.to_proto();",
			"x::pb::from_proto_repeated<Valve>(inner, wrapper.values())",
		),
		Entry("an array wrapper of a struct", `
			Panel struct {
				valves Valves
			}
		`, "x::pb::from_proto_repeated<Valve>(cpp, pb.values())"),
		Entry("a distinct type over an alias of a uuid", `
			Panel struct {
				key Ident
			}
		`,
			"pb.set_key(this->key.to_string());",
			"auto [v, err] = x::uuid::UUID::parse(pb.key());",
		),
		Entry("an optional distinct type", `
			Panel struct {
				code Code?
			}
		`,
			"if (this->code.has_value()) pb.set_code(*this->code);",
			"if (pb.has_code()) cpp.code = pb.code();",
		),
		Entry("a distinct type over an alias of an array", `
			Panel struct {
				names Names
			}
		`,
			"for (const auto& item : this->names) pb.add_names(item);",
			"for (const auto& item : pb.names()) cpp.names.push_back(item);",
		),
		Entry("a distinct type over an alias of an array with a proto message", "",
			"pb.add_values(item)",
			"cpp.push_back(item)",
		),
		Entry("a distinct type over an alias of a fixed-size byte array", `
			Panel struct {
				digest Digest
			}
		`,
			"pb.set_digest(this->digest.data(), this->digest.size());",
		),
		Entry("an optional distinct array", `
			Panel struct {
				names Names?
			}
		`,
			"auto* wrapper = pb.mutable_names();",
			"cpp.names.emplace();",
		),
		Entry("an alias of a fixed-size byte array", `
			Panel struct {
				hash Hash
			}
		`,
			"pb.set_hash(this->hash.data(), this->hash.size());",
			"std::copy(pb.hash().begin(), pb.hash().end(), cpp.hash.begin());",
		),
		Entry("a distinct type over an alias of a hand-written type", `
			Panel struct {
				created Moment
			}
		`,
			"pb.set_created(this->created.to_proto());",
			"cpp.created = ::x::telem::TimeStamp::from_proto(pb.created());",
		),
		Entry(
			"an optional alias of a hand-written type",
			`
			Panel struct {
				created Stamp?
			}
		`,
			"if (this->created.has_value()) pb.set_created(this->created->to_proto());",
			"if (pb.has_created()) "+
				"cpp.created = ::x::telem::TimeStamp::from_proto(pb.created());",
		),
		Entry("a distinct type over an alias of an enum", `
			Panel struct {
				grade Grade
			}
		`,
			"auto [v, err] = ::synnax::base::size_to_pb(this->grade);",
		),
		Entry("a distinct type over an alias of a struct", `
			Panel struct {
				fitting Fitting
			}
		`,
			"auto [v, err] = Valve::from_proto(pb.fitting());",
		),
		Entry("a generic distinct type over a generic alias", `
			Panel struct {
				crate Crate<string>
			}
		`,
			"auto [v, err] = string_box_from_proto(pb.crate());",
		),
		Entry("an alias of a struct", `
			Panel struct {
				valve Valve?
			}
		`, "auto [v, err] = Valve::from_proto(pb.valve());"),
		Entry("a distinct type", `
			Panel struct {
				code Code
			}
		`, "pb.set_code(this->code);", "cpp.code = pb.code();"),
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
