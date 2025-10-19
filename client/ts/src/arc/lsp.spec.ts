// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type jsonRPC } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { createTestClient } from "@/testutil/client";

describe("Arc LSP", () => {
  it("should open an LSP stream and handle initialize request", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

    // Send LSP initialize request (raw JSON, no Content-Length headers)
    const initializeRequest: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        processId: null,
        clientInfo: {
          name: "test-client",
          version: "1.0.0",
        },
        rootUri: null,
        capabilities: {},
      },
    };

    stream.send({
      content: JSON.stringify(initializeRequest),
    });

    // Receive response (raw JSON, no Content-Length headers)
    const [res, err] = await stream.receive();
    expect(err).toBeNull();
    if (!res) throw new Error("Expected response");

    // Parse raw JSON response directly
    const msg = JSON.parse(res.content) as jsonRPC.Response;
    expect(msg.jsonrpc).toBe("2.0");
    expect(msg.id).toBe(1);

    // Check for error response
    if ("error" in msg) throw new Error(`LSP error: ${msg.error.message}`);

    expect("result" in msg).toBe(true);

    // Verify capabilities are present
    if ("result" in msg) {
      const result = msg.result as Record<string, unknown>;
      expect(result).toHaveProperty("capabilities");
    }

    stream.closeSend();
  });

  it("should handle textDocument/didOpen notification", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

    // First initialize
    const initializeRequest: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        processId: null,
        clientInfo: { name: "test-client", version: "1.0.0" },
        rootUri: null,
        capabilities: {},
      },
    };

    stream.send({ content: JSON.stringify(initializeRequest) });
    const [initResponse, initErr] = await stream.receive();
    expect(initErr).toBeNull();
    expect(initResponse).not.toBeNull();
    if (!initResponse) throw new Error("Expected response");

    // Parse raw JSON response
    const initMsg = JSON.parse(initResponse.content) as jsonRPC.Response;
    expect(initMsg.id).toBe(1);

    // Check for error response
    if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

    // Send initialized notification
    const initializedNotification: jsonRPC.Request = {
      jsonrpc: "2.0",
      method: "initialized",
      params: {},
    };

    stream.send({ content: JSON.stringify(initializedNotification) });

    // Send didOpen notification
    const didOpenNotification: jsonRPC.Request = {
      jsonrpc: "2.0",
      method: "textDocument/didOpen",
      params: {
        textDocument: {
          uri: "file:///test.arc",
          languageId: "arc",
          version: 1,
          text: "let x = 1 + 2;",
        },
      },
    };

    stream.send({ content: JSON.stringify(didOpenNotification) });

    // Notifications don't get responses, but we might get diagnostics back
    // Wait a bit to see if we get any messages
    const diagnosticsPromise = stream.receive();
    const timeoutPromise = new Promise((resolve) =>
      setTimeout(() => resolve(null), 100),
    );

    const result = await Promise.race([diagnosticsPromise, timeoutPromise]);

    if (result && Array.isArray(result)) {
      const [diagResponse, diagErr] = result;
      if (diagResponse && !diagErr) {
        // Parse raw JSON notification
        const diagMsg = JSON.parse(diagResponse.content) as jsonRPC.Message;
        expect(diagMsg.jsonrpc).toBe("2.0");
        if ("method" in diagMsg) 
          expect(diagMsg.method).toBe("textDocument/publishDiagnostics");
        
      }
    }

    stream.closeSend();
    client.close();
  });

  it("should handle hover request", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

    // Initialize
    const initializeRequest: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        processId: null,
        clientInfo: { name: "test-client", version: "1.0.0" },
        rootUri: null,
        capabilities: { textDocument: { hover: { dynamicRegistration: true } } },
      },
    };

    stream.send({ content: JSON.stringify(initializeRequest) });
    const [initRes] = await stream.receive();
    expect(initRes).not.toBeNull();
    if (!initRes) throw new Error("Expected response");

    const initMsg = JSON.parse(initRes.content) as jsonRPC.Response;
    if ("error" in initMsg) 
      throw new Error(`LSP error: ${initMsg.error.message}`);
    

    // Send initialized notification
    stream.send({
      content: JSON.stringify({
        jsonrpc: "2.0",
        method: "initialized",
        params: {},
      }),
    });

    // Open a document
    stream.send({
      content: JSON.stringify({
        jsonrpc: "2.0",
        method: "textDocument/didOpen",
        params: {
          textDocument: {
            uri: "file:///test.arc",
            languageId: "arc",
            version: 1,
            text: "let x = 1 + 2;",
          },
        },
      }),
    });

    // Wait for and consume the publishDiagnostics notification
    const [diagResponse] = await stream.receive();
    expect(diagResponse).not.toBeNull();

    // Request hover information
    const hoverRequest: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 2,
      method: "textDocument/hover",
      params: {
        textDocument: { uri: "file:///test.arc" },
        position: { line: 0, character: 4 },
      },
    };

    stream.send({ content: JSON.stringify(hoverRequest) });

    const [hoverResponse, hoverErr] = await stream.receive();
    expect(hoverErr).toBeNull();
    expect(hoverResponse).not.toBeNull();
    if (!hoverResponse) throw new Error("Expected response");

    // Parse raw JSON response
    const responseMsg = JSON.parse(hoverResponse.content) as jsonRPC.Response;
    expect(responseMsg.jsonrpc).toBe("2.0");
    expect(responseMsg.id).toBe(2);

    // Check for error response
    if ("error" in responseMsg) 
      throw new Error(`LSP error: ${responseMsg.error.message}`);
    

    stream.closeSend();
    client.close();
  });

  it("should handle multiple concurrent messages", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

    const receivedMessages: jsonRPC.Response[] = [];

    // Initialize
    const initializeRequest: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        processId: null,
        clientInfo: { name: "test-client", version: "1.0.0" },
        rootUri: null,
        capabilities: {},
      },
    };

    stream.send({ content: JSON.stringify(initializeRequest) });
    const [initResponse] = await stream.receive();
    if (!initResponse) throw new Error("Expected response");
    const initMsg = JSON.parse(initResponse.content) as jsonRPC.Response;
    if ("error" in initMsg) 
      throw new Error(`LSP error: ${initMsg.error.message}`);
    
    receivedMessages.push(initMsg);

    // Send multiple requests with different IDs
    const requests = [
      {
        jsonrpc: "2.0",
        id: 10,
        method: "workspace/symbol",
        params: { query: "test" },
      },
      {
        jsonrpc: "2.0",
        id: 11,
        method: "textDocument/documentSymbol",
        params: { textDocument: { uri: "file:///test.arc" } },
      },
    ];

    for (const req of requests) stream.send({ content: JSON.stringify(req) });

    // Receive responses (order may vary)
    for (let i = 0; i < requests.length; i++) {
      const [response] = await stream.receive();
      if (!response) throw new Error("Expected response");
      const msg = JSON.parse(response.content) as jsonRPC.Response;
      if ("error" in msg) 
        throw new Error(`LSP error: ${msg.error.message}`);
      
      receivedMessages.push(msg);
    }

    // Should have init response + 2 request responses
    expect(receivedMessages.length).toBeGreaterThanOrEqual(3);

    // Verify all responses have correct structure
    for (const msg of receivedMessages) {
      expect(msg.jsonrpc).toBe("2.0");
      expect(msg.id).toBeDefined();
    }

    stream.closeSend();
    client.close();
  });

  it("should properly encode and decode JSON-RPC messages without headers", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

    // Test that our message format is just JSON without Content-Length headers
    const testMessage: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 999,
      method: "test/method",
      params: { data: "test" },
    };

    const messageContent = JSON.stringify(testMessage);

    // Send raw JSON (no headers)
    stream.send({ content: messageContent });

    // The server should respond (even if it's an error for unknown method)
    const [response, err] = await stream.receive();
    expect(err).toBeNull();
    expect(response).not.toBeNull();
    if (!response) throw new Error("Expected response");

    // Response should be parseable JSON
    const parsed = JSON.parse(response.content) as jsonRPC.Response;
    expect(parsed.jsonrpc).toBe("2.0");
    expect(parsed.id).toBe(999);

    // This test expects an error response for unknown method
    if ("error" in parsed) 
      expect(parsed.error).toBeDefined();
    

    stream.closeSend();
    client.close();
  });

  it("should provide semantic tokens for Arc code", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

    // Initialize with semantic tokens capability
    const initializeRequest: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        processId: null,
        clientInfo: { name: "test-client", version: "1.0.0" },
        rootUri: null,
        capabilities: {
          textDocument: {
            semanticTokens: {
              dynamicRegistration: true,
              requests: { full: true },
            },
          },
        },
      },
    };

    stream.send({ content: JSON.stringify(initializeRequest) });
    const [initRes] = await stream.receive();
    expect(initRes).not.toBeNull();
    if (!initRes) throw new Error("Expected response");

    const initMsg = JSON.parse(initRes.content) as jsonRPC.Response;
    if ("error" in initMsg) 
      throw new Error(`LSP error: ${initMsg.error.message}`);
    

    // Verify server advertises semantic tokens support
    if ("result" in initMsg) {
      const result = initMsg.result as Record<string, unknown>;
      const capabilities = result.capabilities as Record<string, unknown>;
      expect(capabilities).toHaveProperty("semanticTokensProvider");
    }

    // Send initialized notification
    stream.send({
      content: JSON.stringify({
        jsonrpc: "2.0",
        method: "initialized",
        params: {},
      }),
    });

    // Open a document with Arc code
    const arcCode = "func add(x i32, y i32) i32 {\n  return x + y\n}";
    stream.send({
      content: JSON.stringify({
        jsonrpc: "2.0",
        method: "textDocument/didOpen",
        params: {
          textDocument: {
            uri: "file:///test.arc",
            languageId: "arc",
            version: 1,
            text: arcCode,
          },
        },
      }),
    });

    // Wait for diagnostics
    await stream.receive();

    // Request semantic tokens
    const semanticTokensRequest: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 2,
      method: "textDocument/semanticTokens/full",
      params: {
        textDocument: { uri: "file:///test.arc" },
      },
    };

    stream.send({ content: JSON.stringify(semanticTokensRequest) });

    const [tokenResponse, tokenErr] = await stream.receive();
    expect(tokenErr).toBeNull();
    expect(tokenResponse).not.toBeNull();
    if (!tokenResponse) throw new Error("Expected response");

    const tokenMsg = JSON.parse(tokenResponse.content) as jsonRPC.Response;
    expect(tokenMsg.jsonrpc).toBe("2.0");
    expect(tokenMsg.id).toBe(2);

    if ("error" in tokenMsg) 
      throw new Error(`LSP error: ${tokenMsg.error.message}`);
    

    // Verify semantic tokens are returned
    if ("result" in tokenMsg) {
      const result = tokenMsg.result as Record<string, unknown>;
      expect(result).toHaveProperty("data");
      const data = result.data as number[];
      expect(Array.isArray(data)).toBe(true);
      // Should have tokens (encoded as [deltaLine, deltaStart, length, type, modifiers] * N)
      expect(data.length).toBeGreaterThan(0);
      // Tokens should be in groups of 5
      expect(data.length % 5).toBe(0);
    }

    stream.closeSend();
    client.close();
  });
});
