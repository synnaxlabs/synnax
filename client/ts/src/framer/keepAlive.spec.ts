// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Pins silent-death detection on /frame/stream: the Core emits keepalive responses on
// request, and the client fails a silent read with Unreachable instead of hanging
// forever, which the hardened streamer turns into a reconnect.

import { Unreachable } from "@synnaxlabs/freighter";
import { sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { type channel } from "@/channel";
import { HardenedStreamer } from "@/framer/hardened";
import {
  createSeverableProxy,
  createTestClient,
  FAST_RETRY,
  newVirtualChannel,
} from "@/testutil";

const client = createTestClient();

// The Core's minimum accepted cadence, so the specs pin the fastest detection a
// client can actually get.
const KEEP_ALIVE = TimeSpan.seconds(2);
// KEEP_ALIVE_DEADLINE_FACTOR x KEEP_ALIVE: how long a read may stay silent once armed.
const DEADLINE = TimeSpan.seconds(6);
// One keepalive interval plus slack, so the client has seen a keepalive and armed.
const ARMED = TimeSpan.milliseconds(2500);
// Covers a full deadline trip plus the reconnect that follows it.
const POLL = { timeout: DEADLINE.milliseconds * 2 };

const write = async (ch: channel.Channel, values: number[]): Promise<void> => {
  const writer = await client.openWriter({ start: TimeStamp.now(), channels: ch.key });
  try {
    await writer.write(ch.key, new Float64Array(values));
  } finally {
    await writer.close();
  }
};

describe("streamer keepalive", () => {
  it("should keep keepalives out of the frames a streamer serves", async () => {
    const ch = await newVirtualChannel(client);
    const streamer = await client.openStreamer({
      channels: ch.key,
      keepAlive: KEEP_ALIVE,
    });
    try {
      // Let several keepalives queue up so the read has to skip past them.
      await sleep.sleep(KEEP_ALIVE.mult(2.5));
      await write(ch, [1, 2, 3]);
      const frame = await streamer.read();
      expect(Array.from(frame.get(ch.key))).toEqual([1, 2, 3]);
    } finally {
      streamer.close();
    }
  });

  it("should reject an interval below the Core's minimum", async () => {
    const ch = await newVirtualChannel(client);
    await expect(
      client.openStreamer({ channels: ch.key, keepAlive: TimeSpan.seconds(1) }),
    ).rejects.toThrow("keep_alive: must be greater than or equal to 2s");
  });

  it("should reject a silent read with Unreachable after the deadline", async () => {
    const proxy = await createSeverableProxy();
    try {
      const proxied = createTestClient({ port: proxy.port });
      const ch = await newVirtualChannel(client);
      const streamer = await proxied.openStreamer({
        channels: ch.key,
        keepAlive: KEEP_ALIVE,
      });
      // Receive at least one keepalive so the deadline is armed.
      await sleep.sleep(ARMED);
      expect(proxy.blackholeStreams()).toBeGreaterThan(0);
      const started = performance.now();
      await expect(streamer.read()).rejects.toSatisfy(
        (exc) =>
          Unreachable.matches(exc) &&
          exc.message === `streamer received no response for ${DEADLINE.toString()}`,
      );
      // The deadline must actually elapse: an instant rejection would mean the deadline
      // armed wrong, not that silence was detected.
      expect(performance.now() - started).toBeGreaterThanOrEqual(
        DEADLINE.milliseconds - 50,
      );
      streamer.close();
    } finally {
      await proxy.close();
    }
  }, 30_000);

  it("should reconnect and resume streaming after a silent death", async () => {
    const proxy = await createSeverableProxy();
    try {
      const proxied = createTestClient({ port: proxy.port });
      const ch = await newVirtualChannel(client);
      const onDrop = vi.fn();
      const onReopen = vi.fn();
      const hardened = await HardenedStreamer.open(
        async (cfg) => await proxied.openStreamer(cfg),
        { channels: ch.key, keepAlive: KEEP_ALIVE },
        FAST_RETRY,
        onReopen,
        onDrop,
      );
      try {
        await write(ch, [1]);
        expect(Array.from((await hardened.read()).get(ch.key))).toEqual([1]);
        await sleep.sleep(ARMED);
        expect(proxy.blackholeStreams()).toBeGreaterThan(0);
        // The proxy still forwards new connections, so the deadline trip inside this
        // read reconnects and the read stays pending for the next frame.
        const pending = hardened.read();
        await expect.poll(() => onDrop.mock.calls.length, POLL).toBe(1);
        expect(Unreachable.matches(onDrop.mock.calls[0][0])).toBe(true);
        await expect.poll(() => onReopen.mock.calls.length, POLL).toBe(1);
        await write(ch, [2]);
        expect(Array.from((await pending).get(ch.key))).toEqual([2]);
      } finally {
        hardened.close();
      }
    } finally {
      await proxy.close();
    }
  }, 30_000);

  it("should leave a silent read pending when keepalive is disabled", async () => {
    const proxy = await createSeverableProxy();
    try {
      const proxied = createTestClient({ port: proxy.port });
      const ch = await newVirtualChannel(client);
      const streamer = await proxied.openStreamer({
        channels: ch.key,
        keepAlive: TimeSpan.ZERO,
      });
      await sleep.sleep(TimeSpan.milliseconds(250));
      expect(proxy.blackholeStreams()).toBeGreaterThan(0);
      // Without keepalives the deadline never arms, which is also how a client behaves
      // against a Core that predates them: the read hangs, as before.
      const pending = streamer.read();
      pending.catch(() => {});
      const result = await Promise.race([
        pending.then(() => "settled"),
        sleep.sleep(TimeSpan.seconds(1)).then(() => "pending"),
      ]);
      expect(result).toEqual("pending");
      streamer.close();
    } finally {
      await proxy.close();
    }
  });
});
