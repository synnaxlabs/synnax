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

var _ = Describe("Go adjacent union generation", func() {
	var (
		loader   *MockFileLoader
		goPlugin *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/patch", `
			@go output "x/go/patch"

			Set struct {
				path string[]

				@doc value "replaces the value at a path."
			}
			Remove struct { path string[] }
			Insert struct { path string[] }
			Move struct { path string[] }
		`)
		goPlugin = types.New(types.DefaultOptions())
	})

	const source = `
		import "schemas/x/patch"

		@go output "core/pkg/service/counter"

		Named struct {
			name string
		}

		Counter struct {
			key   uuid
			value int32

			action Rename extends Named {
				force int32
			}

			@actions
		}
	`

	It("Should declare each payload as a variant of the Action union", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, source, "counter", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").
			ToBeValidGoSource().
			ToContain(
				"type CreatePayload struct {",
				"Counter Counter `json:\"counter\" msgpack:\"counter\"`",
				"// SetPayload replaces the value at a path.\ntype SetPayload patch.Set\n",
				"type RenamePayload struct {",
				"Named\n",
				"Force int32 `json:\"force\" msgpack:\"force\"`",
				"// Action holds one counter mutation.\ntype Action struct {\n"+
					"\tVariant ActionVariant\n}",
				"func (CreatePayload) isActionVariant() {}",
				"func (RenamePayload) isActionVariant() {}",
				`CreateActionType ActionType = "create"`,
				`RenameActionType ActionType = "rename"`,
				"func (u Action) Type() ActionType {",
			).
			ToNotContain("type CreateAction struct", "type RenameAction struct")
	})

	It("Should carry the payload beside the tag in JSON", func(ctx SpecContext) {
		resp := MustGenerate(ctx, source, "counter", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			"}{Type: RenameActionType, Payload: v})",
			"Payload RenamePayload `json:\"rename\"`",
			"Payload *RenamePayload `json:\"rename\"`",
			"if v.Payload == nil {\n\t\t\treturn union.MissingPayload(string(disc.Type))",
			"return union.UnknownVariant(string(disc.Type))",
		)
	})

	It("Should carry the payload beside the tag in msgpack", func(ctx SpecContext) {
		resp := MustGenerate(ctx, source, "counter", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToContain(
			`vmsgpack "github.com/vmihailenco/msgpack/v5"`,
			`"github.com/vmihailenco/msgpack/v5/msgpcode"`,
			"func (u Action) EncodeMsgpack(enc *vmsgpack.Encoder) error {",
			"Payload RenamePayload `msgpack:\"rename\"`",
			"func (u *Action) DecodeMsgpack(dec *vmsgpack.Decoder) error {",
			"if len(raw) == 1 && raw[0] == msgpcode.Nil {",
			"Payload *RenamePayload `msgpack:\"rename\"`",
		)
	})

	It("Should generate no Action union for a type with no actions", func(
		ctx SpecContext,
	) {
		resp := MustGenerate(ctx, `
			@go output "core/pkg/service/counter"

			Counter struct {
				key uuid
			}
		`, "counter", loader, goPlugin)
		ExpectContent(resp, "types.gen.go").ToNotContain("Action")
	})
})
