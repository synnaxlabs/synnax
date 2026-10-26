// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  array,
  DataType,
  dynamicWindow,
  Rate,
  Series,
  Size,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { UnexpectedError, UnsupportedError } from "@/errors";
import { domainOf, offsetOf, RAW_LEVEL, rawIndex } from "@/framer/cache/block";
import { tileRange } from "@/framer/cache/grid";
import { type Feed, type FeedOptions } from "@/framer/feed";
import { Frame } from "@/framer/frame";
import {
  type PointsRequest,
  type Snapshot,
  type SubscribeRequest,
  type Subscription,
} from "@/framer/subscription";
import { SyntheticCore, type SyntheticCoreProps } from "@/testutil";

const RATE = Rate.hz(100);
const PERIOD = RATE.period.valueOf();
const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };

const OPTIONS: FeedOptions = {
  batchDebounce: TimeSpan.milliseconds(5),
  removalDelay: TimeSpan.ZERO,
  breaker: {
    baseInterval: TimeSpan.milliseconds(10),
    maxInterval: TimeSpan.milliseconds(50),
  },
};

const createCore = (props: Partial<SyntheticCoreProps> = {}): SyntheticCore =>
  new SyntheticCore({
    rate: RATE,
    start: START,
    now: START.add(TimeSpan.minutes(10)),
    ...props,
  });

const raw = (keys: channel.Key[], span: TimeSpan): SubscribeRequest => ({
  keys,
  window: dynamicWindow(span),
  limit: { variant: "points", count: Infinity, aggregation: "min_max" },
});

const timesOf = (snap: Snapshot): bigint[] => Array.from(snap.series) as bigint[];

// Waits for the stream of history changes that the feed opens for its first
// subscription.
const historyStreamed = async (core: SyntheticCore): Promise<void> =>
  await expect.poll(() => core.streams(core.historyChange.key), POLL).toBe(true);

// Opens the feed's stream of history changes, so reads after it fetch blocks that the
// changes can keep valid.
const listen = async (core: SyntheticCore, feed: Feed): Promise<void> => {
  feed.subscribe(raw([], TimeSpan.seconds(1))).close();
  await historyStreamed(core);
};

// Returns the time of the Core's newest sample.
const newestOf = (core: SyntheticCore): bigint =>
  START.valueOf() +
  BigInt(Math.ceil(Number(core.now().sub(START).valueOf()) / Number(PERIOD)) - 1) *
    PERIOD;

// Advances the Core's clock by step on each tick of the real clock, so a writer keeps
// writing while a spec waits. Returns a function that stops it.
const tick = (core: SyntheticCore, step = TimeSpan.milliseconds(50)): (() => void) => {
  const id = setInterval(() => core.advance(step), 2);
  return () => clearInterval(id);
};

// Asserts that the timestamps hold each sample of the Core once, in order, with none
// missing.
const expectComplete = (times: bigint[]): void => {
  expect(times.length).toBeGreaterThan(0);
  const steps = new Set(times.slice(1).map((t, i) => t - times[i]));
  expect([...steps]).toEqual([PERIOD]);
};

// Returns the timestamps once they hold each sample from the window start to the newest
// frames once. Lag is how far the newest written sample trails the Core's clock.
const settled = async (
  core: SyntheticCore,
  sub: Subscription,
  key: channel.Key,
  span: TimeSpan,
  lag: TimeSpan = TimeSpan.ZERO,
): Promise<bigint[]> => {
  let times: bigint[] = [];
  await expect
    .poll(() => {
      const start = core.now().sub(span).valueOf();
      const recent = core.now().sub(lag).sub(TimeSpan.milliseconds(500)).valueOf();
      const snap = sub.value(key);
      times = timesOf(snap);
      if (snap.status.variant !== "success" || times.length < 2) return false;
      const steps = times.slice(1).every((t, i) => t - times[i] === PERIOD);
      return steps && times[0] <= start + PERIOD && times.at(-1)! >= recent;
    }, POLL)
    .toBe(true);
  return times;
};

// Returns the series of a data channel once they hold each sample from the window start
// to the newest frames once, at contiguous block alignments.
const settledData = async (
  core: SyntheticCore,
  sub: Subscription,
  key: channel.Key,
  span: TimeSpan,
  lag: TimeSpan = TimeSpan.ZERO,
): Promise<Series[]> => {
  let series: Series[] = [];
  await expect
    .poll(() => {
      const snap = sub.value(key);
      series = snap.series.series;
      if (snap.status.variant !== "success" || series.length === 0) return false;
      const start = core.now().sub(span).add(RATE.period);
      const recent = core.now().sub(lag).sub(TimeSpan.milliseconds(500));
      const contiguous = series.every((s, i) => {
        const prev = series[i - 1];
        if (prev == null) return true;
        if (domainOf(prev.alignment) !== domainOf(s.alignment))
          return offsetOf(s.alignment) === 0;
        return s.alignment === prev.alignment + BigInt(prev.length);
      });
      return (
        contiguous &&
        series[0].timeRange.start.beforeEq(start) &&
        series.at(-1)!.timeRange.end.afterEq(recent)
      );
    }, POLL)
    .toBe(true);
  return series;
};

describe("Feed.subscribe", () => {
  describe("join", () => {
    const cases: Array<[string, Partial<SyntheticCoreProps>]> = [
      ["untimed data series", { dataUntimed: true }],
      ["data framed apart from its index", { dataFramedApart: true }],
      ["acks that lag their keys", { ackDelay: TimeSpan.milliseconds(30) }],
    ];

    it.each(cases)("should join a channel streamed with %s", async (_, props) => {
      const core = createCore(props);
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.seconds(30);
      const key = core.data[0].key;
      const sub = feed.subscribe(raw([key], span));
      const stop = tick(core);
      try {
        await settledData(core, sub, key, span);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should keep the rest of a join when the Core rejects a key", async () => {
      const core = createCore({ channels: 2 });
      const [allowed, denied] = core.data.map((c) => c.key);
      core.deny(denied);
      let reads = 0;
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async (tr, keys, reduction) => {
          if (keys.includes(allowed)) reads++;
          return core.read(tr, keys, reduction);
        },
      });
      const span = TimeSpan.seconds(30);
      const a = feed.subscribe(raw([allowed], span));
      const stop = tick(core);
      try {
        await settledData(core, a, allowed, span);
        const before = reads;
        const b = feed.subscribe(raw([denied], span));
        await expect
          .poll(() => b.value(denied).status, POLL)
          .toMatchObject({
            variant: "error",
            message: "Failed to read channel data",
            description: `You do not have permission to do that. access to channel ${denied} denied`,
          });
        await sleep.sleep(TimeSpan.milliseconds(200));
        expect(reads).toBe(before);
        await settledData(core, a, allowed, span);
        b.close();
      } finally {
        stop();
        a.close();
        await feed.close();
      }
    });
  });

  describe("duplicates", () => {
    it("should show a sample that both the read and the stream hold once", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.seconds(30);
      const sub = feed.subscribe(raw([core.index.key], span));
      const stop = tick(core);
      try {
        await settled(core, sub, core.index.key, span);
        await sleep.sleep(TimeSpan.milliseconds(100));
        expectComplete(timesOf(sub.value(core.index.key)));
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should hold the samples of a writer that lags the commit point", async () => {
      // The newest written sample trails a block edge that the commit point passed.
      const edge = tileRange({
        level: RAW_LEVEL,
        index: rawIndex(START.add(TimeSpan.minutes(10))),
      }).start;
      const lag = TimeSpan.seconds(3);
      const core = createCore({ now: edge.add(TimeSpan.milliseconds(1500)), lag });
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.minutes(1);
      const [index, data] = [core.index.key, core.data[0].key];
      const sub = feed.subscribe(raw([index, data], span));
      const stop = tick(core, TimeSpan.milliseconds(10));
      try {
        await settled(core, sub, index, span, lag);
        await settledData(core, sub, data, span, lag);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });
  });

  describe("alignment", () => {
    it("should split streamed series at block edges with block offsets", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.minutes(3);
      const sub = feed.subscribe(raw([core.index.key], span));
      const stop = tick(core, TimeSpan.seconds(1));
      try {
        await settled(core, sub, core.index.key, span);
        const { series } = sub.value(core.index.key);
        series.series.forEach((s, i) => {
          const block = Number(domainOf(s.alignment));
          const first = new TimeStamp(s.at(0, true) as bigint);
          const last = new TimeStamp(s.at(-1, true) as bigint);
          expect(rawIndex(first)).toBe(block);
          expect(rawIndex(last)).toBe(block);
          const prev = series.series[i - 1];
          if (prev == null) return;
          if (domainOf(prev.alignment) === domainOf(s.alignment))
            expect(s.alignment).toBe(prev.alignment + BigInt(prev.length));
          else expect(offsetOf(s.alignment)).toBe(0);
        });
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should give a streamed block the offsets a fetch of it gives", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.minutes(3);
      const sub = feed.subscribe(raw([core.index.key], span));
      const stop = tick(core, TimeSpan.seconds(1));
      let streamed: Series[];
      try {
        await settled(core, sub, core.index.key, span);
        streamed = sub.value(core.index.key).series.series;
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
      const block = Number(domainOf(streamed.at(-2)!.alignment));
      const fresh = core.openFeed(OPTIONS);
      try {
        const fetched = await fresh.read(
          tileRange({ level: 16, index: block }),
          core.index.key,
        );
        const offsetOf = (series: Series[], t: bigint): bigint | undefined => {
          for (const s of series) {
            const at = (Array.from(s) as bigint[]).indexOf(t);
            if (at >= 0) return s.alignment + BigInt(at);
          }
          return undefined;
        };
        const sample = streamed.at(-2)!.at(-1, true) as bigint;
        expect(offsetOf(streamed, sample)).toBeDefined();
        expect(offsetOf(fetched.series, sample)).toBe(offsetOf(streamed, sample));
      } finally {
        await fresh.close();
      }
    });

    it("should pair the samples of a channel with those of its index", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.minutes(2);
      const x = feed.subscribe(raw([core.index.key], span));
      const y = feed.subscribe(raw([core.data[0].key], span));
      const stop = tick(core, TimeSpan.milliseconds(500));
      try {
        await settled(core, x, core.index.key, span);
        await expect
          .poll(() => y.value(core.data[0].key).status.variant, POLL)
          .toBe("success");
        const xs = x.value(core.index.key).series.series;
        const ys = y.value(core.data[0].key).series.series;
        // Returns true when the x series hold every alignment in [lower, upper).
        const covered = (lower: bigint, upper: bigint): boolean => {
          let at = lower;
          for (const { alignmentBounds: b } of xs)
            if (b.lower <= at && at < b.upper) at = b.upper;
          return at >= upper;
        };
        const newest = xs.at(-1)!.alignmentBounds.upper;
        ys.forEach((s) => {
          const { lower, upper } = s.alignmentBounds;
          // The two subscriptions draw at most one frame apart.
          if (upper <= newest && lower >= xs[0].alignmentBounds.lower)
            expect(covered(lower, upper)).toBe(true);
        });
      } finally {
        stop();
        x.close();
        y.close();
        await feed.close();
      }
    });
  });

  describe("gap", () => {
    it("should fill samples the stream skipped with a read", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      // The window outlasts the spec, so a hole cannot roll out of it.
      const span = TimeSpan.minutes(10);
      const sub = feed.subscribe(raw([core.index.key], span));
      let stop = tick(core);
      try {
        await settled(core, sub, core.index.key, span);
        stop();
        const skipped = core.now();
        core.advance(TimeSpan.seconds(2), false);
        stop = tick(core);
        const times = await settled(core, sub, core.index.key, span);
        expect(times.some((t) => t > skipped.valueOf() + PERIOD)).toBe(true);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should read no hole in a calculated channel's stream alignments", async () => {
      const core = createCore({ derived: true });
      const feed = core.openFeed(OPTIONS);
      const key = core.calculated!.key;
      const sub = feed.subscribe(raw([key], TimeSpan.seconds(30)));
      const stop = tick(core);
      try {
        await expect.poll(() => sub.value(key).series.length > 3000, POLL).toBe(true);
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        const reads = core.reads;
        await sleep.sleep(TimeSpan.milliseconds(200));
        expect(core.reads).toBe(reads);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });
  });

  describe("rolling edge", () => {
    it("should never leave a span of the window blank as tiles replace raw samples", async () => {
      const core = createCore({ now: START.add(TimeSpan.hours(1)) });
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.minutes(5);
      const sub = feed.subscribe({
        keys: [core.index.key],
        window: dynamicWindow(span),
        limit: { variant: "points", count: 200, aggregation: "min_max" },
      });
      const stop = tick(core, TimeSpan.seconds(2));
      try {
        await expect
          .poll(() => sub.value(core.index.key).status.variant, POLL)
          .toBe("success");
        const end = core.now().add(TimeSpan.minutes(6));
        while (core.now().before(end)) {
          const home = new TimeRange(core.now().sub(span), core.now());
          const { series } = sub.value(core.index.key);
          const ranges = series.series.map((s) => s.timeRange);
          expect(ranges[0].start.beforeEq(home.start)).toBe(true);
          ranges.slice(1).forEach((r, i) => {
            expect(r.start.valueOf() - ranges[i].end.valueOf()).toBeLessThan(PERIOD);
          });
          await sleep.sleep(TimeSpan.milliseconds(3));
        }
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should release raw blocks behind the window", async () => {
      const core = createCore();
      const feed = core.openFeed({ ...OPTIONS, budget: Size.kilobytes(1) });
      const span = TimeSpan.seconds(10);
      const sub = feed.subscribe(raw([core.index.key], span));
      const first = core.now();
      const stop = tick(core, TimeSpan.seconds(2));
      try {
        await settled(core, sub, core.index.key, span);
        await expect
          .poll(() => core.now().sub(first).valueOf() > TimeSpan.minutes(3).valueOf())
          .toBe(true);
        await settled(core, sub, core.index.key, span);
        const reads = core.reads;
        await feed.read(new TimeRange(first.sub(span), first), core.index.key);
        expect(core.reads).toBe(reads + 1);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });
  });

  describe("static window that ends after now", () => {
    it("should stream the window's edge without reads until its end commits", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const start = core.now().sub(TimeSpan.seconds(10));
      const end = core.now().add(TimeSpan.seconds(5));
      const sub = feed.subscribe({
        keys: [key],
        window: staticWindow(new TimeRange(start, end)),
        limit: { variant: "points", count: Infinity, aggregation: "min_max" },
      });
      const newest = (): bigint | undefined => timesOf(sub.value(key)).at(-1);
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        await expect.poll(newest, POLL).toBe(newestOf(core));
        const reads = core.reads;
        core.advance(TimeSpan.seconds(2));
        await expect.poll(newest, POLL).toBe(newestOf(core));
        expect(sub.value(key).status.variant).toBe("success");
        expect(core.reads).toBe(reads);
        core.advance(TimeSpan.seconds(3));
        await expect.poll(newest, POLL).toBe(end.valueOf() - PERIOD);
        core.advance(TimeSpan.seconds(2));
        await expect.poll(() => (sub.value(key), core.streams(key)), POLL).toBe(false);
        const times = timesOf(sub.value(key));
        expect(times[0] <= start.valueOf()).toBe(true);
        expectComplete(times);
        core.advance(TimeSpan.seconds(2));
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(timesOf(sub.value(key))).toEqual(times);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should serve the stored samples stamped after now", async () => {
      // The writer's clock runs two minutes ahead of the Core's.
      const core = createCore({ lag: TimeSpan.minutes(-2) });
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const window = new TimeRange(
        core.now().sub(TimeSpan.minutes(1)),
        core.now().add(TimeSpan.minutes(5)),
      );
      const sub = feed.subscribe({
        keys: [key],
        window: staticWindow(window),
        limit: { variant: "points", count: Infinity, aggregation: "min_max" },
      });
      const newest = Array.from(core.readLatest([key], 1).get(key)).at(-1);
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        await expect.poll(() => timesOf(sub.value(key)).at(-1), POLL).toBe(newest);
        const times = timesOf(sub.value(key));
        expect(times[0] <= window.start.valueOf()).toBe(true);
        expectComplete(times);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should draw tiles up to the raw samples at the window's edge", async () => {
      const core = createCore({ now: START.add(TimeSpan.hours(1)) });
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const start = START.add(TimeSpan.minutes(10));
      const sub = feed.subscribe({
        keys: [key],
        window: staticWindow(new TimeRange(start, core.now().add(TimeSpan.minutes(1)))),
        limit: { variant: "points", count: 200, aggregation: "min_max" },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        core.advance(TimeSpan.seconds(10));
        await expect
          .poll(() => timesOf(sub.value(key)).at(-1), POLL)
          .toBe(newestOf(core));
        const { series } = sub.value(key);
        // Raw samples of the whole window would be 300,000.
        expect(series.length).toBeLessThan(15_000);
        const ranges = series.series.map((s) => s.timeRange);
        expect(ranges[0].start.valueOf()).toBe(start.valueOf());
        ranges.slice(1).forEach((r, i) => {
          expect(r.start.valueOf() - ranges[i].end.valueOf()).toBeLessThan(PERIOD);
        });
      } finally {
        sub.close();
        await feed.close();
      }
    });
  });

  describe("data type change", () => {
    it("should restart a channel whose data type changes, never mixing types", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.data[0].key;
      const sub = feed.subscribe(raw([key], TimeSpan.seconds(30)));
      const stop = tick(core);
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        core.retype(key, DataType.FLOAT64);
        let mixed = false;
        await expect
          .poll(() => {
            const types = new Set(
              sub.value(key).series.series.map((s) => s.dataType.toString()),
            );
            mixed ||= types.size > 1;
            return [...types];
          }, POLL)
          .toEqual([DataType.FLOAT64.toString()]);
        expect(mixed).toBe(false);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });
  });

  describe("restart and reconnect", () => {
    it("should keep time order across a Core restart", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.seconds(30);
      const sub = feed.subscribe(raw([core.index.key], span));
      const stop = tick(core);
      try {
        await settled(core, sub, core.index.key, span);
        const restartedAt = core.now();
        core.restart();
        await expect
          .poll(() => core.now().after(restartedAt.add(TimeSpan.seconds(5))))
          .toBe(true);
        await settled(core, sub, core.index.key, span);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should report reconnecting during an outage and fill it after", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.seconds(30);
      const sub = feed.subscribe(raw([core.index.key], span));
      const stop = tick(core);
      try {
        await settled(core, sub, core.index.key, span);
        core.disconnect();
        await expect
          .poll(() => sub.value(core.index.key).status, POLL)
          .toMatchObject({ variant: "warning", message: "reconnecting" });
        core.reconnect();
        await settled(core, sub, core.index.key, span);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should read the span an outage skips while a catch-up read runs", async () => {
      const core = createCore();
      const key = core.index.key;
      let held = true;
      let entered = false;
      let release = (): void => {};
      const released = new Promise<void>((resolve) => (release = resolve));
      let read = false;
      let opens = 0;
      let reopen = (): void => {};
      const reopened = new Promise<void>((resolve) => (reopen = resolve));
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async (tr, keys, reduction) => {
          // Only a catch-up read ends after the Core's clock.
          if (!held || !tr.end.after(core.now())) return core.read(tr, keys, reduction);
          held = false;
          entered = true;
          await released;
          const frame = core.read(tr, keys, reduction);
          read = true;
          return frame;
        },
        openStreamer: async (config) => {
          if (++opens > 1) await reopened;
          return core.openStreamer(config);
        },
      });
      const span = TimeSpan.minutes(2);
      const sub = feed.subscribe(raw([key], span));
      try {
        await expect.poll(() => (sub.value(key), entered), POLL).toBe(true);
        core.disconnect();
        core.reconnect();
        await expect.poll(() => opens, POLL).toBe(2);
        core.advance(TimeSpan.seconds(2));
        release();
        await expect.poll(() => read, POLL).toBe(true);
        core.advance(TimeSpan.seconds(2));
        reopen();
        const stop = tick(core, TimeSpan.milliseconds(10));
        try {
          await settled(core, sub, key, span);
        } finally {
          stop();
        }
      } finally {
        sub.close();
        await feed.close();
      }
    });
  });

  describe("latest", () => {
    it("should keep the newest sample over all time", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 1 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        expect(sub.value(key).series.at(-1)).toBe(newestOf(core));
        expect(sub.value(key).series.length).toBe(1);
        core.advance(TimeSpan.seconds(1));
        await expect
          .poll(() => sub.value(key).series.at(-1), POLL)
          .toBe(newestOf(core));
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should keep the newest count samples in the window", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const window = dynamicWindow(TimeSpan.seconds(2));
      const sub = feed.subscribe({
        keys: [key],
        window,
        limit: { variant: "latest", count: 1000 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        expectComplete(timesOf(sub.value(key)));
        expect(sub.value(key).series.length).toBe(200);
        for (let i = 0; i < 10; i++) core.advance(TimeSpan.milliseconds(500));
        await expect
          .poll(() => timesOf(sub.value(key)).at(-1), POLL)
          .toBe(
            core.now().valueOf() - (core.now().sub(START).valueOf() % PERIOD || PERIOD),
          );
        const times = timesOf(sub.value(key));
        expectComplete(times);
        // Whole series leave, so the snapshot keeps at most one series past the window.
        expect(times.length).toBeLessThanOrEqual(250);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should trim whole series past the count from the front", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 10 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        const before = sub.value(key).series.series;
        core.advance(TimeSpan.milliseconds(50));
        await expect.poll(() => sub.value(key).series.series.length, POLL).toBe(2);
        const after = sub.value(key).series.series;
        expect(after[0]).toBe(before.at(-1));
        expect(after[1].length).toBe(5);
        core.advance(TimeSpan.milliseconds(200));
        await expect.poll(() => sub.value(key).series.series.length, POLL).toBe(1);
        // The two frames grow one buffer.
        expect(sub.value(key).series.length).toBe(20);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should keep the samples of a key that a new request keeps", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const [a, b] = [core.index.key, core.data[0].key];
      const limit = { variant: "latest", count: 100 } as const;
      const sub = feed.subscribe({ keys: [a], limit });
      try {
        await expect.poll(() => sub.value(a).status.variant, POLL).toBe("success");
        const before = sub.value(a).series.series;
        const reads = core.reads;
        sub.update({ keys: [a, b], limit });
        expect(sub.value(a).series.series).toEqual(before);
        expect(sub.value(a).status.variant).toBe("success");
        await expect.poll(() => sub.value(b).status.variant, POLL).toBe("success");
        expect(core.reads).toBe(reads + 1);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should read the newest samples again after the stream reconnects", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 1000 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        const [first] = sub.value(key).series.series;
        core.disconnect();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("warning");
        core.advance(TimeSpan.seconds(1));
        core.reconnect();
        await expect
          .poll(() => timesOf(sub.value(key)).at(-1), POLL)
          .toBe(newestOf(core));
        const snap = sub.value(key);
        expect(snap.status.variant).toBe("success");
        expect(snap.series.series[0]).toBe(first);
        expectComplete(timesOf(snap));
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should leave out a streamed write older than the newest sample", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 2 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        // A writer fills an earlier span in a domain of its own.
        const older = new Series({
          data: new BigInt64Array([START.valueOf()]),
          dataType: DataType.TIMESTAMP,
          timeRange: new TimeRange(START, START.add(TimeSpan.NANOSECOND)),
          alignment: 9n << 32n,
        });
        core.stream(new Frame(key, older));
        core.advance(TimeSpan.milliseconds(10));
        await expect
          .poll(() => timesOf(sub.value(key)).at(-1), POLL)
          .toBe(newestOf(core));
        const times = timesOf(sub.value(key));
        expect(times).toHaveLength(3);
        expectComplete(times);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should keep samples after the end of a static window out of the count", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const end = core.now().sub(TimeSpan.seconds(1));
      const sub = feed.subscribe({
        keys: [key],
        window: staticWindow(new TimeRange(end.sub(TimeSpan.seconds(1)), end)),
        limit: { variant: "latest", count: 100 },
      });
      const marker = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 1 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        await expect.poll(() => marker.value(key).status.variant, POLL).toBe("success");
        const before = timesOf(sub.value(key));
        expect(before).toHaveLength(100);
        core.advance(TimeSpan.seconds(1));
        // The stream delivers each frame to every subscription before the next.
        await expect
          .poll(() => marker.value(key).series.at(-1), POLL)
          .toBe(newestOf(core));
        expect(timesOf(sub.value(key))).toEqual(before);
      } finally {
        marker.close();
        sub.close();
        await feed.close();
      }
    });

    it("should only stream a virtual channel", async () => {
      const core = createCore({ derived: true });
      const feed = core.openFeed(OPTIONS);
      const key = core.virtual!.key;
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 100 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        expect(sub.value(key).series.length).toBe(0);
        const reads = core.reads;
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => sub.value(key).series.length, POLL).toBe(10);
        expect(core.reads).toBe(reads);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should keep every frame of a virtual channel with no index", async () => {
      const core = createCore({ derived: true });
      const feed = core.openFeed(OPTIONS);
      const key = core.virtual!.key;
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 100 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => sub.value(key).series.length, POLL).toBe(10);
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => sub.value(key).series.length, POLL).toBe(20);
        expect(sub.value(key).series.series.map((s) => s.alignment)).toEqual([0n, 0n]);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should stream a calculation the Core cannot read, with a warning", async () => {
      const core = createCore({ derived: true });
      const { key } = core.calculated!;
      let rejected = 0;
      const feed = core.openFeed({
        ...OPTIONS,
        readLatest: async (keys, count, tr) => {
          if (!keys.includes(key)) return core.readLatest(keys, count, tr);
          rejected++;
          throw new UnsupportedError("cannot open iterator on virtual channel");
        },
      });
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 100 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("warning");
        expect(sub.value(key).status.message).toBe(
          "Failed to read calculated channel history",
        );
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => sub.value(key).series.length, POLL).toBe(10);
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(rejected).toBe(1);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should give each channel of a shared read its own result", async () => {
      const core = createCore({ derived: true });
      const { key: calculated } = core.calculated!;
      const { key: data } = core.data[0];
      let shared = 0;
      const feed = core.openFeed({
        ...OPTIONS,
        batchDebounce: TimeSpan.milliseconds(50),
        readLatest: async (keys, count, tr) => {
          if (keys.length > 1) shared++;
          if (!keys.includes(calculated)) return core.readLatest(keys, count, tr);
          throw new UnsupportedError("cannot open iterator on virtual channel");
        },
      });
      const sub = feed.subscribe({
        keys: [data, calculated],
        limit: { variant: "latest", count: 1 },
      });
      try {
        await expect.poll(() => sub.value(data).status.variant, POLL).toBe("success");
        expect(sub.value(data).series.length).toBe(1);
        expect(sub.value(calculated).status.variant).toBe("warning");
        expect(shared).toBe(1);
      } finally {
        sub.close();
        await feed.close();
      }
    });
  });

  describe("update and status", () => {
    const createStatic = (key: channel.Key): PointsRequest => ({
      keys: [key],
      window: staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(8)))),
      limit: { variant: "points", count: 100, aggregation: "min_max" },
    });

    it("should draw held tiles while a zoom's tiles load", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.data[0].key;
      const request = createStatic(key);
      const sub = feed.subscribe(request);
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        const view = new TimeRange(
          START.add(TimeSpan.minutes(4)),
          START.add(TimeSpan.minutes(5)),
        );
        sub.update({ ...request, limit: { ...request.limit, view } });
        const zoomed = sub.value(key);
        expect(zoomed.series.length).toBeGreaterThan(0);
        expect(zoomed.status.variant).toBe("loading");
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should do nothing for an equal request", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.data[0].key;
      const sub = feed.subscribe(createStatic(key));
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        const before = sub.value(key).series.series;
        let changes = 0;
        sub.onChange(() => changes++);
        sub.update(createStatic(key));
        expect(sub.value(key).series.series).toEqual(before);
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(changes).toBe(0);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should move stream demand to the keys of a new request", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const [a, b] = [core.index.key, core.data[0].key];
      const sub = feed.subscribe({ keys: [a], limit: { variant: "latest", count: 1 } });
      try {
        await expect.poll(() => sub.value(a).status.variant, POLL).toBe("success");
        sub.update({ keys: [b], limit: { variant: "latest", count: 1 } });
        expect(() => sub.value(a)).toThrow(`channel ${a} is not in the subscription`);
        await expect.poll(() => sub.value(b).status.variant, POLL).toBe("success");
        core.advance(TimeSpan.seconds(1));
        await expect
          .poll(() => sub.value(b).series.series.at(-1)?.length, POLL)
          .toBe(100);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should park a channel the Core rejects with its error", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const sub = feed.subscribe(createStatic(99));
      try {
        await expect
          .poll(() => sub.value(99).status, POLL)
          .toMatchObject({
            variant: "error",
            message: "Failed to read channel data",
            description: "channel 99 not found",
          });
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(sub.value(99).series.length).toBe(0);
        expect(sub.value(99).status.variant).toBe("error");
      } finally {
        sub.close();
        await feed.close();
      }
    });
  });

  describe("onChange", () => {
    it("should call a handler once for a frame of several channels", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.seconds(30);
      const [index, data] = [core.index.key, core.data[0].key];
      const sub = feed.subscribe(raw([index, data], span));
      try {
        await settled(core, sub, index, span);
        await settledData(core, sub, data, span);
        let calls = 0;
        sub.onChange(() => calls++);
        core.advance(TimeSpan.seconds(1));
        await expect.poll(() => calls, POLL).toBeGreaterThan(0);
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(calls).toBe(1);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should keep the join and later handlers going when one throws", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.seconds(30);
      const key = core.index.key;
      const sub = feed.subscribe(raw([key], span));
      let calls = 0;
      sub.onChange(() => {
        throw new Error("render failed");
      });
      sub.onChange(() => calls++);
      const stop = tick(core);
      try {
        expectComplete(await settled(core, sub, key, span));
        expect(calls).toBeGreaterThan(0);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });
  });

  describe("validity", () => {
    it("should drop the samples a history change names, and keep the rest", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.data[0].key;
      const window = staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(4))));
      const sub = feed.subscribe({
        keys: [key],
        window,
        limit: { variant: "points", count: Infinity, aggregation: "min_max" },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        const before = Array.from(sub.value(key).series) as number[];
        await historyStreamed(core);
        const changed = new TimeRange(
          START.add(TimeSpan.minutes(2)),
          START.add(TimeSpan.minutes(3)),
        );
        core.rewrite(key, changed, 10);
        await expect
          .poll(() => (Array.from(sub.value(key).series) as number[])[12_000], POLL)
          .toBeCloseTo(before[12_000] + 10);
        const after = Array.from(sub.value(key).series) as number[];
        expect(after).toHaveLength(before.length);
        expect(after[0]).toBe(before[0]);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it.each([
      ["an unknown node", undefined],
      ["another node", 2],
    ])("should expire the blocks of a channel on %s", async (_, node) => {
      const core = createCore();
      core.node = node;
      const feed = core.openFeed(OPTIONS);
      const key = core.data[0].key;
      const window = staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(1))));
      const load = async (): Promise<number> => {
        const reads = core.reads;
        const sub = feed.subscribe({
          keys: [key],
          window,
          limit: { variant: "points", count: Infinity, aggregation: "min_max" },
        });
        try {
          await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        } finally {
          sub.close();
        }
        return core.reads - reads;
      };
      try {
        await listen(core, feed);
        expect(await load()).toBe(1);
        core.advance(TimeSpan.minutes(20));
        expect(await load()).toBe(1);
      } finally {
        await feed.close();
      }
    });

    it("should keep signaled blocks past the expiry, and expire calculated ones", async () => {
      const core = createCore({ derived: true });
      const feed = core.openFeed(OPTIONS);
      const tr = new TimeRange(START, START.add(TimeSpan.minutes(1)));
      const window = staticWindow(tr);
      // Returns the Core reads that a subscription to the key's window makes.
      const load = async (key: channel.Key): Promise<number> => {
        const reads = core.reads;
        const sub = feed.subscribe({
          keys: [key],
          window,
          limit: { variant: "points", count: Infinity, aggregation: "min_max" },
        });
        try {
          await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        } finally {
          sub.close();
        }
        return core.reads - reads;
      };
      const { key: data } = core.data[0];
      const { key: calculated } = core.calculated!;
      try {
        // Blocks fetched while history changes stream stay valid until one names them.
        await load(data);
        await historyStreamed(core);
        core.rewrite(data, tr, 0);
        core.rewrite(calculated, tr, 0);
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(await load(data)).toBe(1);
        expect(await load(calculated)).toBe(1);
        core.advance(TimeSpan.minutes(20));
        expect(await load(data)).toBe(0);
        expect(await load(calculated)).toBe(1);
      } finally {
        await feed.close();
      }
    });

    it.each([
      ["raw samples", Infinity],
      ["tiles", 100],
    ])(
      "should read %s again once the blocks that appends land in expire",
      async (_, count) => {
        // The writer trails the clock, so appends land in blocks the feed read empty.
        const core = createCore({ lag: TimeSpan.minutes(5) });
        const feed = core.openFeed(OPTIONS);
        const key = core.data[0].key;
        const request: PointsRequest = {
          keys: [key],
          window: staticWindow(
            new TimeRange(
              START.add(TimeSpan.minutes(6)),
              START.add(TimeSpan.minutes(7)),
            ),
          ),
          limit: { variant: "points", count, aggregation: "min_max" },
        };
        // Returns the samples that a subscription to the window draws.
        const load = async (f: Feed): Promise<number[]> => {
          const sub = f.subscribe(request);
          try {
            await expect
              .poll(() => sub.value(key).status.variant, POLL)
              .toBe("success");
            return Array.from(sub.value(key).series) as number[];
          } finally {
            sub.close();
          }
        };
        const fresh = core.openFeed(OPTIONS);
        try {
          await listen(core, feed);
          expect(await load(feed)).toEqual([]);
          core.advance(TimeSpan.minutes(20));
          const drawn = await load(fresh);
          expect(drawn.length).toBeGreaterThan(0);
          expect(await load(feed)).toEqual(drawn);
        } finally {
          await fresh.close();
          await feed.close();
        }
      },
    );
  });

  describe("history change during a fetch", () => {
    // Opens a feed whose first read of the key waits for release, after it computed its
    // samples.
    const createHeld = (core: SyntheticCore, key: channel.Key) => {
      let release: () => void = () => {};
      const released = new Promise<void>((resolve) => (release = resolve));
      let reads = 0;
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async (tr, keys, reduction) => {
          const frame = core.read(tr, keys, reduction);
          if (keys.includes(key) && reads++ === 0) await released;
          return frame;
        },
      });
      return { feed, release, reads: () => reads };
    };
    const window = staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(4))));
    const changed = new TimeRange(START, START.add(TimeSpan.minutes(4)));

    it("should read raw samples again when a change lands during their fetch", async () => {
      const core = createCore();
      const key = core.data[0].key;
      const { feed, release, reads } = createHeld(core, key);
      const sub = feed.subscribe({
        keys: [key],
        window,
        limit: { variant: "points", count: Infinity, aggregation: "min_max" },
      });
      try {
        await expect
          .poll(() => {
            sub.value(key);
            return reads();
          }, POLL)
          .toBe(1);
        const before = core.read(changed, [key]).get(key).at(0) as number;
        await historyStreamed(core);
        core.rewrite(key, changed, 10);
        await sleep.sleep(TimeSpan.milliseconds(50));
        release();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        expect(sub.value(key).series.at(0)).toBeCloseTo(before + 10);
        expect(reads()).toBe(2);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it.each([
      ["raw samples", Infinity],
      ["tiles", 100],
    ])("should land %s while changes elsewhere in history land", async (_, count) => {
      const core = createCore({ now: START.add(TimeSpan.hours(10)) });
      const key = core.data[0].key;
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async (tr, keys, reduction) => {
          await sleep.sleep(TimeSpan.milliseconds(20));
          return core.read(tr, keys, reduction);
        },
      });
      const sub = feed.subscribe({
        keys: [key],
        window,
        limit: { variant: "points", count, aggregation: "min_max" },
      });
      const far = START.add(TimeSpan.hours(9)).spanRange(TimeSpan.seconds(1));
      await expect.poll(() => core.streams(core.historyChange.key), POLL).toBe(true);
      const churn = setInterval(() => core.rewrite(key, far, 0), 10);
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
      } finally {
        clearInterval(churn);
        sub.close();
        await feed.close();
      }
    });

    it("should read a tile again when a change lands during its fetch", async () => {
      const core = createCore();
      const key = core.data[0].key;
      const { feed, release, reads } = createHeld(core, key);
      const request: SubscribeRequest = {
        keys: [key],
        window,
        limit: { variant: "points", count: 100, aggregation: "min_max" },
      };
      const sub = feed.subscribe(request);
      const fresh = core.openFeed(OPTIONS);
      try {
        await expect
          .poll(() => {
            sub.value(key);
            return reads();
          }, POLL)
          .toBeGreaterThan(0);
        await historyStreamed(core);
        core.rewrite(key, changed, 10);
        await sleep.sleep(TimeSpan.milliseconds(50));
        release();
        const check = fresh.subscribe(request);
        await expect.poll(() => check.value(key).status.variant, POLL).toBe("success");
        await expect
          .poll(() => Array.from(sub.value(key).series), POLL)
          .toEqual(Array.from(check.value(key).series));
        check.close();
      } finally {
        sub.close();
        await feed.close();
        await fresh.close();
      }
    });
  });

  describe("close", () => {
    it("should close every subscription of the feed", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.data[0].key;
      const request = raw([key], TimeSpan.seconds(30));
      const sub = feed.subscribe(request);
      await feed.close();
      expect(() => sub.update(request)).toThrow(
        new UnexpectedError("subscription is closed"),
      );
    });

    it("should close while the history change channel is retrieved", async () => {
      const core = createCore();
      const feed = core.openFeed({
        ...OPTIONS,
        retrieveChannels: async (params) => {
          if (array.toArray(params).includes("sy_history_change"))
            await new Promise(() => {});
          return await core.retrieveChannels(params);
        },
      });
      const sub = feed.subscribe(raw([core.data[0].key], TimeSpan.seconds(30)));
      sub.value(core.data[0].key);
      await expect(feed.close()).resolves.toBeUndefined();
    });
  });
});
