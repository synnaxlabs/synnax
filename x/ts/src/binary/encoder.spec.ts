// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { binary } from "@/binary";

const sampleSchema = z.object({
  channelKey: z.string(),
  timeStamp: z.number(),
  value: z.unknown(),
});

binary.ENCODERS.forEach((e) => {
  describe(`encoder ${e.contentType}`, () => {
    it("should correctly encode and decode items", () => {
      const sample = {
        channelKey: "test",
        timeStamp: 123,
        value: [1, 2, 3],
      };
      const encoded = e.encode(sample);
      expect(e.decode(encoded, sampleSchema)).toEqual(sample);
    });
  });
});

describe("JSON", () => {
  it("should correctly convert keys to snake case", () => {
    const sample = {
      channelKey: "test",
      timeStamp: 123,
      value: new Array([1, 2, 3]),
    };
    const encoded = binary.JSON_CODEC.encodeString(sample);
    const parse = JSON.parse(encoded);
    expect(parse.channel_key).toEqual("test");
  });
  it("should correctly decode keys from snake case", () => {
    const sample = {
      channel_key: "test",
      time_stamp: 123,
      value: new Array([1, 2, 3]),
    };
    const encoded = JSON.stringify(sample);
    const decoded = binary.JSON_CODEC.decodeString(encoded, sampleSchema);
    expect(decoded.channelKey).toEqual("test");
  });

  describe("CSVCodec", () => {
    it("should correctly decode CSV data with valid input", () => {
      const sample = `
      channelKey,timeStamp,value
      test,123,5
      test2,124,6
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        channelKey: ["test", "test2"],
        timeStamp: [123, 124],
        value: [5, 6],
      });
    });

    it("should handle empty CSV data", () => {
      const sample = `
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({});
    });

    it("should handle CSV with only headers", () => {
      const sample = `
      channelKey,timeStamp,value
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        channelKey: [],
        timeStamp: [],
        value: [],
      });
    });

    it("should handle CSV with missing values", () => {
      const sample = `
      channelKey,timeStamp,value
      test,123,
      test2,124,6
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        channelKey: ["test", "test2"],
        timeStamp: [123, 124],
        value: [6],
      });
    });

    it("should handle CSV with extra values", () => {
      const sample = `
      channelKey,timeStamp,value
      test,123,5,extra
      test2,124,6
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        channelKey: ["test", "test2"],
        timeStamp: [123, 124],
        value: [5, 6],
      });
    });

    it("should handle CSV with different types of values", () => {
      const sample = `
      key,number,string
      test,123,"hello"
      test2,456,"world"
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        key: ["test", "test2"],
        number: [123, 456],
        string: ["hello", "world"],
      });
    });

    it("should handle CSV with spaces around values", () => {
      const sample = `
      key, number , string
      test , 123 , "hello"
      test2 , 456 , "world"
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        key: ["test", "test2"],
        number: [123, 456],
        string: ["hello", "world"],
      });
    });

    it("should handle CSV with empty rows", () => {
      const sample = `
      key,number,string
      test,123,"hello"
      ,
      test2,456,"world"
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        key: ["test", "test2"],
        number: [123, 456],
        string: ["hello", "world"],
      });
    });

    it("should handle CSV with single column", () => {
      const sample = `
      key
      test
      test2
      `;
      const decoded = binary.CSV_CODEC.decodeString(sample);
      expect(decoded).toEqual({
        key: ["test", "test2"],
      });
    });
  });
});

describe("JSON-RPC Stream Decoder", () => {
  it("should decode a single message", () => {
    const messages: binary.JSONRPCMessage[] = [];
    const decoder = binary.decodeJSONRPCStream((msg) => messages.push(msg));

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
    const messages: binary.JSONRPCMessage[] = [];
    const decoder = binary.decodeJSONRPCStream((msg) => messages.push(msg));

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
    const messages: binary.JSONRPCMessage[] = [];
    const decoder = binary.decodeJSONRPCStream((msg) => messages.push(msg));

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
});
