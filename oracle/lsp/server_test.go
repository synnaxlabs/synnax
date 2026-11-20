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
	"encoding/json"
	"log"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/synnaxlabs/oracle/lsp"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

var _ = Describe("Server", func() {
	It("Should log a diagnostics publish that fails", func(ctx SpecContext) {
		serverSide, clientSide := net.Pipe()
		logs := gbytes.NewBuffer()
		served := make(chan error, 1)
		go func() { served <- lsp.New(log.New(logs, "", 0)).Serve(ctx, serverSide) }()
		params := MustSucceed(json.Marshal(protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:     "file:///test.oracle",
				Version: 1,
				Text:    "User struct {}\n",
			},
		}))
		MustSucceed(jsonrpc2.NewStream(clientSide).Write(
			ctx,
			jsonrpc2.NewNotification(protocol.MethodTextDocumentDidOpen, params),
		))
		// The client never reads, so the publish fails once the client closes.
		Expect(clientSide.Close()).To(Succeed())
		Eventually(served).Should(Receive())
		Expect(logs).To(gbytes.Say(
			"failed to publish diagnostics for file:///test.oracle: ",
		))
	})
})
