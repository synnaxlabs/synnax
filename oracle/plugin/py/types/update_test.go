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

var _ = Describe("Update types", func() {
	It("Should render an update as a TypedDict without Core-only fields", func(
		ctx SpecContext,
	) {
		loader := NewMockFileLoader()
		loader.Add("schemas/x/patch", `
			@go output "x/go/patch"
			@py output "x/py/x/patch"

			Set struct { path string[] }
			Remove struct { path string[] }
			Insert struct { path string[] }
			Move struct { path string[] }
			Change struct<T> {
				value T?
				@presence
				@py hand
				@py include "x/py/x/patch/struct"
			}
			ValueList struct<E> {
				set E[]?
			}
		`)
		resp := MustGenerate(ctx, `
			import "schemas/x/patch"

			@go output "core/pkg/service/plot"
			@py output "client/py/synnax/plot"

			Plot struct {
				key  uuid
				name string   { @patch }
				note string?  { @patch }
				tags string[] { @patch }
				hash string   { @patch core }

				@actions
			}
		`, "plot", loader, types.New(types.DefaultOptions()))
		ExpectContent(resp, "client/py/synnax/plot/types_gen.py").ToContain(
			"class Update(TypedDict, total=False):",
			"    name: str\n",
			"    note: patch.Change[str]\n",
			"    tags: patch.ValueListDict[str]\n\n\nclass CreatePayload",
		)
	})
})
