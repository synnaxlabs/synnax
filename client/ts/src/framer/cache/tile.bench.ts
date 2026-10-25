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
import { TileReader, tileSpan, type TileSpec } from "@/framer/cache/tile";
import { glTransform, SyntheticCore } from "@/testutil";

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
    transform: glTransform,
    batchDebounce: TimeSpan.ZERO,
    budget,
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

test("read", async ({ bench }) => {
  await bench.compare(
    bench("hit 3 tiles x 2 channels", await hit(1)),
    bench("hit 3 tiles x 11 channels", await hit(10)),
    bench("miss 3 tiles x 2 channels", miss(1)),
    bench("miss 3 tiles x 11 channels", miss(10)),
  );
});

// Every insert past the budget scans the tiles callers hold before it finds one to
// evict, so a held tile costs every later insert.
const insertPastBudget = async (held: number): Promise<() => Promise<void>> => {
  const core = createCore(1);
  const reader = createReader(core, Size.bytes(0));
  const base: TileSpec = {
    key: core.data[0].key,
    level: LEVEL,
    index: 0,
    pointLimit: POINT_LIMIT,
    aggregation: "min_max",
  };
  for (let i = 0; i < held; i++) (await reader.read({ ...base, index: i })).acquire();
  // Tiles past the data are empty, so the Core does no work for them.
  let index = Number(NOW.valueOf() / tileSpan(LEVEL).valueOf()) + 1;
  return async () => {
    await reader.read({ ...base, index: index++ });
  };
};

test("evict", async ({ bench }) => {
  await bench.compare(
    bench("insert past budget, 0 held", await insertPastBudget(0)),
    bench("insert past budget, 100 held", await insertPastBudget(100)),
    bench("insert past budget, 1000 held", await insertPastBudget(1000)),
  );
});

await allocSuiteAsync("tile read", [
  ["hit 3 tiles x 2 channels", await hit(1)],
  ["miss 3 tiles x 2 channels", miss(1)],
]);
