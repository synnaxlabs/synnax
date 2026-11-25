// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
)

var _ = Describe("Document breaker", func() {
	// A client counts a change's columns in UTF-16 units, as the word lookups now do.
	DescribeTable("Should apply a change whose range is in UTF-16 columns",
		func(content string, char uint32, want string) {
			change := &protocol.TextDocumentContentChangePartial{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: char},
					End:   protocol.Position{Line: 0, Character: char},
				},
				Text: " * 2",
			}
			Expect(lsp.ApplyIncrementalChange(content, change)).To(Equal(want))
		},
		Entry("after a two-byte character", "x := 'é' + 1", uint32(8),
			"x := 'é' * 2 + 1"),
		Entry("after an astral character", "x := '\U0001F525' + 1", uint32(9),
			"x := '\U0001F525' * 2 + 1"),
	)
})
