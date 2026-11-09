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

var _ = Describe("Update types", func() {
	It("Should decode an update without the type as base fields only", func(
		ctx SpecContext,
	) {
		loader := NewMockFileLoader()
		loader.Add("schemas/x/patch", `
			@go output "x/go/patch"

			Set struct { path string[] }
			Remove struct { path string[] }
			Insert struct { path string[] }
			Move struct { path string[] }
			Change struct<T> {
				value T?
				@presence
			}
		`)
		resp := MustGenerate(ctx, `
			import "schemas/x/patch"

			@go output "core/pkg/service/view"

			Base struct {
				key  uuid
				name string { @patch }
			}

			View union on type extends Base {
				range {
					where string? { @patch }
				}
				status {
					level int32 { @patch }
				}
				@actions
			}
		`, "view", loader, types.New(types.DefaultOptions()))
		ExpectContent(resp, "core/pkg/service/view/types.gen.go").ToContain(
			"func (BaseUpdate) isUpdateVariant() {}",
			"\tcase BaseUpdate:\n\t\treturn json.MarshalEncode(enc, v)\n",
			"\tcase \"\":\n\t\tvar v BaseUpdate\n\t\tif err := json.Unmarshal(\n"+
				"\t\t\tdata, &v, opts, json.RejectUnknownMembers(true),\n",
		)
	})
})
