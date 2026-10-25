// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type breaker,
  Rate,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { AccessDeniedError, ValidationError } from "@/errors";
import * as grid from "@/framer/cache/grid";
import { Joins } from "@/framer/cache/join";
import { Joiner } from "@/framer/cache/joiner";
import { Points, type PointsProps } from "@/framer/cache/points";
import { Reader, type RemoteReader } from "@/framer/cache/reader";
import { Store } from "@/framer/cache/store";
import { MultiplexedStreamer } from "@/framer/cache/streamer";
import { tileKey, TileReader } from "@/framer/cache/tile";
import { type PointsLimit } from "@/framer/subscription";
import { SyntheticCore } from "@/testutil";

const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };
const DEBOUNCE = TimeSpan.milliseconds(5);
const BREAKER: breaker.Config = {
  baseInterval: TimeSpan.milliseconds(10),
  maxInterval: TimeSpan.milliseconds(50),
};
const RAW: PointsLimit = { variant: "points", count: Infinity, aggregation: "min_max" };
const TILED: PointsLimit = { variant: "points", count: 100, aggregation: "min_max" };

const createCore = (): SyntheticCore =>
  new SyntheticCore({
    rate: Rate.hz(100),
    start: START,
    now: START.add(TimeSpan.minutes(10)),
  });

// Wires points to the Core as a feed does. Each points pulls a value on every change,
// as a consumer does.
const createPoints = (
  core: SyntheticCore,
  readRemote: RemoteReader = async (tr, keys, reduction) =>
    core.read(tr, keys, reduction),
) => {
  const store = new Store({ now: core.now });
  const joins = new Joins(store);
  const { retrieveChannels, now } = core;
  const reader = new Reader({
    readRemote,
    retrieveChannels,
    store,
    batchDebounce: DEBOUNCE,
    now,
  });
  const tiles = new TileReader({ readRemote, store, batchDebounce: DEBOUNCE, now });
  const streamer = new MultiplexedStreamer({
    store,
    joins,
    onWait: (indexes) => joiner.wake(indexes),
    openStreamer: async (config) => core.openStreamer(config),
    removalDelay: TimeSpan.ZERO,
    breaker: BREAKER,
  });
  const joiner = new Joiner({ joins, reader, streamer, now, breaker: BREAKER });
  const props: PointsProps = {
    store,
    joins,
    reader,
    tiles,
    streamer,
    joiner,
    retrieveChannels,
    now,
    breaker: BREAKER,
  };
  const open = new Set<Points>();
  const subscribe = (key: channel.Key, window: Window, limit: PointsLimit): Points => {
    const points: Points = new Points(props, key, window, limit, () =>
      setTimeout(() => {
        if (open.has(points)) points.value();
      }, 0),
    );
    open.add(points);
    points.value();
    return points;
  };
  const unsubscribe = (points: Points): void => {
    open.delete(points);
    points.close();
  };
  const close = async (): Promise<void> => {
    open.forEach(unsubscribe);
    joiner.close();
    await streamer.close();
    await reader.close();
    tiles.close();
    joins.close();
    store.close();
  };
  return { store, subscribe, unsubscribe, close };
};

describe("Points", () => {
  it("should stop reading raw samples after a definitive failure", async () => {
    const core = createCore();
    let reads = 0;
    const { subscribe, close } = createPoints(core, async () => {
      reads++;
      throw new AccessDeniedError("no read access");
    });
    const window = staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(4))));
    const points = subscribe(core.data[0].key, window, RAW);
    try {
      await expect
        .poll(() => points.value().status, POLL)
        .toMatchObject({
          variant: "error",
          message: "Failed to read channel data",
          description: "You do not have permission to do that. no read access",
        });
      await sleep.sleep(TimeSpan.milliseconds(100));
      expect(reads).toBe(1);
    } finally {
      await close();
    }
  });

  it("should keep a rejected tile's error until the request changes", async () => {
    const core = createCore();
    let reduced = 0;
    const { store, subscribe, close } = createPoints(
      core,
      async (tr, keys, reduction) => {
        if (reduction != null && reduced++ === 0)
          throw new ValidationError("reduction rejected");
        return core.read(tr, keys, reduction);
      },
    );
    const key = core.data[0].key;
    const window = staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(8))));
    const points = subscribe(key, window, TILED);
    const failure = {
      variant: "error",
      message: "Failed to read channel data",
      description: "reduction rejected",
    };
    try {
      await expect.poll(() => points.value().status, POLL).toMatchObject(failure);
      // A history change reads the other tiles again, and the reads succeed.
      const before = reduced;
      store.invalidate(key);
      await expect.poll(() => reduced, POLL).toBeGreaterThan(before);
      await sleep.sleep(TimeSpan.milliseconds(50));
      expect(points.value().status).toMatchObject(failure);
      points.update(window, { ...TILED, count: 200 });
      await expect.poll(() => points.value().status.variant, POLL).toBe("success");
    } finally {
      await close();
    }
  });

  it("should never pin a cached tile for a cut tile at its position", async () => {
    const core = createCore();
    const { store, subscribe, unsubscribe, close } = createPoints(core);
    const key = core.data[0].key;
    const span = TimeSpan.minutes(4);
    const level = grid.level(new TimeRange(START, START.add(span)));
    const index = grid.index(START, level) + 1;
    const { start } = grid.tileRange({ level, index });
    const home = new TimeRange(start, start.add(span));
    const pointLimit = grid.pointLimit({
      level,
      view: home,
      points: TILED.count,
      drawn: TILED.aggregation,
      aggregation: TILED.aggregation,
    });
    const uncut = tileKey({
      key,
      level,
      index,
      pointLimit,
      aggregation: TILED.aggregation,
    });
    try {
      const whole = subscribe(key, staticWindow(home), TILED);
      await expect.poll(() => whole.value().status.variant, POLL).toBe("success");
      unsubscribe(whole);
      expect(store.get(uncut)?.pins).toBe(0);
      const shift = TimeSpan.seconds(10);
      const shifted = new TimeRange(home.start.add(shift), home.end.add(shift));
      const cut = subscribe(key, staticWindow(shifted), TILED);
      await expect.poll(() => cut.value().status.variant, POLL).toBe("success");
      expect(store.get(uncut)?.pins).toBe(0);
    } finally {
      await close();
    }
  });
});
