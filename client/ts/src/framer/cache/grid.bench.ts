// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { test } from "vitest";

import { choose, level, pointLimit, type Position, tiles } from "@/framer/cache/grid";

// The module runner serves each export through a getter. This object holds the
// functions themselves, so that cost stays out of the measurement.
const grid = { choose, tiles, level, pointLimit };

// A line source runs each of these several times per frame.

const START = TimeStamp.seconds(1_750_000_000);
const HOUR = new TimeRange(START, START.add(TimeSpan.hours(1)));
const WEEK = new TimeRange(START, START.add(TimeSpan.days(7)));
const HOUR_LEVEL = grid.level(HOUR);

test("grid", async ({ bench }) => {
  await bench.compare(
    bench("level of an hour", () => void grid.level(HOUR)),
    bench("level of a week", () => void grid.level(WEEK)),
    bench("tiles of an hour", () => void grid.tiles(HOUR, HOUR_LEVEL)),
    bench(
      "point limit",
      () =>
        void grid.pointLimit({
          level: HOUR_LEVEL,
          view: HOUR,
          points: 1000,
          drawn: "min_max",
          aggregation: "min_max",
        }),
    ),
  );
});

const SLOTS = grid.tiles(HOUR, HOUR_LEVEL).map((tile) => ({ level: HOUR_LEVEL, tile }));

const positions = (l: number, from: number, count: number): Position[] =>
  Array.from({ length: count }, (_, i) => ({ level: l, tile: from + i }));

// below returns every tile depth levels finer than the slots.
const below = (depth: number): Position[] =>
  positions(HOUR_LEVEL - depth, SLOTS[0].tile * 2 ** depth, SLOTS.length * 2 ** depth);

// The held tiles of a view settled at its level, of a zoom in that waits on the new
// level, and of a zoom out that waits on it.
const OWN = positions(HOUR_LEVEL, SLOTS[0].tile, SLOTS.length);
const COARSER = [{ level: HOUR_LEVEL + 2, tile: Math.floor(SLOTS[0].tile / 4) }];
const FINER = below(4);

test("choose", async ({ bench }) => {
  await bench.compare(
    bench("own tiles held", () => void grid.choose(SLOTS, OWN)),
    bench("a coarser tile held", () => void grid.choose(SLOTS, COARSER)),
    bench("tiles 4 levels finer held", () => void grid.choose(SLOTS, FINER)),
  );
});
