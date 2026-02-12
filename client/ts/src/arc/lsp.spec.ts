// Copyright 2026 Synnax Labs, Inc.
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

type LSPReceiver = {
  receive: () => Promise<[{ content: string }, null] | [null, Error]>;
};

const MAX_DRAIN = 50;

/** Drains messages from the stream until a JSON-RPC response with the expected id arrives. */
const receiveResponse = async (
  stream: LSPReceiver,
  expectedId: number,
): Promise<jsonRPC.Response> => {
  for (let i = 0; i < MAX_DRAIN; i++) {
    const [res, err] = await stream.receive();
    if (err != null) throw err;
    if (res == null) throw new Error("Expected response");
    const msg = JSON.parse(res.content);
    if (!("method" in msg) && "id" in msg && msg.id === expectedId)
      return msg as jsonRPC.Response;
  }
  throw new Error(
    `receiveResponse: drained ${MAX_DRAIN} messages without seeing id=${expectedId}`,
  );
};

/** Drains messages from the stream until a JSON-RPC notification with the expected method arrives. */
const receiveNotification = async (
  stream: LSPReceiver,
  expectedMethod: string,
): Promise<jsonRPC.Request> => {
  for (let i = 0; i < MAX_DRAIN; i++) {
    const [res, err] = await stream.receive();
    if (err != null) throw err;
    if (res == null) throw new Error("Expected message");
    const msg = JSON.parse(res.content);
    if ("method" in msg && msg.method === expectedMethod) return msg as jsonRPC.Request;
  }
  throw new Error(
    `receiveNotification: drained ${MAX_DRAIN} messages without seeing method=${expectedMethod}`,
  );
};

describe("Arc LSP", () => {
  it("should open an LSP stream and handle initialize request", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

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

    const msg = await receiveResponse(stream, 1);
    expect(msg.jsonrpc).toBe("2.0");
    expect(msg.id).toBe(1);
    if ("error" in msg) throw new Error(`LSP error: ${msg.error.message}`);
    expect("result" in msg).toBe(true);

    if ("result" in msg) {
      const result = msg.result as Record<string, unknown>;
      expect(result).toHaveProperty("capabilities");
    }

    stream.closeSend();
  });

  it("should handle textDocument/didOpen notification", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

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
    const initMsg = await receiveResponse(stream, 1);
    if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

    stream.send({
      content: JSON.stringify({
        jsonrpc: "2.0",
        method: "initialized",
        params: {},
      }),
    });

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

    const diagMsg = await receiveNotification(
      stream,
      "textDocument/publishDiagnostics",
    );
    expect(diagMsg.jsonrpc).toBe("2.0");

    stream.closeSend();
    client.close();
  });

  it("should handle hover request", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

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
    const initMsg = await receiveResponse(stream, 1);
    if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

    stream.send({
      content: JSON.stringify({
        jsonrpc: "2.0",
        method: "initialized",
        params: {},
      }),
    });

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

    await receiveNotification(stream, "textDocument/publishDiagnostics");

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

    const responseMsg = await receiveResponse(stream, 2);
    expect(responseMsg.jsonrpc).toBe("2.0");
    expect(responseMsg.id).toBe(2);
    if ("error" in responseMsg)
      throw new Error(`LSP error: ${responseMsg.error.message}`);

    stream.closeSend();
    client.close();
  });

  it("should handle multiple concurrent messages", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

    const receivedMessages: jsonRPC.Response[] = [];

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
    const initMsg = await receiveResponse(stream, 1);
    if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

    receivedMessages.push(initMsg);

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

    for (const req of requests) {
      const msg = await receiveResponse(stream, req.id);
      if ("error" in msg) throw new Error(`LSP error: ${msg.error.message}`);
      receivedMessages.push(msg);
    }

    expect(receivedMessages.length).toBeGreaterThanOrEqual(3);

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

    const testMessage: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 999,
      method: "test/method",
      params: { data: "test" },
    };

    stream.send({ content: JSON.stringify(testMessage) });

    const parsed = await receiveResponse(stream, 999);
    expect(parsed.jsonrpc).toBe("2.0");
    expect(parsed.id).toBe(999);

    if ("error" in parsed) expect(parsed.error).toBeDefined();

    stream.closeSend();
    client.close();
  });

  it("should provide semantic tokens for Arc code", async () => {
    const client = createTestClient();
    const stream = await client.arcs.openLSP();

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
    const initMsg = await receiveResponse(stream, 1);
    if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

    if ("result" in initMsg) {
      const result = initMsg.result as Record<string, unknown>;
      const capabilities = result.capabilities as Record<string, unknown>;
      expect(capabilities).toHaveProperty("semanticTokensProvider");
    }

    stream.send({
      content: JSON.stringify({
        jsonrpc: "2.0",
        method: "initialized",
        params: {},
      }),
    });

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

    await receiveNotification(stream, "textDocument/publishDiagnostics");

    const semanticTokensRequest: jsonRPC.Request = {
      jsonrpc: "2.0",
      id: 2,
      method: "textDocument/semanticTokens/full",
      params: {
        textDocument: { uri: "file:///test.arc" },
      },
    };

    stream.send({ content: JSON.stringify(semanticTokensRequest) });

    const tokenMsg = await receiveResponse(stream, 2);
    expect(tokenMsg.jsonrpc).toBe("2.0");
    expect(tokenMsg.id).toBe(2);
    if ("error" in tokenMsg) throw new Error(`LSP error: ${tokenMsg.error.message}`);

    if ("result" in tokenMsg) {
      const result = tokenMsg.result as Record<string, unknown>;
      expect(result).toHaveProperty("data");
      const data = result.data as number[];
      expect(Array.isArray(data)).toBe(true);
      expect(data.length).toBeGreaterThan(0);
      // Tokens are encoded as [deltaLine, deltaStart, length, type, modifiers] * N
      expect(data.length % 5).toBe(0);
    }

    stream.closeSend();
    client.close();
  });

  describe("Block Expression Wrapping", () => {
    it("should handle block URI with metadata", async () => {
      const client = createTestClient();
      const stream = await client.arcs.openLSP();

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
      const initMsg = await receiveResponse(stream, 1);
      if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "initialized",
          params: {},
        }),
      });

      const metadata = { is_block: true };
      const encoded = btoa(JSON.stringify(metadata));
      const blockURI = `arc://block/test123#${encoded}`;

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "textDocument/didOpen",
          params: {
            textDocument: {
              uri: blockURI,
              languageId: "arc",
              version: 1,
              text: "return x * 2",
            },
          },
        }),
      });

      const diagMsg = await receiveNotification(
        stream,
        "textDocument/publishDiagnostics",
      );
      expect(diagMsg.jsonrpc).toBe("2.0");
      if ("params" in diagMsg) {
        const params = diagMsg.params as Record<string, unknown>;
        expect(params.uri).toBe(blockURI);
        const diagnostics = params.diagnostics as unknown[];
        expect(Array.isArray(diagnostics)).toBe(true);
      }

      stream.closeSend();
      client.close();
    });

    it("should provide correct diagnostics for block with syntax error", async () => {
      const client = createTestClient();
      const stream = await client.arcs.openLSP();

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
      const initMsg = await receiveResponse(stream, 1);
      if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "initialized",
          params: {},
        }),
      });

      const metadata = { is_block: true };
      const encoded = btoa(JSON.stringify(metadata));
      const blockURI = `arc://block/syntax-error#${encoded}`;

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "textDocument/didOpen",
          params: {
            textDocument: {
              uri: blockURI,
              languageId: "arc",
              version: 1,
              text: "return x +",
            },
          },
        }),
      });

      const diagMsg = await receiveNotification(
        stream,
        "textDocument/publishDiagnostics",
      );
      if ("params" in diagMsg) {
        const params = diagMsg.params as Record<string, unknown>;
        const diagnostics = params.diagnostics as Array<{
          range: { start: { line: number; character: number } };
          message: string;
          severity: number;
        }>;

        expect(diagnostics.length).toBeGreaterThan(0);
        // Position should be mapped to line 0 for original expression
        expect(diagnostics[0].range.start.line).toBe(0);
      }

      stream.closeSend();
      client.close();
    });

    it("should handle multi-line block expressions", async () => {
      const client = createTestClient();
      const stream = await client.arcs.openLSP();

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
      const initMsg = await receiveResponse(stream, 1);
      if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "initialized",
          params: {},
        }),
      });

      const metadata = { is_block: true };
      const encoded = btoa(JSON.stringify(metadata));
      const blockURI = `arc://block/multiline#${encoded}`;

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "textDocument/didOpen",
          params: {
            textDocument: {
              uri: blockURI,
              languageId: "arc",
              version: 1,
              text: "let temp = x * 2\nlet result = temp + 1\nreturn result",
            },
          },
        }),
      });

      const diagMsg = await receiveNotification(
        stream,
        "textDocument/publishDiagnostics",
      );
      if ("params" in diagMsg) {
        const params = diagMsg.params as Record<string, unknown>;
        expect(params.uri).toBe(blockURI);
        const diagnostics = params.diagnostics as unknown[];
        expect(Array.isArray(diagnostics)).toBe(true);
      }

      stream.closeSend();
      client.close();
    });

    it("should handle textDocument/didChange for block expressions", async () => {
      const client = createTestClient();
      const stream = await client.arcs.openLSP();

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
      const initMsg = await receiveResponse(stream, 1);
      if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "initialized",
          params: {},
        }),
      });

      const metadata = { is_block: true };
      const encoded = btoa(JSON.stringify(metadata));
      const blockURI = `arc://block/change-test#${encoded}`;

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "textDocument/didOpen",
          params: {
            textDocument: {
              uri: blockURI,
              languageId: "arc",
              version: 1,
              text: "return x",
            },
          },
        }),
      });

      await receiveNotification(stream, "textDocument/publishDiagnostics");

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "textDocument/didChange",
          params: {
            textDocument: { uri: blockURI, version: 2 },
            contentChanges: [{ text: "return x + y" }],
          },
        }),
      });

      const changeDiagMsg = await receiveNotification(
        stream,
        "textDocument/publishDiagnostics",
      );
      if ("params" in changeDiagMsg) {
        const params = changeDiagMsg.params as Record<string, unknown>;
        expect(params.uri).toBe(blockURI);
      }

      stream.closeSend();
      client.close();
    });

    it("should reject non-block URIs without metadata", async () => {
      const client = createTestClient();
      const stream = await client.arcs.openLSP();

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
      const initMsg = await receiveResponse(stream, 1);
      if ("error" in initMsg) throw new Error(`LSP error: ${initMsg.error.message}`);

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "initialized",
          params: {},
        }),
      });

      // Block URI without metadata fragment - expression won't be wrapped
      const invalidBlockURI = "arc://block/no-metadata";

      stream.send({
        content: JSON.stringify({
          jsonrpc: "2.0",
          method: "textDocument/didOpen",
          params: {
            textDocument: {
              uri: invalidBlockURI,
              languageId: "arc",
              version: 1,
              text: "return x * 2",
            },
          },
        }),
      });

      const diagMsg = await receiveNotification(
        stream,
        "textDocument/publishDiagnostics",
      );
      if ("params" in diagMsg) {
        const params = diagMsg.params as Record<string, unknown>;
        const diagnostics = params.diagnostics as Array<{ severity: number }>;
        const errors = diagnostics.filter((d) => d.severity === 1);
        expect(errors.length).toBeGreaterThan(0);
      }

      stream.closeSend();
      client.close();
    });
  });
});
