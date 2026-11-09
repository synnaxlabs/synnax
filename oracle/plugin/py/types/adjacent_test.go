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

var _ = Describe("Python adjacent union generation", func() {
	It("Should carry each payload beside the tag", func(ctx SpecContext) {
		resp := MustGenerate(ctx, `
			@py output "client/py/synnax/board"

			Board struct {
				key uuid

				action SetNodePosition {
					x float64
				}

				action Import {
					from string
				}
			}
		`, "board", NewMockFileLoader(), types.New(types.DefaultOptions()))
		ExpectContent(resp, "types_gen.py").ToContain(
			"class SetNodePositionPayload(BaseModel):",
			"class SetNodePositionAction(BaseModel):\n"+
				`    type: Literal["set_node_position"] = "set_node_position"`+"\n"+
				"    set_node_position: SetNodePositionPayload\n",
			`import_: ImportPayload = Field(alias="import")`,
			"Action = Annotated[",
		)
	})
})
