// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { allocSuiteAsync } from "@synnaxlabs/x/bench";
import { bench, describe } from "vitest";

import { type channel } from "@/channel";
import { createKeys, createSequence } from "@/framer/benchutil";
import { Cache } from "@/framer/cache/cache";
import { MultiplexedStreamer } from "@/framer/cache/streamer";
import { type Frame } from "@/framer/frame";
import { type Streamer } from "@/framer/streamer";

class StubStreamer implements Streamer {
  keys: channel.Key[];
  private i = 0;
  private pending: ((r: IteratorResult<Frame>) => void) | null = null;

  constructor(
    private readonly frames: Frame[],
    keys: channel.Key[],
    private readonly onDrained: () => void,
  ) {
    this.keys = keys;
  }

  async next(): Promise<IteratorResult<Frame>> {
    if (this.i < this.frames.length)
      return { done: false, value: this.frames[this.i++] };
    this.onDrained();
    return await new Promise<IteratorResult<Frame>>((res) => (this.pending = res));
  }

  async read(): Promise<Frame> {
    const res = await this.next();
    if (res.done) throw new Error("drained");
    return res.value;
  }

  async update(): Promise<void> {}

  close(): void {
    this.pending?.({ done: true, value: undefined });
    this.pending = null;
  }

  [Symbol.asyncIterator](): AsyncIterator<Frame> {
    return this;
  }
}

let _sink = 0;

// Drains a full frame sequence through the streamer into the cache. Each run pays
// a fixed ~100ms demand-reconcile debounce on top of the ingest work, so use frame
// counts large enough for ingest to dominate.
const runIngest = async (
  frames: Frame[],
  keys: channel.Key[],
  handlerCount: number,
): Promise<void> => {
  const cache = new Cache({ gcInterval: TimeSpan.hours(1) });
  let drained = (): void => undefined;
  const done = new Promise<void>((res) => (drained = res));
  const stub = new StubStreamer(frames, keys, () => drained());
  const streamer = new MultiplexedStreamer({ cache, openStreamer: async () => stub });
  const perHandler = Math.ceil(keys.length / handlerCount);
  for (let i = 0; i < handlerCount; i++)
    streamer.stream(
      (res) => {
        _sink += res.size;
      },
      keys.slice(i * perHandler, (i + 1) * perHandler),
    );
  await done;
  await streamer.close();
  cache.close();
};

const BENCH_OPTS = { warmupIterations: 1, iterations: 3, time: 0 };

describe("ingest", () => {
  {
    const keys = createKeys(100);
    const frames = createSequence(keys, 10, 500);
    bench(
      "500fr x 100ch x 10smp, 1 handler",
      async () => await runIngest(frames, keys, 1),
      BENCH_OPTS,
    );
    bench(
      "500fr x 100ch x 10smp, 10 handlers",
      async () => await runIngest(frames, keys, 10),
      BENCH_OPTS,
    );
  }
  {
    const keys = createKeys(1000);
    const frames = createSequence(keys, 1, 100);
    bench(
      "100fr x 1000ch x 1smp, 1 handler",
      async () => await runIngest(frames, keys, 1),
      BENCH_OPTS,
    );
  }
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
