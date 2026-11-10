// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import v8 from "node:v8";
import vm from "node:vm";

import { DataType, Rate, Series, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { test } from "vitest";

import { type channel } from "@/channel";
import { framer } from "@/framer";
import { Joins } from "@/framer/cache/join";
import { Latest, type LatestProps } from "@/framer/cache/latest";
import { Store } from "@/framer/cache/store";
import { MultiplexedStreamer } from "@/framer/cache/streamer";
import { SyntheticCore } from "@/testutil";

const START = TimeStamp.seconds(1_000_000);
const COUNTS = [1, 70_000];
const WRITES = 1000;
const BENCH_OPTS = { warmupIterations: 20, iterations: 200, time: 0 };

v8.setFlagsFromString("--expose-gc");
const gc = vm.runInNewContext("gc") as () => void;

// Wires latests to a synthetic Core as a feed does.
const createCore = () => {
  const core = new SyntheticCore({
    rate: Rate.hz(100),
    start: START,
    now: START.add(TimeSpan.minutes(10)),
    derived: true,
  });
  const store = new Store({ now: core.now });
  const streamer = new MultiplexedStreamer({
    store,
    joins: new Joins(store),
    onWait: () => {},
    openStreamer: async (config) => core.openStreamer(config),
    removalDelay: TimeSpan.ZERO,
  });
  const props: LatestProps = {
    store,
    streamer,
    retrieveChannels: async (keys) => await core.retrieveChannels(keys),
    now: core.now,
    readLatest: async (keys, count, tr) => core.readLatest(keys, count, tr),
  };
  return { core, props, close: async () => await streamer.close() };
};

const waitFor = async (ready: () => boolean): Promise<void> => {
  while (!ready()) await sleep.sleep(TimeSpan.milliseconds(1));
};

// Returns the nth frame of one-sample writes to the key, at alignments in a row.
const createWrites = (key: channel.Key, n: number): framer.Frame => {
  const writes = Array.from({ length: WRITES }, (_, i) => {
    const alignment = BigInt(n * WRITES + i);
    const stamp = START.add(TimeSpan.nanoseconds(alignment));
    return new Series({
      data: new Float32Array(1),
      dataType: DataType.FLOAT32,
      alignment,
      timeRange: stamp.spanRange(TimeSpan.NANOSECOND),
    });
  });
  return new framer.Frame(new Array(WRITES).fill(key), writes);
};

// Opens a latest on a channel with no index. run streams the frame that prepare builds
// and resolves once the latest takes it.
const openStream = async (count: number) => {
  const { core, props, close } = createCore();
  const key = core.virtual!.key;
  let taken: () => void = () => {};
  const latest = new Latest(props, key, undefined, count, () => taken());
  await waitFor(() => latest.value().status.variant === "success" && core.streams(key));
  let frame = new framer.Frame();
  let next = 0;
  const prepare = (): void => {
    frame = createWrites(key, next++);
  };
  const run = async (): Promise<void> => {
    const done = new Promise<void>((resolve) => (taken = resolve));
    core.stream(frame);
    await done;
  };
  const stop = async (): Promise<void> => {
    latest.close();
    await close();
  };
  return { prepare, run, stop };
};

test("stream", async ({ bench }) => {
  const streams = await Promise.all(COUNTS.map(openStream));
  await bench.compare(
    ...streams.map((s, i) =>
      bench(
        `${WRITES} writes, no index, count ${COUNTS[i]}`,
        { beforeEach: s.prepare },
        s.run,
      ),
    ),
    BENCH_OPTS,
  );
  await Promise.all(streams.map(async (s) => await s.stop()));
});

// Returns the bytes each open latest holds once loaded, with the typed array storage
// the heap leaves out.
const retained = async (open: () => Latest, n = 100): Promise<number> => {
  const used = (): number => {
    gc();
    gc();
    const { heapUsed, arrayBuffers } = process.memoryUsage();
    return heapUsed + arrayBuffers;
  };
  const before = used();
  const latests = Array.from({ length: n }, open);
  await waitFor(() => latests.every((l) => l.value().status.variant === "success"));
  const after = used();
  latests.forEach((l) => l.close());
  return Math.round((after - before) / n);
};

test("retained", async () => {
  const { core, props, close } = createCore();
  const rows: { case: string; "bytes/latest": number }[] = [];
  for (const count of COUNTS)
    for (const [name, key] of [
      ["indexed", core.data[0].key],
      ["no index", core.virtual!.key],
    ] as const)
      rows.push({
        case: `${name}, count ${count}`,
        "bytes/latest": await retained(
          () => new Latest(props, key, undefined, count, () => {}),
        ),
      });
  console.table(rows);
  await close();
});
