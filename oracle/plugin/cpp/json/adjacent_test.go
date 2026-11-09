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

var _ = Describe("C++ JSON adjacent unions and any fields", func() {
	var (
		loader     *MockFileLoader
		jsonPlugin *json.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		jsonPlugin = json.New(json.Options{FileNamePattern: "json.gen.h"})
	})

	It("Should read and write each payload beside the tag", func(ctx SpecContext) {
		resp := MustGenerate(ctx, `
			@cpp output "client/cpp/board"

			Board struct {
				key uuid

				action SetNodePosition {
					x float64
				}

				action Reset {}
			}
		`, "board", loader, jsonPlugin)
		ExpectContent(resp, "client/cpp/board/json.gen.h").ToContain(
			"inline Action parse_action(x::json::Parser parser) {",
			`const auto discriminator = parser.field<std::string>("type");`,
			`if (discriminator == "set_node_position") return `+
				`SetNodePositionPayload::parse(parser.child("set_node_position"));`,
			`parser.field_err("type", "unknown Action type: " + discriminator);`,
			"inline x::json::json to_json(const Action& value) {",
			`{"set_node_position", "reset"};`,
			"const std::string discriminator = discriminators[value.index()];",
			`j["type"] = discriminator;`,
			"j[discriminator] = std::visit(",
		)
	})

	It("Should read an absent bare any field as null", func(ctx SpecContext) {
		resp := MustGenerate(ctx, `
			@cpp output "client/cpp/board"

			Cell struct {
				value any
				fill  any?
			}
		`, "board", loader, jsonPlugin)
		ExpectContent(resp, "client/cpp/board/json.gen.h").ToContain(
			`parser.field<x::json::json>("value", nullptr)`,
		)
	})
})
