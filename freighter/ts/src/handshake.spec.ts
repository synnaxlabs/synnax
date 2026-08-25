// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createServer, type Server, type Socket } from "node:net";

import { binary, TimeSpan, url } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { z } from "zod";

import { Unreachable } from "@/errors";
import { WebSocketClient } from "@/websocket";

const messageZ = z.object({ message: z.string().optional() });

/**
 * Accepts connections and then says nothing, modeling a path that swallows the
 * upgrade: the socket is neither refused nor closed, so no error event ever fires.
 */
const createSilentServer = async (): Promise<{ port: number; close: () => void }> => {
  const held: Socket[] = [];
  const server: Server = createServer((socket) => held.push(socket));
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const { port } = server.address() as { port: number };
  return {
    port,
    close: () => {
      held.forEach((s) => s.destroy());
      server.close();
    },
  };
};

describe("websocket handshake deadline", () => {
  let silent: { port: number; close: () => void };

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
