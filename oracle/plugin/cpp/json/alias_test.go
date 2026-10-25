// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/oracle/plugin/cpp/json"
	. "github.com/synnaxlabs/oracle/testutil"
)

const aliasTelem = `
@cpp output "x/cpp/telem"

TimeStamp int64 {
	@cpp hand
}

TimeSpan int64 {
	@cpp hand
}
`

const aliasBase = `
import "schemas/x/telem"

@cpp output "client/cpp/base"

Key = uuid

Stamp = telem.TimeStamp

Span = telem.TimeSpan

Labels = string[]

Labeled struct {
	label string
}

Valve struct extends Labeled {
	open bool
}

ValveList Valve[]

Edge union on variant {
	pipe Labeled
}

Signal union on kind {
	none {}
	level {
		value float64
	}
}
`

const aliasHead = `
import "schemas/base"

@cpp output "client/cpp/test"

Valve = base.Valve
Gate = Valve
ValveList = base.ValveList
Edge = base.Edge
Signal = base.Signal
Key = base.Key
Ident Key
Stamp = base.Stamp
Span = base.Span
Moment Stamp
Labels = base.Labels
Names Labels
`

var _ = Describe("C++ JSON aliases", func() {
	DescribeTable("Should code a use of an alias as a use of its target",
		func(ctx SpecContext, use string, expected ...string) {
			loader := NewMockFileLoader().
				Add("schemas/x/telem", aliasTelem).
				Add("schemas/base", aliasBase)
			resp := MustGenerate(
				ctx,
				aliasHead+use,
				"test",
				loader,
				json.New(json.DefaultOptions()),
			)
			ExpectContent(resp, "test/json.gen.h").ToContain(expected...)
		},
		Entry("a union", `
			Panel struct {
				edge Edge
			}
		`,
			`::synnax::base::parse_edge(parser.child("edge"))`,
			`j["edge"] = ::synnax::base::to_json(this->edge);`,
		),
		Entry("a union array", `
			Panel struct {
				edges Edge[]
			}
		`,
			"result.push_back(::synnax::base::parse_edge(p));",
			"arr.push_back(::synnax::base::to_json(item));",
		),
		Entry("a union default", `
			Panel struct {
				signal Signal = none
			}
		`, `: Signal{::synnax::base::NoneSignal{}}`),
		Entry("a struct array", `
			Panel struct {
				valves Valve[]
			}
		`, `j["valves"] = x::json::to_array(this->valves);`),
		Entry("an alias of an alias", `
			Panel struct {
				gate Gate
			}
		`,
			`j["gate"] = this->gate.to_json();`,
			`#include "client/cpp/base/json.gen.h"`,
		),
		Entry("a distinct array", `
			Panel struct {
				list ValveList
			}
		`,
			`.list = parser.field<ValveList>("list"),`,
			`j["list"] = this->list.to_json();`,
		),
		Entry("an alias of a timestamp", `
			Panel struct {
				created Stamp
			}
		`, `j["created"] = this->created.nanoseconds();`),
		Entry("an alias of a timespan", `
			Panel struct {
				span Span
			}
		`, `j["span"] = this->span.nanoseconds();`),
		Entry("a distinct type over an alias of a timestamp", `
			Panel struct {
				created Moment
			}
		`, `j["created"] = this->created.nanoseconds();`),
		Entry("an optional alias of a timestamp", `
			Panel struct {
				created Stamp?
			}
		`, `if (this->created.has_value()) `+
			`j["created"] = this->created->nanoseconds();`),
		Entry("a distinct type over an alias of a uuid", `
			Panel struct {
				key Ident
			}
		`, `j["key"] = this->key.to_json();`),
		Entry("an optional distinct type over an alias of a uuid", `
			Panel struct {
				key Ident?
			}
		`, `if (this->key.has_value()) j["key"] = this->key->to_json();`),
		Entry("an array of a distinct type over an alias of a uuid", `
			Panel struct {
				keys Ident[]
			}
		`, "for (const auto& item : this->keys) arr.push_back(item.to_json());"),
		Entry("a distinct type over an alias of an array", `
			Panel struct {
				names Names
			}
		`,
			`.names = parser.field<Names>("names"),`,
			`j["names"] = this->names.to_json();`,
			"inline Names Names::parse(x::json::Parser parser) {",
		),
		Entry("a distinct type over an alias of an array alone in its file", "",
			"inline Names Names::parse(x::json::Parser parser) {"),
		Entry("an optional distinct array", `
			Panel struct {
				list ValveList?
			}
		`,
			`.list = parser.field<std::optional<ValveList>>("list"),`,
			`if (this->list.has_value()) j["list"] = this->list->to_json();`,
		),
		Entry("a distinct array's elements", `
			Gates Gate[]

			Panel struct {
				gates Gates
			}
		`, "j.push_back(item.to_json());"),
	)
})
