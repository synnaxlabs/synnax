// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, Frame, type framer } from "@synnaxlabs/client";
import { type MultiSeries, Series, sleep, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { Cache } from "@/telem/client/cache/cache";
import { MockRetriever } from "@/telem/client/reader.spec";
import { Streamer } from "@/telem/client/streamer";

class MockStreamer implements framer.Streamer {
  private keysI: channel.Params[];
  readonly updateVi = vi.fn();
  readonly closeVi = vi.fn();
  readonly iteratorVi = vi.fn();
  readonly nextVi = vi.fn();
  readonly reads?: framer.Frame[];
  readonly nextFn?: () => Promise<IteratorResult<framer.Frame>>;

  constructor(
    keys: channel.Keys,
    nextFn?: () => Promise<IteratorResult<framer.Frame>>,
    reads?: framer.Frame[],
  ) {
    this.keysI = [keys];
    this.reads = reads;
    this.nextFn = nextFn;
  }

  get keys(): channel.Keys {
    return this.keysI.at(-1) as channel.Keys;
  }

  update(keys: channel.Params): Promise<void> {
    this.keysI.push(keys);
    this.updateVi();
    return Promise.resolve();
  }

  close(): void {
    this.closeVi();
  }

  async next(): Promise<IteratorResult<framer.Frame>> {
    if (this.reads == null && this.nextFn == null)
      throw new Error("no next function or reads provided");
    if (this.nextFn != null) return await this.nextFn();
    const fr = this.reads?.shift();
    this.nextVi(fr);
    if (fr == null) return { done: true, value: undefined };
    return { done: false, value: fr };
  }

  async read(): Promise<framer.Frame> {
    const res = await this.next();
    if (res.done) throw new Error("no more frames");
    return res.value;
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    this.iteratorVi();
    return this;
  }
}

const createStreamOpener =
  (streamers: MockStreamer[]): framer.StreamOpener =>
  async () => {
    const streamer = streamers.shift();
    if (streamer == null) throw new Error("no streamers left");
    return streamer;
  };

describe("Streamer", () => {
  describe("construction", () => {
    it("should correctly construct a new streamer that operates", async () => {
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: createStreamOpener([new MockStreamer([])]),
      });
      expect(streamer).toBeDefined();
    });
  });

  describe("basic operation", () => {
    it("should allow the caller to subscribe to changes from a channel", async () => {
      const retriever = new MockRetriever();
      let i = 0;
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: retriever }),
        openStreamer: createStreamOpener([
          new MockStreamer([1], async () => {
            await sleep.sleep(TimeSpan.milliseconds(5));
            i++;
            return {
              done: false,
              value: new Frame({
                1: new Series({
                  data: new Float32Array([1]),
                  alignment: BigInt(i),
                }),
              }),
            };
          }),
        ]),
      });

      const responses: Map<channel.Key, MultiSeries>[] = [];
      const disconnect = await streamer.stream((d) => responses.push(d), [1]);
      await expect.poll(() => responses.length > 5).toBe(true);
      disconnect();

      // We should only ever get data for that particular channel.
      expect(responses.filter((r) => r.get(1)?.series.length === 0)).toHaveLength(
        responses.length - 1,
      );
      // The first response should have no data, as it's just pulling initial relevant
      // values from the cache.
      expect(responses[0].get(1)?.series.length).toEqual(0);
      // We should only every has a single response that has data, as its the only
      // buffer we're allocating and subsequent calls just tell the handler to re-read
      // the buffer.
      expect(responses.filter((r) => r.get(1)?.series.length === 1)).toHaveLength(1);
    });
  });
});
