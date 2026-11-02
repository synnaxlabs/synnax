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

import { choose, indexes, level, pointLimit, type Position } from "@/framer/cache/grid";

// A line source runs each of these several times per frame.

const START = TimeStamp.seconds(1_750_000_000);
const HOUR = new TimeRange(START, START.add(TimeSpan.hours(1)));
const WEEK = new TimeRange(START, START.add(TimeSpan.days(7)));
const HOUR_LEVEL = level(HOUR);

test("grid", async ({ bench }) => {
  await bench.compare(
    bench("level of an hour", () => void level(HOUR)),
    bench("level of a week", () => void level(WEEK)),
    bench("indexes of an hour", () => void indexes(HOUR, HOUR_LEVEL)),
    bench(
      "point limit",
      () =>
        void pointLimit({
          level: HOUR_LEVEL,
          view: HOUR,
          points: 1000,
          drawn: "min_max",
          aggregation: "min_max",
        }),
    ),
  );
});

const SLOTS = indexes(HOUR, HOUR_LEVEL).map((index) => ({ level: HOUR_LEVEL, index }));

const positions = (l: number, from: number, count: number): Position[] =>
  Array.from({ length: count }, (_, i) => ({ level: l, index: from + i }));

// below returns every tile depth levels finer than the slots.
const below = (depth: number): Position[] =>
  positions(HOUR_LEVEL - depth, SLOTS[0].index * 2 ** depth, SLOTS.length * 2 ** depth);

// The held tiles of a view settled at its level, of a zoom in that waits on the new
// level, and of a zoom out that waits on it.
const OWN = positions(HOUR_LEVEL, SLOTS[0].index, SLOTS.length);
const COARSER = [{ level: HOUR_LEVEL + 2, index: Math.floor(SLOTS[0].index / 4) }];
const FINER = below(4);

test("choose", async ({ bench }) => {
  await bench.compare(
    bench("own tiles held", () => void choose(SLOTS, OWN)),
    bench("a coarser tile held", () => void choose(SLOTS, COARSER)),
    bench("tiles 4 levels finer held", () => void choose(SLOTS, FINER)),
  );
});
