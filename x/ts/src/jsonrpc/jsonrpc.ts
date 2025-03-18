// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary } from "@/binary";

export interface Message {
  jsonrpc: string;
  id?: number;
  method?: string;
  params?: unknown;
  result?: unknown;
}

export interface ChunkParser {
  (chunk: Uint8Array | ArrayBuffer | string): void;
}

/**
 * Decodes a a stream of utf-8 encoded chunks into JSON-RPC messages. This function
 * is particularly useful for communicating with JSON-RPC based services over stdio.
 *
 * @example
 *
 * const decoder = JSONRPC.decodeStream(console.log);
 *
 * decoder(`Content-Length: 42
 *
 *
 * {"jsonrpc":"2.0","method":"$/status/show"}Content-Length: 82
 *
 * {"id":5,"jsonrpc":"2.0","method":"workspace/semanticTokens/refresh","params":null}`)
 *
 * // Logs the following:
 *
 * { jsonrpc: '2.0', method: '$/status/show' }
 * { id: 5, jsonrpc: '2.0', method: 'workspace/semanticTokens/refresh', params: null }
 *
 * @param onMessage - The callback to call when a message is decoded.
 * @returns A function that accepts a chunk and decodes it into a message.
 */
export const streamDecodeChunks = (
  onMessage: (message: Message) => void,
): ChunkParser => {
  const decoder = new TextDecoder();
  let buffer = new Uint8Array(0);
  let expectedLength: number | null = null;

  return (chunk) => {
    let newChunk: Uint8Array;
    if (typeof chunk === "string") newChunk = new TextEncoder().encode(chunk);
    else if (chunk instanceof ArrayBuffer) newChunk = new Uint8Array(chunk);
    else newChunk = chunk;

    const newBuffer = new Uint8Array(buffer.length + newChunk.length);
    newBuffer.set(buffer);
    newBuffer.set(newChunk, buffer.length);
    buffer = newBuffer;
    while (buffer.length > 0) {
      if (expectedLength === null) {
        const headerStr = decoder.decode(buffer);
        const headerMatch = headerStr.match(/^Content-Length: (\d+)\r?\n\r?\n/);
        if (!headerMatch) break;

        expectedLength = parseInt(headerMatch[1]);
        const headerByteLength = new TextEncoder().encode(headerMatch[0]).length;
        buffer = buffer.slice(headerByteLength);
      }

      if (expectedLength !== null && buffer.length >= expectedLength) {
        const messageBytes = buffer.slice(0, expectedLength);
        buffer = buffer.slice(expectedLength);
        expectedLength = null;
        const messageStr = decoder.decode(messageBytes);
        const parsed = binary.JSON_CODEC.decodeString(messageStr);
        onMessage(parsed);
      } else break;
    }
  };
};

export const encodeMessage = (message: Message): string => {
  const messageStr = JSON.stringify(message);
  const contentLength = new TextEncoder().encode(messageStr).length;
  const header = `Content-Length: ${contentLength}\r\n\r\n`;
  return header + messageStr;
};
