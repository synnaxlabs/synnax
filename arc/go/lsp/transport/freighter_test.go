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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/formatter"
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/lsp/transport"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/freighter/mock"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Freighter Transport", func() {
	var (
		server       *lsp.Server
		clientStream *mock.ClientStream[transport.JSONRPCMessage, transport.JSONRPCMessage]
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

	// ctx is derived from context.Background() because it is stored in a shared
	// var and used by helper functions and a goroutine, so it must outlive BeforeEach.
	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		server = MustSucceed(lsp.New(lsp.Config{
			Instrumentation: alamos.New("test"),
			NewRoot: func() *symbol.Symbol {
				return NewRoot(nil)
			},
		}))
		var serverStream *mock.ServerStream[transport.JSONRPCMessage, transport.JSONRPCMessage]
		clientStream, serverStream = mock.NewStreams[transport.JSONRPCMessage, transport.JSONRPCMessage](
			ctx,
		)
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
		It(
			"Should handle an initialize request and return capabilities",
			func(ctx SpecContext) {
				sendRequest(1, "initialize", map[string]any{
					"clientInfo": map[string]any{"name": "test-client"},
				})
				response := receiveResponse(1)
				result, ok := response["result"].(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(result).To(HaveKey("capabilities"))
			},
		)
	})

	Describe("DidOpen + Hover", func() {
		It(
			"Should handle document open notification followed by hover request",
			func(ctx SpecContext) {
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
			},
		)
	})

	Describe("Symbol requests", func() {
		// The TS client probes these methods without checking capabilities; they must
		// answer with an empty result, not method-not-found.
		It(
			"Should return empty results for documentSymbol and workspace/symbol",
			func(ctx SpecContext) {
				sendRequest(1, "initialize", map[string]any{
					"clientInfo": map[string]any{"name": "test-client"},
				})
				receiveResponse(1)
				sendRequest(2, "workspace/symbol", map[string]any{"query": "test"})
				Expect(receiveResponse(2)).ToNot(HaveKey("error"))
				sendRequest(3, "textDocument/documentSymbol", map[string]any{
					"textDocument": map[string]any{"uri": "file:///test.arc"},
				})
				Expect(receiveResponse(3)).ToNot(HaveKey("error"))
			},
		)
	})

	Describe("Multiple sequential requests", func() {
		It(
			"Should handle multiple requests over the same connection",
			func(ctx SpecContext) {
				sendRequest(1, "initialize", map[string]any{
					"clientInfo": map[string]any{"name": "test-client"},
				})
				receiveResponse(1)

				sendRequest(2, "shutdown", nil)
				receiveResponse(2)
			},
		)
	})

	Describe("Notification ordering", func() {
		// Each didChange replaces the span of its predecessor's text, so the edits
		// are only correct when applied in wire order. The formatting probe reveals
		// the server's final content.
		It(
			"Should apply rapid incremental didChange edits in wire order",
			func(ctx SpecContext) {
				sendRequest(1, "initialize", map[string]any{
					"clientInfo": map[string]any{"name": "test-client"},
				})
				receiveResponse(1)
				docURI := "file:///ordering.arc"
				content := "x:=0"
				sendNotification("textDocument/didOpen", map[string]any{
					"textDocument": map[string]any{
						"uri":        docURI,
						"languageId": "arc",
						"version":    1,
						"text":       content,
					},
				})
				for i := 1; i <= 100; i++ {
					next := fmt.Sprintf("x:=%d", 1000+i)
					sendNotification("textDocument/didChange", map[string]any{
						"textDocument": map[string]any{"uri": docURI, "version": i + 1},
						"contentChanges": []map[string]any{{
							"range": map[string]any{
								"start": map[string]any{"line": 0, "character": 0},
								"end": map[string]any{
									"line":      0,
									"character": len(content),
								},
							},
							"text": next,
						}},
					})
					content = next
				}
				sendRequest(2, "textDocument/formatting", map[string]any{
					"textDocument": map[string]any{"uri": docURI},
					"options":      map[string]any{"tabSize": 4, "insertSpaces": true},
				})
				response := receiveResponse(2)
				result, ok := response["result"].([]any)
				Expect(ok).To(BeTrue(), "formatting should return edits")
				Expect(result).To(HaveLen(1))
				edit, ok := result[0].(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(edit["newText"]).To(Equal(formatter.Format(content)))
			},
		)
	})

	Describe("Connection close", func() {
		It("Should exit cleanly when the client stream closes", func(ctx SpecContext) {
			sendRequest(1, "initialize", map[string]any{
				"clientInfo": map[string]any{"name": "test-client"},
			})
			receiveResponse(1)
			Expect(clientStream.CloseSend()).To(Succeed())
		})
	})

	Describe("Eager client", func() {
		// Pins the streamAdapter ready gate: a notification queued before
		// ServeFreighter starts must not reach a handler until SetClient has wired the
		// client, or publishDiagnostics would panic on a nil client.
		It(
			"Should publish diagnostics for a didOpen queued before the server starts",
			func() {
				eagerCtx, eagerCancel := context.WithTimeout(
					context.Background(),
					5*time.Second,
				)
				defer eagerCancel()
				eagerServer := MustSucceed(lsp.New(lsp.Config{
					Instrumentation: alamos.New("eager"),
					NewRoot:         func() *symbol.Symbol { return NewRoot(nil) },
				}))
				eagerClient, eagerStream := mock.NewStreams[transport.JSONRPCMessage, transport.JSONRPCMessage](
					eagerCtx,
					1,
				)
				content := MustSucceed(json.Marshal(map[string]any{
					"jsonrpc": "2.0",
					"method":  "textDocument/didOpen",
					"params": map[string]any{
						"textDocument": map[string]any{
							"uri":        "file:///eager.arc",
							"languageId": "arc",
							"version":    1,
							"text":       "func broken( {",
						},
					},
				}))
				Expect(eagerClient.Send(transport.JSONRPCMessage{
					Content: string(content),
				})).To(Succeed())
				eagerErr := make(chan error, 1)
				go func() {
					eagerErr <- transport.ServeFreighter(eagerCtx, transport.Config{
						Server: eagerServer,
						Stream: eagerStream,
					})
				}()
				for {
					msg := MustSucceed(eagerClient.Receive())
					var note map[string]any
					Expect(json.Unmarshal([]byte(msg.Content), &note)).To(Succeed())
					if note["method"] == "textDocument/publishDiagnostics" {
						break
					}
				}
				eagerCancel()
				Eventually(eagerErr).Should(Receive())
			},
		)
	})

	Describe("JSONRPCMessage", func() {
		Describe("UnmarshalJSON", func() {
			It("Should unmarshal valid JSON object", func(ctx SpecContext) {
				var msg transport.JSONRPCMessage
				input := `{"content":"test message"}`
				Expect(json.Unmarshal([]byte(input), &msg)).To(Succeed())
				Expect(msg.Content).To(Equal("test message"))
			})
			It("Should handle raw string as content", func(ctx SpecContext) {
				var msg transport.JSONRPCMessage
				input := `"raw string content"`
				Expect(json.Unmarshal([]byte(input), &msg)).To(Succeed())
				Expect(msg.Content).To(Equal(input))
			})
			It("Should handle empty object", func(ctx SpecContext) {
				var msg transport.JSONRPCMessage
				input := `{}`
				Expect(json.Unmarshal([]byte(input), &msg)).To(Succeed())
				Expect(msg.Content).To(Equal(""))
			})
		})
	})
})

var _ = Describe("Transport Failure Modes", func() {
	var (
		fCtx    context.Context
		fCancel context.CancelFunc
	)

	BeforeEach(func() {
		fCtx, fCancel = context.WithTimeout(context.Background(), 5*time.Second)
	})

	AfterEach(func() { fCancel() })

	serve := func(maxLen int) (
		*mock.ClientStream[transport.JSONRPCMessage, transport.JSONRPCMessage],
		chan error,
	) {
		server := MustSucceed(lsp.New(lsp.Config{
			Instrumentation: alamos.New("failure"),
			NewRoot:         func() *symbol.Symbol { return NewRoot(nil) },
		}))
		client, stream := mock.NewStreams[transport.JSONRPCMessage, transport.JSONRPCMessage](
			fCtx,
			1,
		)
		errs := make(chan error, 1)
		go func() {
			errs <- transport.ServeFreighter(fCtx, transport.Config{
				Server:           server,
				Stream:           stream,
				MaxContentLength: maxLen,
			})
		}()
		return client, errs
	}

	It("Should reject a config missing the server and stream", func(ctx SpecContext) {
		Expect(
			transport.ServeFreighter(ctx),
		).To(MatchError(ContainSubstring("must be non-nil")))
	})

	It(
		"Should fail the connection when an incoming message exceeds the max length",
		func() {
			client, errs := serve(8)
			Expect(client.Send(transport.JSONRPCMessage{
				Content: `{"jsonrpc":"2.0","method":"initialized","params":{}}`,
			})).To(Succeed())
			Eventually(errs, 10*time.Second).Should(
				Receive(MatchError(ContainSubstring("content length"))))
		},
	)

	It("Should fail the connection on a malformed JSON payload", func() {
		client, errs := serve(transport.DefaultMaxContentLength)
		Expect(
			client.Send(transport.JSONRPCMessage{Content: "{not json"}),
		).To(Succeed())
		Eventually(errs, 10*time.Second).Should(
			Receive(MatchError(ContainSubstring("parse error"))))
	})

	It("Should fail the connection when a response exceeds the max length", func() {
		client, errs := serve(150)
		content := MustSucceed(json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]any{"clientInfo": map[string]any{"name": "t"}},
		}))
		Expect(
			client.Send(transport.JSONRPCMessage{Content: string(content)}),
		).To(Succeed())
		Eventually(errs, 10*time.Second).Should(
			Receive(MatchError(ContainSubstring("content length"))))
	})
})
