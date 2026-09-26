// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHash } from "node:crypto";
import { createServer, type Socket } from "node:net";

import { LOW_PERF_SPECIAL_CHAR } from "@/framer/codec";

const WS_GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";
const HEADER_END = "\r\n\r\n";
const OPEN_FRAME_PAYLOAD = Buffer.concat([
  Buffer.from([LOW_PERF_SPECIAL_CHAR]),
  Buffer.from(JSON.stringify({ type: "open", payload: null })),
]);

export interface SilentPeer {
  /** Port the peer accepts connections on. */
  port: number;
  /** Resolves once a client sends a freighter message of the given type. */
  received: (type: string) => Promise<void>;
  /** Shuts the peer down, destroying every held connection. */
  close: () => Promise<void>;
}

/** Encodes payload as one unmasked binary frame. Payloads stay under 126 bytes. */
const encodeFrame = (payload: Buffer): Buffer =>
  Buffer.concat([Buffer.from([0x82, payload.length]), payload]);

/**
 * Decodes the complete masked client frames at the head of buf.
 * @returns the frame payloads and the bytes left over from a partial frame.
 */
const decodeFrames = (buf: Buffer): { payloads: Buffer[]; rest: Buffer } => {
  const payloads: Buffer[] = [];
  let offset = 0;
  while (buf.length - offset >= 2) {
    let length = buf[offset + 1] & 0x7f;
    let header = 2;
    if (length === 126) {
      if (buf.length - offset < 4) break;
      length = buf.readUInt16BE(offset + 2);
      header = 4;
    }
    const start = offset + header + 4;
    if (buf.length < start + length) break;
    const mask = buf.subarray(offset + header, start);
    const payload = Buffer.from(buf.subarray(start, start + length));
    for (let i = 0; i < payload.length; i++) payload[i] ^= mask[i % 4];
    payloads.push(payload);
    offset = start + length;
  }
  return { payloads, rest: buf.subarray(offset) };
};

/**
 * Creates a peer that completes the WebSocket upgrade and the freighter open
 * handshake, then never sends again. It models a Core that stops answering after
 * accepting a stream: no close or reset ever reaches the client.
 */
export const createSilentPeer = async (): Promise<SilentPeer> => {
  const held = new Set<Socket>();
  const seen = new Set<string>();
  const waiters = new Map<string, Array<() => void>>();
  const record = (type: string): void => {
    seen.add(type);
    waiters.get(type)?.forEach((resolve) => resolve());
    waiters.delete(type);
  };
  const server = createServer((socket) => {
    held.add(socket);
    socket.on("close", () => held.delete(socket));
    socket.on("error", () => {});
    let header = "";
    let body: Buffer = Buffer.alloc(0);
    socket.on("data", (chunk: Buffer) => {
      if (!header.includes(HEADER_END)) {
        header += chunk.toString("latin1");
        if (!header.includes(HEADER_END)) return;
        const key = /sec-websocket-key: (.+)/i.exec(header)?.[1].trim() ?? "";
        const accept = createHash("sha1")
          .update(key + WS_GUID)
          .digest("base64");
        socket.write(
          "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n" +
            `Connection: Upgrade\r\nSec-WebSocket-Accept: ${accept}${HEADER_END}`,
        );
        socket.write(encodeFrame(OPEN_FRAME_PAYLOAD));
        return;
      }
      const { payloads, rest } = decodeFrames(Buffer.concat([body, chunk]));
      body = rest;
      for (const payload of payloads) {
        const { type } = JSON.parse(payload.toString()) as { type: string };
        record(type);
      }
    });
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  if (address == null || typeof address === "string")
    throw new Error("silent peer failed to bind a port");
  return {
    port: address.port,
    received: async (type) => {
      if (seen.has(type)) return;
      await new Promise<void>((resolve) => {
        const pending = waiters.get(type) ?? [];
        pending.push(resolve);
        waiters.set(type, pending);
      });
    },
    close: async () => {
      held.forEach((socket) => socket.destroy());
      await new Promise<void>((resolve) => server.close(() => resolve()));
    },
  };
};
