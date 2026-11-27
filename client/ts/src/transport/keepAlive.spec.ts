// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import { sleep, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  createProxiedTestClient,
  createTestClient,
  newIndexedPair,
  newVirtualChannel,
} from "@/testutil";

const TIMEOUT = TimeSpan.seconds(3);
const KEEP_ALIVE = { interval: TimeSpan.seconds(1), timeout: TIMEOUT };
const STALLED = (err: unknown): boolean =>
  err instanceof Error &&
  Unreachable.matches(err) &&
  err.message === `stream received no message for ${TIMEOUT.toString()}`;
const SPEC_TIMEOUT = 15_000;

const client = createTestClient();

const createStalledClient = async () => {
  const { proxy, client: proxied } = await createProxiedTestClient({
    streamKeepAlive: KEEP_ALIVE,
    onRetry: () => {},
  });
  // Every stream the client opens after this stalls.
  const stall = (): void => {
    expect(proxy.blackholeStreams()).toBeGreaterThan(0);
  };
  return { client: proxied, stall };
};

describe("stream keep-alive", () => {
  it(
    "should fail a commit on a stalled writer",
    async () => {
      const { client: proxied, stall } = await createStalledClient();
      const ch = await newVirtualChannel(client);
      const writer = await proxied.openWriter({
        start: TimeStamp.now(),
        channels: ch.key,
      });
      await writer.write(ch.key, [1]);
      stall();
      await expect(writer.commit()).rejects.toSatisfy(STALLED);
    },
    SPEC_TIMEOUT,
  );

  it(
    "should fail a seek on a stalled iterator",
    async () => {
      const { client: proxied, stall } = await createStalledClient();
      const channels = await newIndexedPair(client);
      const iter = await proxied.openIterator(TimeRange.MAX, channels);
      stall();
      await expect(iter.seekFirst()).rejects.toSatisfy(STALLED);
    },
    SPEC_TIMEOUT,
  );

  it(
    "should fail a read on a stalled streamer",
    async () => {
      const { client: proxied, stall } = await createStalledClient();
      const ch = await newVirtualChannel(client);
      const streamer = await proxied.openStreamer({
        channels: ch.key,
        keepAlive: TimeSpan.ZERO,
      });
      stall();
      await expect(streamer.read()).rejects.toSatisfy(STALLED);
      streamer.close();
    },
    SPEC_TIMEOUT,
  );

  it(
    "should fail a receive on a stalled Arc LSP stream",
    async () => {
      const { client: proxied, stall } = await createStalledClient();
      const stream = await proxied.arcs.openLSP();
      stall();
      await expect(stream.receive()).rejects.toSatisfy(STALLED);
    },
    SPEC_TIMEOUT,
  );

  it(
    "should keep an idle stream open while the Core sends keep-alives",
    async () => {
      const { client: proxied } = await createStalledClient();
      const ch = await newVirtualChannel(client);
      const streamer = await proxied.openStreamer({
        channels: ch.key,
        keepAlive: TimeSpan.ZERO,
      });
      const read = streamer.read();
      await sleep.sleep(TIMEOUT.mult(1.5));
      await using writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: ch.key,
      });
      await writer.write(ch.key, new Float64Array([1]));
      expect(Array.from((await read).get(ch.key))).toEqual([1]);
      streamer.close();
    },
    SPEC_TIMEOUT,
  );
});
