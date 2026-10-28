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
  dynamicWindow,
  Rate,
  type Series,
  Size,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { type framer } from "@/framer";
import { RAW_LEVEL, rawIndex } from "@/framer/cache/block";
import {
  index as tileIndex,
  level as tileLevel,
  pointLimit as tilePointLimit,
  tileRange,
  tileSpan,
} from "@/framer/cache/grid";
import { Joins } from "@/framer/cache/join";
import { Joiner } from "@/framer/cache/joiner";
import { Points, type PointsProps } from "@/framer/cache/points";
import { Reader, type RemoteReader } from "@/framer/cache/reader";
import { Store, type StoreProps } from "@/framer/cache/store";
import { MultiplexedStreamer } from "@/framer/cache/streamer";
import { tileKey, TileReader } from "@/framer/cache/tile";
import { AccessDeniedError, ValidationError } from "@/index";
import { SyntheticCore } from "@/testutil";

// More points than any window here holds samples, so every sample draws.
const EXACT = 2 ** 20;

const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };
const DEBOUNCE = TimeSpan.milliseconds(5);
const BREAKER: breaker.Config = {
  baseInterval: TimeSpan.milliseconds(10),
  maxInterval: TimeSpan.milliseconds(50),
};
const RAW: framer.PointsLimit = {
  variant: "points",
  count: EXACT,
  aggregation: "min_max",
};
const TILED: framer.PointsLimit = {
  variant: "points",
  count: 100,
  aggregation: "min_max",
};

// Returns true when the range is one whole tile of the grid.
const isTile = (tr: TimeRange): boolean => {
  const level = Math.log2(Number(tr.span.valueOf() / TimeSpan.MILLISECOND.valueOf()));
  return (
    Number.isInteger(level) &&
    tileRange({ level, index: tileIndex(tr.start, level) }).equals(tr)
  );
};

const leadingOf = (store: Store, key: channel.Key): Series | null =>
  store.open(key)?.buffer?.leadingBuffer ?? null;

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
  storeProps: StoreProps = {},
  pointsBreaker: breaker.Config = BREAKER,
) => {
  const store = new Store({ now: core.now, ...storeProps });
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
    breaker: pointsBreaker,
  };
  const open = new Set<Points>();
  const subscribe = (
    key: channel.Key,
    window: Window,
    limit: framer.PointsLimit,
  ): Points => {
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
  return { store, joins, subscribe, unsubscribe, close };
};

describe("Points", () => {
  describe("snapshot", () => {
    const window = dynamicWindow(TimeSpan.seconds(30));

    it("should return the same snapshot while nothing changes", async () => {
      const core = createCore();
      const { subscribe, close } = createPoints(core);
      const range = new TimeRange(START, START.add(TimeSpan.minutes(8)));
      const points = subscribe(core.data[0].key, staticWindow(range), TILED);
      try {
        await expect.poll(() => points.value().status.variant, POLL).toBe("success");
        expect(points.value()).toBe(points.value());
      } finally {
        await close();
      }
    });

    it("should return the same snapshot for a channel it cannot find", async () => {
      const core = createCore();
      const { subscribe, close } = createPoints(core);
      const points = subscribe(12_345, window, RAW);
      try {
        await expect.poll(() => points.value().status.variant, POLL).toBe("error");
        expect(points.value()).toBe(points.value());
      } finally {
        await close();
      }
    });

    it("should keep the snapshot while the view moves inside its tiles", async () => {
      const core = createCore();
      let tileReads = 0;
      const { subscribe, close } = createPoints(core, async (tr, keys, reduction) => {
        if (reduction != null) tileReads++;
        return core.read(tr, keys, reduction);
      });
      const range = new TimeRange(START, START.add(TimeSpan.minutes(8)));
      const span = TimeSpan.minutes(2);
      const level = tileLevel(new TimeRange(START, START.add(span)));
      const tile = tileRange({
        level,
        index: tileIndex(START.add(span.mult(2)), level),
      });
      const limitAt = (end: TimeStamp): framer.PointsLimit => ({
        ...TILED,
        view: new TimeRange(end.sub(span), end),
      });
      let end = tile.start.add(TimeSpan.seconds(10));
      const points = subscribe(core.data[0].key, staticWindow(range), limitAt(end));
      try {
        await expect.poll(() => points.value().status.variant, POLL).toBe("success");
        const before = points.value();
        const reads = tileReads;
        for (let i = 0; i < 10; i++) {
          end = end.add(TimeSpan.milliseconds(1));
          points.update(staticWindow(range), limitAt(end));
          expect(points.value()).toBe(before);
        }
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(tileReads).toBe(reads);
        points.update(staticWindow(range), limitAt(tile.end.add(TimeSpan.seconds(1))));
        expect(points.value()).not.toBe(before);
        await expect.poll(() => tileReads, POLL).toBeGreaterThan(reads);
      } finally {
        await close();
      }
    });

    it("should draw a settling view from held tiles and read none", async () => {
      const core = createCore();
      let tileReads = 0;
      const { subscribe, close } = createPoints(core, async (tr, keys, reduction) => {
        if (reduction != null) tileReads++;
        return core.read(tr, keys, reduction);
      });
      const window = staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(8))));
      const points = subscribe(core.data[0].key, window, TILED);
      try {
        await expect.poll(() => points.value().status.variant, POLL).toBe("success");
        const reads = tileReads;
        const view = new TimeRange(
          START.add(TimeSpan.minutes(2)),
          START.add(TimeSpan.minutes(3)),
        );
        points.update(window, { ...TILED, view, settling: true });
        expect(points.value().series.length).toBeGreaterThan(0);
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(tileReads).toBe(reads);
        points.update(window, { ...TILED, view });
        points.value();
        await expect.poll(() => tileReads, POLL).toBeGreaterThan(reads);
      } finally {
        await close();
      }
    });

    it("should change the series when a tile lands", async () => {
      const core = createCore();
      const { subscribe, close } = createPoints(core);
      const range = new TimeRange(START, START.add(TimeSpan.minutes(8)));
      const points = subscribe(core.data[0].key, staticWindow(range), TILED);
      try {
        await expect.poll(() => points.value().status.variant, POLL).toBe("success");
        expect(points.value().series.length).toBeGreaterThan(0);
      } finally {
        await close();
      }
    });

    it("should keep the series and move the time as the stream grows", async () => {
      const core = createCore();
      const { store, joins, subscribe, close } = createPoints(core);
      const key = core.index.key;
      const points = subscribe(key, window, RAW);
      const servesBuffer = () =>
        points.value().series.series.at(-1) === leadingOf(store, key);
      try {
        await expect.poll(() => joins.joined(key), POLL).toBe(true);
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(servesBuffer, POLL).toBe(true);
        const before = points.value();
        const { length } = before.series;
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => points.value().series.length, POLL).toBe(length + 10);
        const after = points.value();
        expect(after).not.toBe(before);
        expect(after.series).toBe(before.series);
        expect(after.time?.valueOf()).toBe(
          before.time!.add(TimeSpan.milliseconds(100)).valueOf(),
        );
      } finally {
        await close();
      }
    });

    it("should serve the new buffer when the stream fills one", async () => {
      const core = createCore();
      const { store, joins, subscribe, close } = createPoints(core, undefined, {
        dynamicBufferSize: 20,
      });
      const key = core.index.key;
      const points = subscribe(key, window, RAW);
      const servesBuffer = () =>
        points.value().series.series.at(-1) === leadingOf(store, key);
      try {
        await expect.poll(() => joins.joined(key), POLL).toBe(true);
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(servesBuffer, POLL).toBe(true);
        const filled = leadingOf(store, key);
        core.advance(TimeSpan.milliseconds(300));
        await expect.poll(() => leadingOf(store, key) !== filled, POLL).toBe(true);
        expect(servesBuffer()).toBe(true);
      } finally {
        await close();
      }
    });
    it("should serve the new block when the stream crosses a block edge", async () => {
      const core = createCore();
      const { store, joins, subscribe, close } = createPoints(core);
      const key = core.index.key;
      const points = subscribe(key, window, RAW);
      const servesBuffer = () =>
        points.value().series.series.at(-1) === leadingOf(store, key);
      try {
        await expect.poll(() => joins.joined(key), POLL).toBe(true);
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(servesBuffer, POLL).toBe(true);
        const block = rawIndex(core.now());
        core.advance(TimeSpan.seconds(4));
        await expect.poll(() => store.open(key)?.index, POLL).toBe(block + 1);
        expect(servesBuffer()).toBe(true);
      } finally {
        await close();
      }
    });
  });

  describe("long window", () => {
    it("should read raw samples only at the live edge", async () => {
      const core = new SyntheticCore({
        rate: Rate.hz(0.5),
        start: START,
        now: START.add(TimeSpan.days(31)),
      });
      const raw: TimeRange[] = [];
      const { subscribe, close } = createPoints(core, async (tr, keys, reduction) => {
        if (reduction == null) raw.push(tr);
        return core.read(tr, keys, reduction);
      });
      const window = dynamicWindow(TimeSpan.days(30));
      const points = subscribe(core.data[0].key, window, TILED);
      try {
        await expect.poll(() => raw.length, POLL).toBeGreaterThan(0);
        const edge = core.now().sub(tileSpan(RAW_LEVEL).mult(2));
        raw.forEach((tr) =>
          expect(tr.start.valueOf()).toBeGreaterThanOrEqual(edge.valueOf()),
        );
        await expect.poll(() => points.value().status.variant, POLL).toBe("success");
        expect(points.value().series.series.length).toBeLessThan(64);
      } finally {
        await close();
      }
    });

    it("should read each tile once as the edge moves", async () => {
      const core = new SyntheticCore({
        rate: Rate.hz(0.5),
        start: START,
        now: START.add(TimeSpan.days(31)),
      });
      const reduced: TimeRange[] = [];
      const { subscribe, close } = createPoints(core, async (tr, keys, reduction) => {
        if (reduction != null) reduced.push(tr);
        return core.read(tr, keys, reduction);
      });
      const window = dynamicWindow(TimeSpan.days(30));
      const points = subscribe(core.data[0].key, window, TILED);
      const block = tileSpan(RAW_LEVEL);
      try {
        for (let i = 0; i < 8; i++) {
          await expect.poll(() => points.value().status.variant, POLL).toBe("success");
          core.advance(block);
          points.value();
        }
        await expect.poll(() => points.value().status.variant, POLL).toBe("success");
        expect(reduced.filter((tr) => !isTile(tr))).toEqual([]);
        const ranges = reduced.map((tr) => `${tr.start.valueOf()}/${tr.end.valueOf()}`);
        expect(new Set(ranges).size).toBe(ranges.length);
      } finally {
        await close();
      }
    });
  });

  it("should read raw samples again after a failed read", async () => {
    const core = createCore();
    let failures = 1;
    const { subscribe, close } = createPoints(core, async (tr, keys, reduction) => {
      if (failures-- > 0) throw new Error("stream closed unexpectedly");
      return core.read(tr, keys, reduction);
    });
    const window = dynamicWindow(TimeSpan.minutes(2));
    const points = subscribe(core.data[0].key, window, RAW);
    try {
      await expect.poll(() => points.value().status.variant, POLL).toBe("success");
      expect(failures).toBeLessThan(0);
      expect(points.value().series.length).toBeGreaterThanOrEqual(12_000);
    } finally {
      await close();
    }
  });

  it("should stop finding a channel once its retries run out", async () => {
    const core = createCore();
    let retrieves = 0;
    core.retrieveChannels = async () => {
      retrieves++;
      throw new Error("stream closed unexpectedly");
    };
    const { subscribe, close } = createPoints(
      core,
      undefined,
      {},
      {
        ...BREAKER,
        maxRetries: 1,
      },
    );
    const points = subscribe(
      core.data[0].key,
      dynamicWindow(TimeSpan.seconds(30)),
      RAW,
    );
    try {
      await expect
        .poll(() => points.value().status, POLL)
        .toMatchObject({ variant: "error", description: "stream closed unexpectedly" });
      await sleep.sleep(TimeSpan.milliseconds(100));
      points.value();
      expect(retrieves).toBe(2);
    } finally {
      await close();
    }
  });

  it("should read a window that stops streaming while its read waits", async () => {
    const core = createCore();
    const { subscribe, close } = createPoints(core);
    const now = core.now();
    const live = new TimeRange(
      now.sub(TimeSpan.minutes(2)),
      now.add(TimeSpan.minutes(1)),
    );
    const points = subscribe(core.data[0].key, staticWindow(live), RAW);
    try {
      await expect.poll(() => points.value().series.length, POLL).toBeGreaterThan(0);
      core.disconnect();
      await sleep.sleep(TimeSpan.milliseconds(100));
      const past = new TimeRange(START, START.add(TimeSpan.minutes(2)));
      points.update(staticWindow(past), RAW);
      points.value();
      core.reconnect();
      await expect
        .poll(() => points.value().series.length, POLL)
        .toBeGreaterThanOrEqual(12_000);
    } finally {
      await close();
    }
  });

  it("should stop reading raw samples after a definitive failure", async () => {
    const core = createCore();
    let reads = 0;
    const { subscribe, close } = createPoints(core, async () => {
      reads++;
      throw new AccessDeniedError("no read access");
    });
    const points = subscribe(core.data[0].key, dynamicWindow(TimeSpan.minutes(2)), RAW);
    try {
      await expect
        .poll(() => points.value().status, POLL)
        .toMatchObject({
          variant: "error",
          message: "Failed to read channel data",
          description: "You do not have permission to do that. no read access",
        });
      await sleep.sleep(TimeSpan.milliseconds(100));
      const settled = reads;
      await sleep.sleep(TimeSpan.milliseconds(100));
      expect(reads).toBe(settled);
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

  it("should keep a rejected tile stopped while the view moves", async () => {
    const core = createCore();
    let rejected = 0;
    const { subscribe, close } = createPoints(core, async (tr, keys, reduction) => {
      if (reduction != null && rejected++ === 0)
        throw new ValidationError("reduction rejected");
      return core.read(tr, keys, reduction);
    });
    const range = new TimeRange(START, START.add(TimeSpan.minutes(8)));
    const window = staticWindow(range);
    const points = subscribe(core.data[0].key, window, TILED);
    try {
      await expect
        .poll(() => points.value().status.description, POLL)
        .toBe("reduction rejected");
      points.update(window, { ...TILED, view: range });
      await sleep.sleep(TimeSpan.milliseconds(50));
      expect(points.value().status).toMatchObject({
        variant: "error",
        message: "Failed to read channel data",
        description: "reduction rejected",
      });
    } finally {
      await close();
    }
  });

  it.each([
    ["a zoomed view", { view: new TimeRange(START, START.add(TimeSpan.minutes(1))) }],
    ["kept extremes", { aggregation: "average", extremesKept: true }],
  ] as const)("should finish loading the tiles of %s", async (_, limit) => {
    const core = createCore();
    const { subscribe, close } = createPoints(core);
    const window = staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(8))));
    const points = subscribe(core.data[0].key, window, { ...TILED, ...limit });
    try {
      await expect.poll(() => points.value().status.variant, POLL).toBe("success");
      expect(points.value().series.length).toBeGreaterThan(0);
    } finally {
      await close();
    }
  });

  it("should draw a view that reaches past a static window", async () => {
    const core = createCore();
    const { subscribe, close } = createPoints(core);
    const range = new TimeRange(
      START.add(TimeSpan.minutes(4)),
      START.add(TimeSpan.minutes(6)),
    );
    const view = new TimeRange(
      START.add(TimeSpan.minutes(2)),
      START.add(TimeSpan.minutes(8)),
    );
    const points = subscribe(core.index.key, staticWindow(range), { ...TILED, view });
    try {
      await expect.poll(() => points.value().status.variant, POLL).toBe("success");
      const { lower, upper } = points.value().series.bounds;
      expect(lower).toBeLessThan(Number(range.start.valueOf()));
      expect(upper).toBeGreaterThan(Number(range.end.valueOf()));
    } finally {
      await close();
    }
  });

  it("should evict the tiles it unpins past the budget", async () => {
    const core = createCore();
    const { store, subscribe, unsubscribe, close } = createPoints(core, undefined, {
      budget: Size.bytes(1),
    });
    const key = core.data[0].key;
    const span = TimeSpan.minutes(4);
    const level = tileLevel(new TimeRange(START, START.add(span)));
    const index = tileIndex(START, level) + 1;
    const { start } = tileRange({ level, index });
    const home = new TimeRange(start, start.add(span));
    const pointLimit = tilePointLimit({
      level,
      view: home,
      points: TILED.count,
      drawn: TILED.aggregation,
      aggregation: TILED.aggregation,
    });
    const tile = tileKey({
      key,
      level,
      index,
      pointLimit,
      aggregation: TILED.aggregation,
    });
    try {
      const points = subscribe(key, staticWindow(home), TILED);
      await expect.poll(() => points.value().status.variant, POLL).toBe("success");
      expect(store.get(tile)?.pins).toBe(1);
      unsubscribe(points);
      expect(store.get(tile)).toBeUndefined();
    } finally {
      await close();
    }
  });

  it("should release the raw blocks the window moves past", async () => {
    const core = createCore();
    const { store, subscribe, close } = createPoints(core, undefined, {
      budget: Size.bytes(1),
    });
    const key = core.index.key;
    const span = TimeSpan.seconds(10);
    const first = rawIndex(core.now().sub(span));
    const points = subscribe(key, dynamicWindow(span), RAW);
    try {
      await expect.poll(() => points.value().status.variant, POLL).toBe("success");
      expect(store.findRaw(key, first)).toBeDefined();
      for (let i = 0; i < 6; i++) {
        core.advance(TimeSpan.seconds(30));
        await expect.poll(() => points.value().status.variant, POLL).toBe("success");
      }
      await expect.poll(() => store.findRaw(key, first), POLL).toBeUndefined();
    } finally {
      await close();
    }
  });

  it("should never pin a cached tile for a cut tile at its position", async () => {
    const core = createCore();
    const { store, subscribe, unsubscribe, close } = createPoints(core);
    const key = core.data[0].key;
    const span = TimeSpan.minutes(4);
    const level = tileLevel(new TimeRange(START, START.add(span)));
    const index = tileIndex(START, level) + 1;
    const { start } = tileRange({ level, index });
    const home = new TimeRange(start, start.add(span));
    const pointLimit = tilePointLimit({
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
