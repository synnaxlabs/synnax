// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Size, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { allocSuiteAsync } from "@synnaxlabs/x/bench";
import { test } from "vitest";

import { type channel } from "@/channel";
import { tileSpan } from "@/framer/cache/grid";
import { Store } from "@/framer/cache/store";
import { tileKey, TileReader, type TileSpec } from "@/framer/cache/tile";
import { SyntheticCore } from "@/testutil";

// One week of 1kHz data. A plot 1000 pixels wide that shows the last day of it draws
// level 26 tiles with a point limit of 1024.
const START = TimeStamp.seconds(1_750_000_000);
const NOW = START.add(TimeSpan.days(7));
const LEVEL = 26;
const POINT_LIMIT = 1024;

const createCore = (channels: number): SyntheticCore =>
  new SyntheticCore({ rate: 1000, start: START, now: NOW, channels });

const createReader = (core: SyntheticCore, budget?: Size): TileReader =>
  new TileReader({
    readRemote: async (tr, keys, reduction) => core.read(tr, keys, reduction),
    batchDebounce: TimeSpan.ZERO,
    store: new Store({ budget }),
  });

// specs returns the tiles of the last day for the index and every data channel.
const specs = (core: SyntheticCore, tiles = 3): TileSpec[] => {
  const last = Number(NOW.valueOf() / tileSpan(LEVEL).valueOf());
  const keys: channel.Key[] = [core.index.key, ...core.data.map((c) => c.key)];
  return Array.from({ length: tiles }, (_, i) => last - tiles + 1 + i).flatMap(
    (index) =>
      keys.map((key) => ({
        key,
        level: LEVEL,
        index,
        pointLimit: POINT_LIMIT,
        aggregation: "min_max" as const,
      })),
  );
};

const readAll = async (reader: TileReader, all: TileSpec[]): Promise<void> => {
  await Promise.all(all.map(async (spec) => await reader.read(spec)));
};

// A pan within a level reads the tiles the plot already holds.
const hit = async (channels: number): Promise<() => Promise<void>> => {
  const core = createCore(channels);
  const reader = createReader(core);
  const all = specs(core);
  await readAll(reader, all);
  return async () => await readAll(reader, all);
};

// A zoom to a new level reads every tile of the view from the Core, with each tile's
// channels in one request.
const miss = (channels: number): (() => Promise<void>) => {
  const core = createCore(channels);
  const all = specs(core);
  return async () => {
    const reader = createReader(core);
    await readAll(reader, all);
    reader.close();
  };
};

// A miss waits out one batch timer, which the baseline measures alone.
const batchTimer = async (): Promise<void> =>
  await new Promise((resolve) => setTimeout(resolve, 0));

test("read", async ({ bench }) => {
  await bench.compare(
    bench("baseline: one batch timer", batchTimer),
    bench("hit 3 tiles x 2 channels", await hit(1)),
    bench("hit 3 tiles x 11 channels", await hit(10)),
    bench("miss 3 tiles x 2 channels", miss(1)),
    bench("miss 3 tiles x 11 channels", miss(10)),
  );
});

// An insert past the budget scans the tiles callers hold before it finds one to evict,
// so each held tile costs every later insert. The held tiles fill the budget, so every
// insert evicts the tile inserted before it.
const INSERTS = 100;
// Tiles of 4 seconds, so thousands of them fit in the data.
const HELD_LEVEL = 12;
// Inserts cycle through this many tiles, which is more than the budget fits.
const CYCLE = 20_000;

const insertPastBudget = async (held: number): Promise<() => Promise<void>> => {
  const core = createCore(1);
  const spec = (index: number): TileSpec => ({
    key: core.data[0].key,
    level: HELD_LEVEL,
    index,
    pointLimit: POINT_LIMIT,
    aggregation: "min_max",
  });
  const first = Number(START.valueOf() / tileSpan(HELD_LEVEL).valueOf()) + 1;
  const heldSpecs = Array.from({ length: held }, (_, i) => spec(first + i));
  const sizer = createReader(core);
  const sizes = await Promise.all(
    heldSpecs.map(async (s) =>
      (await sizer.read(s)).series.reduce((sum, x) => sum + x.byteLength.valueOf(), 0),
    ),
  );
  sizer.close();
  let remoteReads = 0;
  // The budget fits the held tiles, with a tenth to spare for each tile's overhead.
  const store = new Store({
    budget: Size.bytes(1.1 * sizes.reduce((sum, b) => sum + b, 0)),
  });
  const reader = new TileReader({
    readRemote: async (tr, keys, reduction) => {
      remoteReads++;
      return core.read(tr, keys, reduction);
    },
    batchDebounce: TimeSpan.ZERO,
    store,
  });
  await readAll(reader, heldSpecs);
  heldSpecs.forEach((s) => store.get(tileKey(s))!.pins++);
  let n = 0;
  const insert = async (): Promise<void> => {
    const batch = Array.from({ length: INSERTS }, () =>
      spec(first + held + (n++ % CYCLE)),
    );
    await Promise.all(batch.map(async (s) => await reader.read(s)));
  };
  // Inserts past the spare budget reach the state where every insert evicts.
  for (let i = 0; i < Math.max(held, INSERTS); i += INSERTS) await insert();
  const readsFrom = async (s: TileSpec): Promise<number> => {
    const before = remoteReads;
    await reader.read(s);
    return remoteReads - before;
  };
  if (held > 0 && (await readsFrom(heldSpecs[held - 1])) > 0)
    throw new Error("a held tile left the cache");
  if ((await readsFrom(spec(first + held))) === 0)
    throw new Error("inserts did not evict");
  return insert;
};

test("evict", async ({ bench }) => {
  await bench.compare(
    bench("baseline: one batch timer", batchTimer),
    bench("100 inserts past budget, 0 held", await insertPastBudget(0)),
    bench("100 inserts past budget, 1000 held", await insertPastBudget(1000)),
    bench("100 inserts past budget, 4000 held", await insertPastBudget(4000)),
  );
});

await allocSuiteAsync("tile read", [
  ["hit 3 tiles x 2 channels", await hit(1)],
  ["miss 3 tiles x 2 channels", miss(1)],
]);
