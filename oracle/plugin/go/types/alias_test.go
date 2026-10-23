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

Signal union on kind {
	none {}
	level {
		value float64
	}
}
`

const aliasHead = `
import "schemas/base"

@go output "core/pkg/test"

Size = base.Size
Method = base.Method
Credentials = base.Credentials
Signal = base.Signal
`

var _ = Describe("Go aliases", func() {
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
			ExpectContent(resp, "test/types.gen.go").ToContain(expected)
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
	)
})
