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

var _ = Describe("Presence types", func() {
	It("Should hold an optional presence field without std::optional", func(
		ctx SpecContext,
	) {
		loader := NewMockFileLoader()
		loader.Add("schemas/x/patch", `
			@cpp output "x/cpp/patch"

			Change struct<T> {
				value T?
				@cpp hand
				@presence
			}
		`)
		resp := MustGenerate(ctx, `
			import "schemas/x/patch"

			@cpp output "client/cpp/doc"

			Update struct {
				note  patch.Change<string>?
				title string?
			}
		`, "doc", loader, types.New(types.DefaultOptions()))
		ExpectContent(resp, "types.gen.h").
			ToContain(
				"::x::patch::Change<std::string> note;",
				"std::optional<std::string> title;",
			).
			ToNotContain("std::optional<::x::patch::Change")
	})
})
