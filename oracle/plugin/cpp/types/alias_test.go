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
	"github.com/synnaxlabs/oracle/plugin/cpp/types"
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

Level enum {
	low = 0
	high = 1
}

Method enum {
	password = "password"
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
@go output "core/pkg/service/test"
@pb

Size = base.Size
Level = base.Level
Method = base.Method
Labeled = base.Labeled
Valve = base.Valve
Credentials = base.Credentials
Code = base.Code
Signal = base.Signal
`

var _ = Describe("C++ aliases", func() {
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
			ExpectContent(resp, "test/types.gen.h").ToContain(expected...)
		},
		Entry("a string enum", `
			Panel struct {
				size Size
			}
		`, "using Size = std::string;"),
		Entry("an integer enum's zero value", `
			Panel struct {
				level Level
			}
		`, "Level level = {};"),
		Entry("a distinct type's zero value", `
			Panel struct {
				code Code
			}
		`, "Code code = 0;"),
		Entry("a distinct default", `
			Panel struct {
				code Code = 5
			}
		`, "Code code = Code(5);"),
		Entry("a struct default", `
			Panel struct {
				labeled Labeled = { label = "x" }
			}
		`, `Labeled labeled = {.label = "x"};`),
		Entry("a union default", `
			Panel struct {
				signal Signal = none
			}
		`, "Signal signal = ::synnax::base::NoneSignal{};"),
		Entry("a borrowed enum's members", `
			Proof union on method Method {
				password Credentials
			}
		`, "std::string method = ::synnax::base::METHOD_PASSWORD;"),
		Entry("a variant payload's proto message", `
			Node union on variant {
				valve Valve
			}
		`,
			`#include "core/pkg/service/base/pb/base.pb.h"`,
			"using proto_type = ::service::base::pb::Valve;",
		),
	)
})
