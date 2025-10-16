// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { jsonRPC } from "@/jsonrpc";

describe("JSON-RPC", () => {
  describe("streamDecodeChunks", () => {
    it("should decode a single message", () => {
      const messages: jsonRPC.Message[] = [];
      const decoder = jsonRPC.streamDecodeChunks((msg) => messages.push(msg));

      const input = `Content-Length: 138

{"jsonrpc":"2.0","method":"$/status/report","params":{"text":"😺Lua","tooltip":"Workspace   : \\nCached files: 0/0\\nMemory usage: 1M\\n"}}`;

      decoder(new TextEncoder().encode(input));

      expect(messages).toHaveLength(1);
      expect(messages[0]).toEqual({
        jsonrpc: "2.0",
        method: "$/status/report",
        params: {
          text: "😺Lua",
          tooltip: "Workspace   : \nCached files: 0/0\nMemory usage: 1M\n",
        },
      });
    });

    it("should handle multiple messages in a single chunk", () => {
      const messages: jsonRPC.Message[] = [];
      const decoder = jsonRPC.streamDecodeChunks((msg) => messages.push(msg));

      const input = `Content-Length: 42

{"jsonrpc":"2.0","method":"$/status/show"}Content-Length: 82

{"id":5,"jsonrpc":"2.0","method":"workspace/semanticTokens/refresh","params":null}`;

      decoder(new TextEncoder().encode(input));

      expect(messages).toHaveLength(2);
      expect(messages[0]).toEqual({
        jsonrpc: "2.0",
        method: "$/status/show",
      });
      expect(messages[1]).toEqual({
        id: 5,
        jsonrpc: "2.0",
        method: "workspace/semanticTokens/refresh",
        params: null,
      });
    });

    it("should handle messages split across multiple chunks", () => {
      const messages: jsonRPC.Message[] = [];
      const decoder = jsonRPC.streamDecodeChunks((msg) => messages.push(msg));

      const chunk1 = `Content-Length: 42

{"jsonrpc":"2.0",`;

      const chunk2 = `"method":"$/status/show"}`;

      decoder(new TextEncoder().encode(chunk1));
      decoder(new TextEncoder().encode(chunk2));

      expect(messages).toHaveLength(1);
      expect(messages[0]).toEqual({
        jsonrpc: "2.0",
        method: "$/status/show",
      });
    });

    it("should handle string input", () => {
      const messages: jsonRPC.Message[] = [];
      const decoder = jsonRPC.streamDecodeChunks((msg) => messages.push(msg));

      const input = `Content-Length: 42

{"jsonrpc":"2.0","method":"$/status/show"}`;

      decoder(input); // Pass string directly

      expect(messages).toHaveLength(1);
      expect(messages[0]).toEqual({
        jsonrpc: "2.0",
        method: "$/status/show",
      });
    });

    it("should handle mixed string and Uint8Array chunks", () => {
      const messages: jsonRPC.Message[] = [];
      const decoder = jsonRPC.streamDecodeChunks((msg) => messages.push(msg));

      const chunk1 = `Content-Length: 42

{"jsonrpc":"2.0",`;

      const chunk2 = new TextEncoder().encode(`"method":"$/status/show"}`);

      decoder(chunk1); // String chunk
      decoder(chunk2); // Uint8Array chunk

      expect(messages).toHaveLength(1);
      expect(messages[0]).toEqual({
        jsonrpc: "2.0",
        method: "$/status/show",
      });
    });
  });
  describe("encodeMessage", () => {
    it("should encode a message", () => {
      const message: jsonRPC.Request = {
        jsonrpc: "2.0",
        id: 1,
        method: "$/status/show",
      };
      const messageStr = JSON.stringify(message);
      const encoded = jsonRPC.encodeMessage(message);
      expect(encoded.startsWith("Content-Length:")).toBe(true);
      const contentLength = encoded.match(/Content-Length: (\d+)/);
      expect(contentLength).not.toBeNull();
      expect(contentLength?.[1]).toBe(messageStr.length.toString());
      expect(encoded.endsWith(messageStr)).toBe(true);
    });
  });
});
