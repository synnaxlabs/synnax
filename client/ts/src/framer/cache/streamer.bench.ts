// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { allocSuiteAsync } from "@synnaxlabs/x/bench";
import { test } from "vitest";

import { type channel } from "@/channel";
import { type framer } from "@/framer";
import { Joins } from "@/framer/cache/join";
import { Store } from "@/framer/cache/store";
import { MultiplexedStreamer } from "@/framer/cache/streamer";
import { createKeys, createSequence } from "@/framer/testutil";

class StubStreamer implements framer.Streamer {
  keys: channel.Key[];
  private i = 0;
  private pending: ((r: IteratorResult<framer.Frame>) => void) | null = null;

  constructor(
    private readonly frames: framer.Frame[],
    keys: channel.Key[],
    private readonly onDrained: () => void,
  ) {
    this.keys = keys;
  }

  async next(): Promise<IteratorResult<framer.Frame>> {
    if (this.i < this.frames.length)
      return { done: false, value: this.frames[this.i++] };
    this.onDrained();
    return await new Promise<IteratorResult<framer.Frame>>(
      (res) => (this.pending = res),
    );
  }

  async read(): Promise<framer.Frame> {
    const res = await this.next();
    if (res.done) throw new Error("drained");
    return res.value;
  }

  async update(): Promise<void> {}

  close(): void {
    this.pending?.({ done: true, value: undefined });
    this.pending = null;
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    return this;
  }
}

let _sink = 0;

// Drains a full frame sequence through the streamer into the store. Each run pays a
// fixed ~100ms demand-reconcile debounce on top of the ingest work, so use frame counts
// large enough for ingest to dominate.
const runIngest = async (
  frames: framer.Frame[],
  keys: channel.Key[],
  handlerCount: number,
): Promise<void> => {
  const store = new Store();
  let drained = (): void => undefined;
  const done = new Promise<void>((res) => (drained = res));
  const stub = new StubStreamer(frames, keys, () => drained());
  const streamer = new MultiplexedStreamer({
    store,
    joins: new Joins(store),
    onWait: () => {},
    openStreamer: async () => stub,
  });
  const perHandler = Math.ceil(keys.length / handlerCount);
  for (let i = 0; i < handlerCount; i++)
    streamer.demand(keys.slice(i * perHandler, (i + 1) * perHandler), (frame) => {
      _sink += frame.size;
    });
  await done;
  await streamer.close();
  store.close();
};

const BENCH_OPTS = { warmupIterations: 1, iterations: 3, time: 0 };

test("ingest", async ({ bench }) => {
  const narrowKeys = createKeys(100);
  const narrowFrames = createSequence(narrowKeys, 10, 500);
  const wideKeys = createKeys(1000);
  const wideFrames = createSequence(wideKeys, 1, 100);
  await bench.compare(
    bench(
      "500fr x 100ch x 10smp, 1 handler",
      async () => await runIngest(narrowFrames, narrowKeys, 1),
    ),
    bench(
      "500fr x 100ch x 10smp, 10 handlers",
      async () => await runIngest(narrowFrames, narrowKeys, 10),
    ),
    bench(
      "100fr x 1000ch x 1smp, 1 handler",
      async () => await runIngest(wideFrames, wideKeys, 1),
    ),
    BENCH_OPTS,
  );
});

{
  const keys = createKeys(100);
  const frames = createSequence(keys, 10, 100);
  await allocSuiteAsync(
    "streamer ingest 100fr x 100ch x 10smp",
    [["1 handler", async () => await runIngest(frames, keys, 1)]],
    8,
  );
}
