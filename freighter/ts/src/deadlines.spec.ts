// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createHash } from "node:crypto";
import { createServer, type Server, type Socket } from "node:net";

import { binary, TimeSpan, url } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Unreachable } from "@/errors";
import { WebSocketClient } from "@/websocket";

const messageZ = z.object({ message: z.string().optional() });

const WS_GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";
const HEADER_END = "\r\n\r\n";

/** Encodes payload as an unmasked binary frame. Payloads stay under 126 bytes. */
const frame = (payload: string): Buffer => {
  const body = Buffer.from(payload);
  return Buffer.concat([Buffer.from([0x82, body.length]), body]);
};

const upgrade = (socket: Socket, req: string): void => {
  const key = /sec-websocket-key: (.+)/i.exec(req)?.[1].trim() ?? "";
  const accept = createHash("sha1")
    .update(key + WS_GUID)
    .digest("base64");
  socket.write(
    "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n" +
      `Connection: Upgrade\r\nSec-WebSocket-Accept: ${accept}\r\n\r\n`,
  );
  socket.write(frame(JSON.stringify({ type: "open", payload: null })));
};

interface SilentServer {
  port: number;
  dropped: () => number;
  close: () => void;
}

/**
 * Accepts connections and then says nothing, modeling a path that swallows every
 * message: the socket is neither refused nor closed, so no error event ever fires.
 * @param upgraded - Acknowledges the stream before going silent, so the caller gets a
 * live stream to work with. Left false, the upgrade itself is swallowed.
 */
const createSilentServer = async (upgraded = false): Promise<SilentServer> => {
  const held: Socket[] = [];
  let dropped = 0;
  const server: Server = createServer((socket) => {
    held.push(socket);
    socket.on("close", () => dropped++);
    if (!upgraded) return;
    let req = "";
    socket.on("data", (data) => {
      if (req.includes(HEADER_END)) return;
      req += data.toString();
      if (req.includes(HEADER_END)) upgrade(socket, req);
    });
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const { port } = server.address() as { port: number };
  return {
    port,
    dropped: () => dropped,
    close: () => {
      held.forEach((s) => s.destroy());
      server.close();
    },
  };
};

const createClient = (port: number): WebSocketClient =>
  new WebSocketClient(new url.URL({ host: "127.0.0.1", port }), new binary.JSONCodec());

describe("websocket handshake deadline", () => {
  let silent: SilentServer;

  beforeEach(async () => {
    silent = await createSilentServer();
  });
  afterEach(() => silent.close());

  it("should reject a handshake that is never answered", async () => {
    const client = new WebSocketClient(
      new url.URL({ host: "127.0.0.1", port: silent.port }),
      new binary.JSONCodec(),
      false,
      TimeSpan.milliseconds(150),
    );
    const start = performance.now();
    await expect(client.stream("/test", messageZ, messageZ)).rejects.toThrow(
      Unreachable,
    );
    expect(performance.now() - start).toBeLessThan(2000);
    // The abandoned socket is released rather than left to connect unheld.
    await expect.poll(() => silent.dropped()).toBeGreaterThan(0);
  });

  it("should carry the deadline across withCodec", async () => {
    const client = new WebSocketClient(
      new url.URL({ host: "127.0.0.1", port: silent.port }),
      new binary.JSONCodec(),
      false,
      TimeSpan.milliseconds(150),
    ).withCodec(new binary.JSONCodec());
    await expect(client.stream("/test", messageZ, messageZ)).rejects.toThrow(
      Unreachable,
    );
  });
});

describe("websocket close deadline", () => {
  let silent: SilentServer;

  beforeEach(async () => {
    silent = await createSilentServer(true);
  });
  afterEach(() => {
    vi.useRealTimers();
    silent.close();
  });

  it("should fail a close the peer never acknowledges", async () => {
    const stream = await createClient(silent.port).stream("/test", messageZ, messageZ);
    vi.useFakeTimers();
    const received = expect(stream.receive()).rejects.toThrow(Unreachable);
    stream.closeSend();
    await vi.advanceTimersByTimeAsync(TimeSpan.minutes(1).milliseconds);
    await received;
  });

  it("should fail a receive issued after the deadline passes", async () => {
    const stream = await createClient(silent.port).stream("/test", messageZ, messageZ);
    vi.useFakeTimers();
    stream.closeSend();
    await vi.advanceTimersByTimeAsync(TimeSpan.minutes(1).milliseconds);
    await expect(stream.receive()).rejects.toThrow(Unreachable);
  });
});
