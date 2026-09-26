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
  /** Starts reading from a stalled server's connections. */
  resume: () => void;
  /** Resolves once a client sends a freighter close message. */
  closeReceived: () => Promise<void>;
  close: () => void;
}

interface SilentServerOptions {
  /**
   * Acknowledges the stream before going silent, so the caller gets a live stream to
   * work with. Left false, the upgrade itself is swallowed.
   */
  upgraded?: boolean;
  /** Stops reading once upgraded, so everything the client sends stays queued. */
  stalled?: boolean;
}

/** Finds a freighter close message in masked client frames under 64KiB. */
const createCloseDetector = (onClose: () => void) => {
  let buf = Buffer.alloc(0);
  return (chunk: Buffer): void => {
    buf = Buffer.concat([buf, chunk]);
    while (buf.length >= 2) {
      let length = buf[1] & 0x7f;
      let header = 2;
      if (length === 126) {
        if (buf.length < 4) return;
        length = buf.readUInt16BE(2);
        header = 4;
      }
      const start = header + 4;
      if (buf.length < start + length) return;
      if (length < 126) {
        const mask = buf.subarray(header, start);
        const payload = Buffer.from(buf.subarray(start, start + length));
        for (let i = 0; i < payload.length; i++) payload[i] ^= mask[i % 4];
        if (payload.toString().includes('"close"')) onClose();
      }
      buf = buf.subarray(start + length);
    }
  };
};

/**
 * Accepts connections and then says nothing, modeling a path that swallows every
 * message: the socket is neither refused nor closed, so no error event ever fires.
 */
const createSilentServer = async ({
  upgraded = false,
  stalled = false,
}: SilentServerOptions = {}): Promise<SilentServer> => {
  const held: Socket[] = [];
  let dropped = 0;
  let closed = false;
  const closeWaiters: Array<() => void> = [];
  const onClose = (): void => {
    closed = true;
    closeWaiters.splice(0).forEach((resolve) => resolve());
  };
  const server: Server = createServer((socket) => {
    held.push(socket);
    socket.on("close", () => dropped++);
    if (!upgraded) return;
    let req = "";
    const detect = createCloseDetector(onClose);
    socket.on("data", (data) => {
      if (req.includes(HEADER_END)) {
        detect(data);
        return;
      }
      req += data.toString();
      if (!req.includes(HEADER_END)) return;
      upgrade(socket, req);
      if (stalled) socket.pause();
    });
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const { port } = server.address() as { port: number };
  return {
    port,
    dropped: () => dropped,
    resume: () => held.forEach((s) => s.resume()),
    closeReceived: async () => {
      if (closed) return;
      await new Promise<void>((resolve) => closeWaiters.push(resolve));
    },
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
    silent = await createSilentServer({ upgraded: true });
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

describe("websocket close deadline with queued data", () => {
  let silent: SilentServer;

  beforeEach(async () => {
    silent = await createSilentServer({ upgraded: true, stalled: true });
  });
  afterEach(() => {
    vi.useRealTimers();
    silent.close();
  });

  it("should start the deadline only once the queued data is sent", async () => {
    const stream = await createClient(silent.port).stream("/test", messageZ, messageZ);
    vi.useFakeTimers();
    const receive = stream.receive().then(
      () => "resolved",
      (err: unknown) => err,
    );
    // Far more than the operating system buffers between two sockets, so most of it
    // stays queued in the client while the peer is stalled.
    const message = "x".repeat(60_000);
    for (let i = 0; i < 1_100; i++) stream.send({ message });
    stream.closeSend();
    await vi.advanceTimersByTimeAsync(TimeSpan.minutes(5).milliseconds);
    await expect(Promise.race([receive, Promise.resolve("pending")])).resolves.toBe(
      "pending",
    );
    silent.resume();
    await silent.closeReceived();
    await vi.advanceTimersByTimeAsync(TimeSpan.seconds(31).milliseconds);
    await expect(receive).resolves.toBeInstanceOf(Unreachable);
  });
});
