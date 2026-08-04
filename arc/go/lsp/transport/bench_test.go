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
	"testing"
	"time"

	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/lsp/transport"
	"github.com/synnaxlabs/arc/symbol"
	symtestutil "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/freighter/mock"
)

// benchProgram is a representative arc sequence exercising variables, channel
// reads/writes, and multiple stages.
const benchProgram = "sequence main {\n" +
	"    counter i64 := 0\n" +
	"    stage fill {\n" +
	"        press_ch -> counter\n" +
	"        counter -> log_ch\n" +
	"    }\n" +
	"    stage drain {\n" +
	"        0 -> valve_cmd\n" +
	"    }\n" +
	"}\n"

const benchURI = "file:///bench.arc"

func benchChannels() []symbol.Symbol {
	return []symbol.Symbol{
		{Name: "press_ch", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
		{Name: "log_ch", Kind: symbol.KindChannel, Type: types.Chan(types.I64())},
		{Name: "valve_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
	}
}

// benchHarness drives the LSP server over the freighter transport with raw JSON-RPC
// payloads, exactly as the TS client does. All benchmark traffic flows through the
// full stack: mock stream -> stream adapter -> jsonrpc2 -> protocol dispatch ->
// handler and back.
type benchHarness struct {
	b      *testing.B
	stream *mock.ClientStream[transport.JSONRPCMessage, transport.JSONRPCMessage]
	nextID int
}

func newHarness(b *testing.B) *benchHarness {
	b.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	server, err := lsp.New(lsp.Config{
		NewRoot: func() *symbol.Symbol {
			return symtestutil.NewRoot(nil, benchChannels()...)
		},
		// Keep debounced background analysis out of the measured path; DidSave and
		// DidOpen publish synchronously regardless.
		DebounceDelay:    time.Hour,
		MaxDebounceDelay: time.Hour,
	})
	if err != nil {
		b.Fatal(err)
	}
	clientStream, serverStream := mock.NewStreams[transport.JSONRPCMessage, transport.JSONRPCMessage](
		ctx,
		64,
		64,
	)
	errCh := make(chan error, 1)
	go func() {
		errCh <- transport.ServeFreighter(ctx, transport.Config{
			Server: server,
			Stream: serverStream,
		})
	}()
	b.Cleanup(func() {
		cancel()
		<-errCh
	})
	h := &benchHarness{b: b, stream: clientStream}
	h.request("initialize", map[string]any{
		"clientInfo": map[string]any{"name": "bench-client"},
	})
	h.notify("initialized", map[string]any{})
	return h
}

func (h *benchHarness) send(msg map[string]any) {
	h.b.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		h.b.Fatal(err)
	}
	if err := h.stream.Send(
		transport.JSONRPCMessage{Content: string(data)},
	); err != nil {
		h.b.Fatal(err)
	}
}

func (h *benchHarness) notify(method string, params any) {
	h.send(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

// pump reads messages until pred matches, transparently answering server-to-client
// requests (e.g. workspace/semanticTokens/refresh) so the server never blocks.
func (h *benchHarness) pump(pred func(msg map[string]any) bool) {
	h.b.Helper()
	for {
		raw, err := h.stream.Receive()
		if err != nil {
			h.b.Fatal(err)
		}
		var msg map[string]any
		if err := json.Unmarshal([]byte(raw.Content), &msg); err != nil {
			h.b.Fatal(err)
		}
		_, isCall := msg["method"]
		id, hasID := msg["id"]
		if isCall && hasID && id != nil {
			h.send(map[string]any{"jsonrpc": "2.0", "id": id, "result": nil})
			continue
		}
		if pred(msg) {
			return
		}
	}
}

// request sends a request and blocks until its response arrives.
func (h *benchHarness) request(method string, params any) {
	h.b.Helper()
	h.nextID++
	id := h.nextID
	h.send(
		map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params},
	)
	h.pump(func(msg map[string]any) bool {
		respID, ok := msg["id"].(float64)
		return ok && int(respID) == id
	})
}

func isPublishDiagnostics(msg map[string]any) bool {
	m, _ := msg["method"].(string)
	return m == "textDocument/publishDiagnostics"
}

// openDoc opens benchProgram and waits for the synchronous diagnostics publish.
func (h *benchHarness) openDoc() {
	h.b.Helper()
	h.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        benchURI,
			"languageId": "arc",
			"version":    1,
			"text":       benchProgram,
		},
	})
	h.pump(isPublishDiagnostics)
}

// BenchmarkInitialize measures a full request/response roundtrip through the
// transport with a trivial handler body.
func BenchmarkInitialize(b *testing.B) {
	h := newHarness(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.request("initialize", map[string]any{
			"clientInfo": map[string]any{"name": "bench-client"},
		})
	}
}

// BenchmarkAnalyzePublish measures a synchronous parse+analyze+publish cycle
// (didSave), the cost paid on every save and external republish.
func BenchmarkAnalyzePublish(b *testing.B) {
	h := newHarness(b)
	h.openDoc()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.notify("textDocument/didSave", map[string]any{
			"textDocument": map[string]any{"uri": benchURI},
		})
		h.pump(isPublishDiagnostics)
	}
}

// BenchmarkHover measures a hover request over a channel identifier.
func BenchmarkHover(b *testing.B) {
	h := newHarness(b)
	h.openDoc()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.request("textDocument/hover", map[string]any{
			"textDocument": map[string]any{"uri": benchURI},
			"position":     map[string]any{"line": 3, "character": 9},
		})
	}
}

// BenchmarkCompletion measures a completion request inside an identifier.
func BenchmarkCompletion(b *testing.B) {
	h := newHarness(b)
	h.openDoc()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.request("textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": benchURI},
			"position":     map[string]any{"line": 4, "character": 22},
		})
	}
}

// BenchmarkSemanticTokensFull measures a full semantic tokens computation.
func BenchmarkSemanticTokensFull(b *testing.B) {
	h := newHarness(b)
	h.openDoc()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.request("textDocument/semanticTokens/full", map[string]any{
			"textDocument": map[string]any{"uri": benchURI},
		})
	}
}

// BenchmarkDidChange measures incremental-change notification throughput: each
// iteration replaces one token with identical text, so content stays stable. A
// hover roundtrip at the end drains the pipeline; its cost amortizes across b.N.
func BenchmarkDidChange(b *testing.B) {
	h := newHarness(b)
	h.openDoc()
	version := 1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		version++
		h.notify("textDocument/didChange", map[string]any{
			"textDocument": map[string]any{"uri": benchURI, "version": version},
			"contentChanges": []map[string]any{{
				"range": map[string]any{
					"start": map[string]any{"line": 4, "character": 8},
					"end":   map[string]any{"line": 4, "character": 15},
				},
				"text": "counter",
			}},
		})
	}
	h.request("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": benchURI},
		"position":     map[string]any{"line": 3, "character": 9},
	})
}
