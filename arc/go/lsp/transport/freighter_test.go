// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package transport_test

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/lsp/transport"
	"github.com/synnaxlabs/freighter/fmock"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Freighter Transport", func() {
	var (
		server       *lsp.Server
		clientStream *fmock.ClientStream[transport.JSONRPCMessage, transport.JSONRPCMessage]
		ctx          context.Context
		cancel       context.CancelFunc
		errChan      chan error
	)

	sendRequest := func(id int, method string, params any) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
		}
		if params != nil {
			req["params"] = params
		}
		content := MustSucceed(json.Marshal(req))
		Expect(clientStream.Send(transport.JSONRPCMessage{
			Content: string(content),
		})).To(Succeed())
	}

	sendNotification := func(method string, params any) {
		req := map[string]any{
			"jsonrpc": "2.0",
			"method":  method,
		}
		if params != nil {
			req["params"] = params
		}
		content := MustSucceed(json.Marshal(req))
		Expect(clientStream.Send(transport.JSONRPCMessage{
			Content: string(content),
		})).To(Succeed())
	}

	receiveResponse := func(id int) map[string]any {
		for {
			msg := MustSucceed(clientStream.Receive())
			var response map[string]any
			Expect(json.Unmarshal([]byte(msg.Content), &response)).To(Succeed())
			if respID, ok := response["id"]; ok && respID != nil {
				Expect(respID).To(BeEquivalentTo(id))
				return response
			}
		}
	}

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		server = MustSucceed(lsp.New(lsp.Config{
			Instrumentation: alamos.New("test"),
		}))
		var serverStream *fmock.ServerStream[transport.JSONRPCMessage, transport.JSONRPCMessage]
		clientStream, serverStream = fmock.NewStreams[transport.JSONRPCMessage, transport.JSONRPCMessage](ctx)
		errChan = make(chan error, 1)
		go func() {
			errChan <- transport.ServeFreighter(ctx, transport.Config{
				Server: server,
				Stream: serverStream,
			})
		}()
	})

	AfterEach(func() {
		cancel()
		Eventually(errChan).Should(Receive())
	})

	Describe("Initialize", func() {
		It("Should handle an initialize request and return capabilities", func() {
			sendRequest(1, "initialize", map[string]any{
				"clientInfo": map[string]any{"name": "test-client"},
			})
			response := receiveResponse(1)
			result, ok := response["result"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(result).To(HaveKey("capabilities"))
		})
	})

	Describe("DidOpen + Hover", func() {
		It("Should handle document open notification followed by hover request", func() {
			sendRequest(1, "initialize", map[string]any{
				"clientInfo": map[string]any{"name": "test-client"},
			})
			receiveResponse(1)
			sendNotification("initialized", map[string]any{})
			sendNotification("textDocument/didOpen", map[string]any{
				"textDocument": map[string]any{
					"uri":        "file:///test.arc",
					"languageId": "arc",
					"version":    1,
					"text":       "x := 1\n",
				},
			})
			sendRequest(2, "textDocument/hover", map[string]any{
				"textDocument": map[string]any{"uri": "file:///test.arc"},
				"position":     map[string]any{"line": 0, "character": 0},
			})
			receiveResponse(2)
		})
	})

	Describe("Multiple sequential requests", func() {
		It("Should handle multiple requests over the same connection", func() {
			sendRequest(1, "initialize", map[string]any{
				"clientInfo": map[string]any{"name": "test-client"},
			})
			receiveResponse(1)

			sendRequest(2, "shutdown", nil)
			receiveResponse(2)
		})
	})

	Describe("Connection close", func() {
		It("Should exit cleanly when the client stream closes", func() {
			sendRequest(1, "initialize", map[string]any{
				"clientInfo": map[string]any{"name": "test-client"},
			})
			receiveResponse(1)
			Expect(clientStream.CloseSend()).To(Succeed())
		})
	})

	Describe("JSONRPCMessage", func() {
		Describe("UnmarshalJSON", func() {
			It("Should unmarshal valid JSON object", func() {
				var msg transport.JSONRPCMessage
				input := `{"content":"test message"}`
				Expect(json.Unmarshal([]byte(input), &msg)).To(Succeed())
				Expect(msg.Content).To(Equal("test message"))
			})
			It("Should handle raw string as content", func() {
				var msg transport.JSONRPCMessage
				input := `"raw string content"`
				Expect(json.Unmarshal([]byte(input), &msg)).To(Succeed())
				Expect(msg.Content).To(Equal(input))
			})
			It("Should handle empty object", func() {
				var msg transport.JSONRPCMessage
				input := `{}`
				Expect(json.Unmarshal([]byte(input), &msg)).To(Succeed())
				Expect(msg.Content).To(Equal(""))
			})
		})
	})
})
