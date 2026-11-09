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

var _ = Describe("Presence types", func() {
	var (
		loader   *MockFileLoader
		goPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/patch", `
			@go output "x/go/patch"

			Change struct<T> {
				value T?
				@go hand
				@presence
			}
		`)
		goPlugin = types.New(types.DefaultOptions())
	})

	It("Should hold an optional field of a presence type by value", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, `
			import "schemas/x/patch"

			@go output "core/doc"

			Update struct {
				note  patch.Change<string>?
				title string?
			}
		`, "doc", loader, goPlugin)
		ExpectContent(resp, "core/doc/types.gen.go").ToContain(
			"Note patch.Change[string] `json:\"note,omitzero\" msgpack:\"note,omitempty\"`",
			"Title *string",
		)
	})

	It("Should encode a struct that holds a presence type through its package", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, `
			import "schemas/x/patch"

			@go output "core/doc"

			Update struct {
				note patch.Change<string>?
			}
		`, "doc", loader, goPlugin)
		ExpectContent(resp, "core/doc/types.gen.go").ToBeValidGoSource().ToContain(
			"func (u Update) EncodeMsgpack(enc *vmsgpack.Encoder) error {",
			"return patch.EncodeMsgpack(enc, u)",
			"func (u *Update) DecodeMsgpack(dec *vmsgpack.Decoder) error {",
			"return patch.DecodeMsgpack(dec, u)",
		)
	})

	It("Should give a struct without a presence field no msgpack codec", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, `
			@go output "core/doc"

			Update struct {
				title string?
			}
		`, "doc", loader, goPlugin)
		ExpectContent(resp, "core/doc/types.gen.go").ToNotContain("EncodeMsgpack")
	})
})
