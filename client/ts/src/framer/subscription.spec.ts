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
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { assert, describe, expect, it, vi } from "vitest";

import { type channel } from "@/channel";
import { framer } from "@/framer";
import { domainOf, offsetOf, RAW_LEVEL, rawIndex } from "@/framer/cache/block";
import { index as tileIndex, tileRange, tileSpan } from "@/framer/cache/grid";
import { UnexpectedError, UnsupportedError, ValidationError } from "@/index";
import { SyntheticCore, type SyntheticCoreProps } from "@/testutil";

// More points than any window here holds samples, so every sample draws.
const EXACT = 2 ** 20;

const RATE = Rate.hz(100);
const PERIOD = RATE.period.valueOf();
const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };

const OPTIONS: framer.FeedOptions = {
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

const raw = (keys: channel.Key[], span: TimeSpan): framer.SubscribeRequest => ({
  keys,
  window: dynamicWindow(span),
  limit: { variant: "points", count: EXACT, aggregation: "min_max" },
});

// Returns the series of the channel once the subscription loads them.
const loaded = async (
  sub: framer.Subscription,
  key: channel.Key,
): Promise<Series[]> => {
  await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
  return sub.value(key).series.series;
};

const timesOf = (snap: framer.Snapshot): bigint[] =>
  Array.from(snap.series) as bigint[];

// Waits for the stream of history changes that the feed opens for its first
// subscription.
const historyStreamed = async (core: SyntheticCore): Promise<void> =>
  await expect.poll(() => core.streams(core.historyChange.key), POLL).toBe(true);

// Opens the feed's stream of history changes, so reads after it fetch blocks that the
// changes can keep valid.
const listen = async (core: SyntheticCore, feed: framer.Feed): Promise<void> => {
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
  sub: framer.Subscription,
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
  sub: framer.Subscription,
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

  describe("calculated", () => {
    it("should draw the index of a calculated channel", async () => {
      const core = createCore({ derived: true });
      const feed = core.openFeed(OPTIONS);
      const key = core.calculatedIndex!.key;
      const span = TimeSpan.seconds(30);
      const sub = feed.subscribe(raw([key], span));
      const stop = tick(core);
      try {
        await settled(core, sub, key, span);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });
  });

  describe("history read failure", () => {
    // Returns the x series of the snapshot that holds each y series' alignments, with
    // the timestamps the y series pairs with.
    const pairsOf = (
      x: framer.Snapshot,
      y: framer.Snapshot,
    ): Array<[Series, bigint[]]> =>
      y.series.series.map((ys) => {
        const xs = x.series.series.find(
          (s) =>
            s.alignment <= ys.alignment &&
            s.alignment + BigInt(s.length) >= ys.alignment + BigInt(ys.length),
        );
        assert(xs != null);
        const at = Number(ys.alignment - xs.alignment);
        const times = Array.from({ length: ys.length }, (_, i) =>
          BigInt(xs.data[at + i]),
        );
        return [ys, times];
      });

    it("should draw streamed samples while the first history read runs", async () => {
      const core = createCore();
      let release = (): void => {};
      const released = new Promise<void>((resolve) => (release = resolve));
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async (tr, keys, reduction) => {
          await released;
          return core.read(tr, keys, reduction);
        },
      });
      const span = TimeSpan.seconds(30);
      const { key } = core.data[0];
      const { key: index } = core.index;
      const sub = feed.subscribe(raw([key, index], span));
      const stop = tick(core);
      try {
        await expect
          .poll(() => sub.value(key).series.length, POLL)
          .toBeGreaterThan(100);
        expect(sub.value(key).status).toMatchObject({
          variant: "loading",
          message: "Loading channel data",
        });
        const pairs = pairsOf(sub.value(index), sub.value(key));
        expect(pairs.length).toBeGreaterThan(0);
        pairs.forEach(([ys, times]) => {
          expect(times[0]).toBe(ys.timeRange.start.valueOf());
          expectComplete(times);
        });
        release();
        await settledData(core, sub, key, span);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should draw streamed samples while every history read fails", async () => {
      const core = createCore();
      let failing = true;
      // A retry redraws the line too, so retries wait out the checks of single frames.
      const interval = TimeSpan.seconds(2);
      const feed = core.openFeed({
        ...OPTIONS,
        breaker: { baseInterval: interval, maxInterval: interval },
        readRemote: async (tr, keys, reduction) => {
          if (failing) throw new Error("read failed");
          return core.read(tr, keys, reduction);
        },
      });
      const span = TimeSpan.seconds(30);
      const { key } = core.data[0];
      const { key: index } = core.index;
      const sub = feed.subscribe(raw([key, index], span));
      let stop = tick(core);
      try {
        await expect
          .poll(() => sub.value(key).series.length, POLL)
          .toBeGreaterThan(100);
        expect(sub.value(key).status).toMatchObject({
          variant: "error",
          message: "Failed to read channel data",
          description: "read failed",
        });
        const pairs = pairsOf(sub.value(index), sub.value(key));
        expect(pairs.length).toBeGreaterThan(0);
        pairs.forEach(([ys, times]) => {
          expect(times[0]).toBe(ys.timeRange.start.valueOf());
          expectComplete(times);
        });
        stop();
        // Each frame reaches the line before the window moves.
        for (let i = 0; i < 5; i++) {
          const newest = sub.value(key).series.series.at(-1);
          core.advance(RATE.period);
          await expect
            .poll(() => sub.value(key).series.series.at(-1)?.timeRange.end.valueOf(), {
              ...POLL,
              timeout: 500,
            })
            .toBe(newestOf(core) + 1n);
          // Each read returns the same series, so each uploads to the GPU once.
          expect(sub.value(key).series.series.at(-2)).toBe(newest);
        }
        expect(sub.value(key).time?.valueOf()).toBe(newestOf(core) + 1n);
        stop = tick(core);
        failing = false;
        await settledData(core, sub, key, span);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should place a channel's held samples by its joined index", async () => {
      const core = createCore();
      const { key } = core.data[0];
      const { key: index } = core.index;
      let failing = true;
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async (tr, keys, reduction) => {
          if (failing && keys.includes(key)) throw new Error("read failed");
          return core.read(tr, keys, reduction);
        },
      });
      const span = TimeSpan.seconds(30);
      const sub = feed.subscribe(raw([key, index], span));
      let stop = tick(core);
      try {
        await settled(core, sub, index, span);
        await expect
          .poll(() => sub.value(key).series.length, POLL)
          .toBeGreaterThan(100);
        const pairs = pairsOf(sub.value(index), sub.value(key));
        expect(pairs.length).toBeGreaterThan(0);
        pairs.forEach(([ys, times]) => {
          expect(domainOf(ys.alignment)).toBe(BigInt(rawIndex(ys.timeRange.start)));
          expect(times[0]).toBe(ys.timeRange.start.valueOf());
        });
        stop();
        const newest = sub.value(key).series.series.at(-1);
        core.advance(RATE.period);
        await expect
          .poll(
            () => sub.value(key).series.series.at(-1)?.timeRange.end.valueOf(),
            POLL,
          )
          .toBe(newestOf(core) + 1n);
        expect(sub.value(key).series.series.at(-2)).toBe(newest);
        stop = tick(core);
        failing = false;
        await settledData(core, sub, key, span);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should draw a held write into an earlier span in time order", async () => {
      const core = createCore();
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async () => {
          throw new Error("read failed");
        },
      });
      const span = TimeSpan.seconds(30);
      const { key: index } = core.index;
      const sub = feed.subscribe(raw([index], span));
      const stop = tick(core);
      try {
        await expect
          .poll(() => sub.value(index).series.length, POLL)
          .toBeGreaterThan(100);
        stop();
        // A writer fills an earlier span in a domain of its own.
        const at = sub.value(index).series.series[0].timeRange.start.sub(PERIOD);
        core.stream(
          new framer.Frame(
            index,
            new Series({
              data: new BigInt64Array([at.valueOf()]),
              dataType: DataType.TIMESTAMP,
              timeRange: new TimeRange(at, at.add(TimeSpan.NANOSECOND)),
              alignment: 9n << 32n,
            }),
          ),
        );
        await expect
          .poll(() => timesOf(sub.value(index)).includes(at.valueOf()), POLL)
          .toBe(true);
        expect(timesOf(sub.value(index))[0]).toBe(at.valueOf());
        expectComplete(timesOf(sub.value(index)));
        expect(sub.value(index).time?.valueOf()).toBe(newestOf(core) + 1n);
      } finally {
        stop();
        sub.close();
        await feed.close();
      }
    });

    it("should read a block once while the stream holds part of it", async () => {
      // The view opens shortly before a block edge, and the commit point passes the
      // edge while the catch-up read waits.
      const edge = tileRange({
        level: RAW_LEVEL,
        index: rawIndex(START.add(TimeSpan.minutes(10))),
      }).end;
      const core = createCore({ now: edge.sub(TimeSpan.seconds(5)) });
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async (tr, keys, reduction) => {
          // Only a catch-up read ends after the Core's clock.
          if (tr.end.after(core.now())) await released;
          return core.read(tr, keys, reduction);
        },
      });
      const span = TimeSpan.seconds(30);
      const { key: index } = core.index;
      const sub = feed.subscribe(raw([index], span));
      const stop = tick(core);
      try {
        const passed = edge.add(TimeSpan.seconds(2)).valueOf();
        await expect
          .poll(() => timesOf(sub.value(index)).at(-1), POLL)
          .toBeGreaterThan(passed);
        stop();
        await sleep.sleep(TimeSpan.milliseconds(100));
        const times = timesOf(sub.value(index));
        expect(times.every((t, i) => i === 0 || t > times[i - 1])).toBe(true);
      } finally {
        release();
        stop();
        sub.close();
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
      const span = TimeSpan.minutes(2);
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
      const span = TimeSpan.minutes(2);
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
      const sample = streamed.at(-2)!.at(-1, true) as bigint;
      const fresh = core.openFeed(OPTIONS);
      const read = fresh.subscribe(raw([core.index.key], span));
      try {
        const fetched = await loaded(read, core.index.key);
        const offsetOf = (series: Series[], t: bigint): bigint | undefined => {
          for (const s of series) {
            const at = (Array.from(s) as bigint[]).indexOf(t);
            if (at >= 0) return s.alignment + BigInt(at);
          }
          return undefined;
        };
        expect(offsetOf(streamed, sample)).toBeDefined();
        expect(offsetOf(fetched, sample)).toBe(offsetOf(streamed, sample));
      } finally {
        read.close();
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

    it("should pair a channel added after its index streams", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const span = TimeSpan.minutes(2);
      const { key: index } = core.index;
      const { key: data } = core.data[0];
      const x = feed.subscribe(raw([index], span));
      const stop = tick(core, TimeSpan.milliseconds(500));
      const y = feed.subscribe(raw([], span));
      try {
        await settled(core, x, index, span);
        const opened = core.now();
        await expect
          .poll(() => core.now().sub(opened).seconds, POLL)
          .toBeGreaterThan(30);
        y.update(raw([data], span));
        await expect.poll(() => y.value(data).status.variant, POLL).toBe("success");
        stop();
        await sleep.sleep(TimeSpan.milliseconds(100));
        const times = new Map<bigint, bigint>();
        x.value(index).series.series.forEach((s) => {
          for (let i = 0; i < s.length; i++)
            times.set(s.alignment + BigInt(i), BigInt(s.data[i]));
        });
        const ys = y.value(data).series;
        // The y samples span the history read before y opened and the stream after.
        expect(ys.length).toBeGreaterThan(RATE.sampleCount(span) / 2);
        // The Core writes sample n of the data channel as sin(n / 1000 + key).
        const unpaired = ys.series.flatMap((s) =>
          Array.from({ length: s.length }, (_, i) => s.alignment + BigInt(i)).filter(
            (a, i) => {
              const t = times.get(a);
              if (t == null) return true;
              const n = Number((t - START.valueOf()) / PERIOD);
              return Math.abs(Number(s.data[i]) - Math.sin(n / 1000 + data)) > 1e-5;
            },
          ),
        );
        expect(unpaired).toEqual([]);
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

    it("should draw samples stamped after the feed's clock", async () => {
      const core = createCore();
      const edge = tileRange({
        level: RAW_LEVEL,
        index: rawIndex(core.now()) + 1,
      }).start;
      core.advance(
        new TimeSpan(
          edge.valueOf() - core.now().valueOf() + TimeSpan.seconds(1).valueOf(),
        ),
      );
      // The writer's clock runs ahead of the Core's, so the newest samples sit in a raw
      // block after the one of now.
      const feed = core.openFeed({
        ...OPTIONS,
        now: () => core.now().sub(TimeSpan.seconds(2)),
      });
      const key = core.index.key;
      const sub = feed.subscribe(raw([key], TimeSpan.seconds(30)));
      try {
        await expect
          .poll(() => timesOf(sub.value(key)).at(-1), POLL)
          .toBe(newestOf(core));
      } finally {
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
        limit: { variant: "points", count: EXACT, aggregation: "min_max" },
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
        limit: { variant: "points", count: EXACT, aggregation: "min_max" },
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
    it("should keep its tiles while the Core is down, and read them again after", async () => {
      const core = createCore();
      // The stream warns on each failed attempt to open again.
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const sub = feed.subscribe({
        keys: [key],
        window: staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(5)))),
        limit: { variant: "points", count: 100, aggregation: "min_max" },
      });
      // A consumer reads a value on every change, as Pluto does.
      sub.onChange(() => sub.value(key));
      try {
        await historyStreamed(core);
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        const { series } = sub.value(key);
        const reads = core.reads;
        core.disconnect();
        await expect
          .poll(() => warn.mock.calls.at(-1)?.[0], POLL)
          .toMatchObject({ message: "failed to open streamer" });
        expect(core.reads).toBe(reads);
        expect(sub.value(key)).toMatchObject({
          series,
          status: { variant: "success" },
        });
        core.reconnect();
        await expect.poll(() => core.reads, POLL).toBeGreaterThan(reads);
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
      } finally {
        sub.close();
        await feed.close();
      }
    });

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

    it("should read raw samples after an outage only near the live edge", async () => {
      const core = createCore({ rate: Rate.hz(10), now: START.add(TimeSpan.days(2)) });
      const raw: TimeRange[] = [];
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async (tr, keys, reduction) => {
          if (reduction == null) raw.push(tr);
          return core.read(tr, keys, reduction);
        },
      });
      const key = core.data[0].key;
      const sub = feed.subscribe({
        keys: [key],
        window: dynamicWindow(TimeSpan.days(1)),
        limit: { variant: "points", count: 100, aggregation: "min_max" },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        core.disconnect();
        await expect
          .poll(() => sub.value(key).status, POLL)
          .toMatchObject({ variant: "warning", message: "reconnecting" });
        core.advance(TimeSpan.hours(6));
        const reads = raw.length;
        core.reconnect();
        // Only a catch-up read ends after the Core's clock.
        await expect
          .poll(() => raw.slice(reads).some((tr) => tr.end.after(core.now())), POLL)
          .toBe(true);
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        const edge = core.now().sub(tileSpan(RAW_LEVEL).mult(2));
        raw
          .slice(reads)
          .forEach((tr) =>
            expect(tr.start.valueOf()).toBeGreaterThanOrEqual(edge.valueOf()),
          );
      } finally {
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

    it("should count no streamed sample again after the stream reconnects", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.index.key;
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 1000 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        core.advance(TimeSpan.milliseconds(500));
        await expect
          .poll(() => timesOf(sub.value(key)).at(-1), POLL)
          .toBe(newestOf(core));
        const { appended } = sub.value(key);
        assert(appended != null);
        core.disconnect();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("warning");
        core.advance(TimeSpan.seconds(1));
        core.reconnect();
        await expect
          .poll(() => timesOf(sub.value(key)).at(-1), POLL)
          .toBe(newestOf(core));
        expectComplete(timesOf(sub.value(key)));
        expect(sub.value(key).appended).toBe(appended + 100);
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
        core.disconnect();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("warning");
        core.advance(TimeSpan.seconds(1));
        core.reconnect();
        await expect
          .poll(() => timesOf(sub.value(key)).at(-1), POLL)
          .toBe(newestOf(core));
        const snap = sub.value(key);
        expect(snap.status.variant).toBe("success");
        expectComplete(timesOf(snap));
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should read the newest samples again when an outage rewrites them", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const { key } = core.data[0];
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 10 },
      });
      const valuesOf = (): number[] => Array.from(sub.value(key).series) as number[];
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        await historyStreamed(core);
        const { appended } = sub.value(key);
        core.disconnect();
        // The stream misses the history change, so only the reconnect can find it.
        core.rewrite(key, new TimeRange(START, core.now()), 10);
        core.reconnect();
        const stored = Array.from(core.readLatest([key], 10).get(key)) as number[];
        await expect.poll(valuesOf, POLL).toEqual(stored);
        expect(sub.value(key).appended).toBe(appended);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should keep the samples an outage skips when it reads the newest again", async () => {
      const core = createCore();
      let held = false;
      let opens = 0;
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      const feed = core.openFeed({
        ...OPTIONS,
        readLatest: async (keys, count, tr) => {
          if (held) await released;
          return core.readLatest(keys, count, tr);
        },
        openStreamer: async (config) => {
          opens++;
          return core.openStreamer(config);
        },
      });
      const { key } = core.data[0];
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 10 },
      });
      const valuesOf = (): number[] =>
        (Array.from(sub.value(key).series) as number[]).slice(-10);
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        await historyStreamed(core);
        const { appended } = sub.value(key);
        assert(appended != null);
        const before = opens;
        held = true;
        core.disconnect();
        core.rewrite(key, new TimeRange(START, core.now()), 10);
        core.advance(TimeSpan.milliseconds(30), false);
        core.reconnect();
        await expect.poll(() => opens, POLL).toBe(before + 1);
        // Samples stream after the reconnect while the read of the newest waits.
        core.advance(TimeSpan.milliseconds(30));
        await sleep.sleep(TimeSpan.milliseconds(50));
        held = false;
        release();
        const stored = Array.from(core.readLatest([key], 10).get(key)) as number[];
        await expect.poll(valuesOf, POLL).toEqual(stored);
        // 30 ms at 100 Hz during the outage, and again after it.
        expect(sub.value(key).appended).toBe(appended + 6);
      } finally {
        release();
        sub.close();
        await feed.close();
      }
    });

    it("should land a read before the samples streamed after a stream drop", async () => {
      const core = createCore();
      let held = false;
      let waiting = false;
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      const feed = core.openFeed({
        ...OPTIONS,
        readLatest: async (keys, count, tr) => {
          if (held) {
            waiting = true;
            await released;
          }
          return core.readLatest(keys, count, tr);
        },
      });
      const key = core.index.key;
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 1000 },
      });
      // A consumer reads a value on every change, as Pluto does.
      sub.onChange(() => sub.value(key));
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        held = true;
        core.disconnect();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("warning");
        // The stream never delivers the samples written while it is down.
        core.advance(TimeSpan.seconds(1));
        core.reconnect();
        await expect.poll(() => waiting, POLL).toBe(true);
        core.advance(TimeSpan.milliseconds(100));
        await expect
          .poll(() => timesOf(sub.value(key)).at(-1), POLL)
          .toBe(newestOf(core));
        release();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        expectComplete(timesOf(sub.value(key)));
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
        core.stream(new framer.Frame(key, older));
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

    it("should read the newest samples again when a history change reaches them", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const { key } = core.data[0];
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 10 },
      });
      const valuesOf = (): number[] => Array.from(sub.value(key).series) as number[];
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        const before = valuesOf();
        await historyStreamed(core);
        const reads = core.reads;
        // A change before every sample held cannot reach the newest ten.
        core.rewrite(key, new TimeRange(START, START.add(TimeSpan.minutes(1))), 10);
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(core.reads).toBe(reads);
        core.rewrite(key, new TimeRange(START, core.now()), 10);
        await expect
          .poll(() => valuesOf().at(-1), POLL)
          .toBeCloseTo(before.at(-1)! + 10);
        expect(valuesOf()).toHaveLength(10);
      } finally {
        sub.close();
        await feed.close();
      }
    });

    it("should count only the samples streamed since a history change", async () => {
      const core = createCore();
      let held = false;
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      const feed = core.openFeed({
        ...OPTIONS,
        readLatest: async (keys, count, tr) => {
          if (held) await released;
          return core.readLatest(keys, count, tr);
        },
      });
      const { key } = core.data[0];
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 20 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        await historyStreamed(core);
        const { appended } = sub.value(key);
        assert(appended != null);
        held = true;
        core.rewrite(key, new TimeRange(START, core.now()), 10);
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("loading");
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => sub.value(key).appended, POLL).toBe(appended + 10);
        release();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        await expect.poll(() => sub.value(key).appended, POLL).toBe(appended + 10);
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(sub.value(key).appended).toBe(appended + 10);
      } finally {
        release();
        sub.close();
        await feed.close();
      }
    });

    it("should keep the bound of a waiting read across a second history change", async () => {
      const core = createCore();
      let held = false;
      let reads = 0;
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      const feed = core.openFeed({
        ...OPTIONS,
        readLatest: async (keys, count, tr) => {
          reads++;
          if (held) await released;
          return core.readLatest(keys, count, tr);
        },
      });
      const { key } = core.data[0];
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 20 },
      });
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        await historyStreamed(core);
        const { appended } = sub.value(key);
        assert(appended != null);
        held = true;
        const tr = new TimeRange(START, core.now());
        core.rewrite(key, tr, 10);
        await expect.poll(() => reads, POLL).toBe(2);
        core.rewrite(key, tr, 10);
        await expect.poll(() => reads, POLL).toBe(3);
        release();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        expect(sub.value(key).appended).toBe(appended);
      } finally {
        release();
        sub.close();
        await feed.close();
      }
    });

    it("should count its first read when a history change interrupts it", async () => {
      const core = createCore();
      let reads = 0;
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      const feed = core.openFeed({
        ...OPTIONS,
        readLatest: async (keys, count, tr) => {
          reads++;
          await released;
          return core.readLatest(keys, count, tr);
        },
      });
      const { key } = core.data[0];
      const sub = feed.subscribe({
        keys: [key],
        limit: { variant: "latest", count: 20 },
      });
      try {
        await expect.poll(() => reads, POLL).toBe(1);
        await historyStreamed(core);
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => sub.value(key).appended, POLL).toBe(10);
        core.rewrite(key, new TimeRange(START, core.now()), 10);
        await expect.poll(() => reads, POLL).toBe(2);
        release();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        expect(sub.value(key).appended).toBe(20);
      } finally {
        release();
        sub.close();
        await feed.close();
      }
    });

    it("should count the read of an update when a history change interrupts it", async () => {
      const core = createCore();
      let held = false;
      let reads = 0;
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      const feed = core.openFeed({
        ...OPTIONS,
        readLatest: async (keys, count, tr) => {
          reads++;
          if (held) await released;
          return core.readLatest(keys, count, tr);
        },
      });
      const { key } = core.data[0];
      const request = (count: number): framer.SubscribeRequest => ({
        keys: [key],
        limit: { variant: "latest", count },
      });
      const sub = feed.subscribe(request(10));
      try {
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        await historyStreamed(core);
        const { appended } = sub.value(key);
        assert(appended != null);
        held = true;
        sub.update(request(20));
        await expect.poll(() => reads, POLL).toBe(2);
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => sub.value(key).appended, POLL).toBe(appended + 10);
        core.rewrite(key, new TimeRange(START, core.now()), 10);
        await expect.poll(() => reads, POLL).toBe(3);
        release();
        await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
        expect(sub.value(key).appended).toBe(appended + 20);
      } finally {
        release();
        sub.close();
        await feed.close();
      }
    });
  });

  describe("update and status", () => {
    const createStatic = (key: channel.Key): framer.PointsRequest => ({
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

    it.each([Infinity, NaN, -1])("should reject a point count of %s", async (count) => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const request = createStatic(core.data[0].key);
      const invalid = { ...request, limit: { ...request.limit, count } };
      const error = new ValidationError(
        `points limit count ${count} must be finite and not negative`,
      );
      const sub = feed.subscribe(request);
      try {
        expect(() => feed.subscribe(invalid)).toThrow(error);
        expect(() => sub.update(invalid)).toThrow(error);
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
    // Two whole tiles, which the feed keeps after a load.
    const tiles = tileRange({ level: 15, index: tileIndex(START, 15) + 1 });

    it("should drop the samples a history change names, and keep the rest", async () => {
      const core = createCore();
      const feed = core.openFeed(OPTIONS);
      const key = core.data[0].key;
      const window = staticWindow(new TimeRange(START, START.add(TimeSpan.minutes(4))));
      const sub = feed.subscribe({
        keys: [key],
        window,
        limit: { variant: "points", count: EXACT, aggregation: "min_max" },
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
      const window = staticWindow(tiles);
      const load = async (): Promise<number> => {
        const reads = core.reads;
        const sub = feed.subscribe({
          keys: [key],
          window,
          limit: { variant: "points", count: EXACT, aggregation: "min_max" },
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
        const reads = await load();
        expect(reads).toBeGreaterThan(0);
        expect(await load()).toBe(0);
        core.advance(TimeSpan.minutes(20));
        expect(await load()).toBe(reads);
      } finally {
        await feed.close();
      }
    });

    it("should keep signaled blocks past the expiry, and expire calculated ones", async () => {
      const core = createCore({ derived: true });
      const feed = core.openFeed(OPTIONS);
      const window = staticWindow(tiles);
      // Returns the Core reads that a subscription to the key's window makes.
      const load = async (key: channel.Key): Promise<number> => {
        const reads = core.reads;
        const sub = feed.subscribe({
          keys: [key],
          window,
          limit: { variant: "points", count: EXACT, aggregation: "min_max" },
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
        const reads = await load(data);
        expect(reads).toBeGreaterThan(0);
        await historyStreamed(core);
        core.rewrite(data, tiles, 0);
        core.rewrite(calculated, tiles, 0);
        await sleep.sleep(TimeSpan.milliseconds(50));
        expect(await load(data)).toBe(reads);
        expect(await load(calculated)).toBe(reads);
        core.advance(TimeSpan.minutes(20));
        expect(await load(data)).toBe(0);
        expect(await load(calculated)).toBe(reads);
      } finally {
        await feed.close();
      }
    });

    it.each([
      ["raw samples", EXACT],
      ["tiles", 100],
    ])(
      "should read %s again once the blocks that appends land in expire",
      async (_, count) => {
        // The writer trails the clock, so appends land in blocks the feed read empty.
        const core = createCore({ lag: TimeSpan.minutes(5) });
        const feed = core.openFeed(OPTIONS);
        const key = core.data[0].key;
        const request: framer.PointsRequest = {
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
        const load = async (f: framer.Feed): Promise<number[]> => {
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
    // Opens a feed whose first read that held accepts waits for release, after it
    // computed its samples. Reads counts the reads that held accepts.
    const createHeld = (
      core: SyntheticCore,
      held: (tr: TimeRange, keys: channel.Key[]) => boolean,
    ) => {
      let release: () => void = () => {};
      const released = new Promise<void>((resolve) => (release = resolve));
      let reads = 0;
      const feed = core.openFeed({
        ...OPTIONS,
        readRemote: async (tr, keys, reduction) => {
          const frame = core.read(tr, keys, reduction);
          if (held(tr, keys) && reads++ === 0) await released;
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
      const span = TimeSpan.minutes(2);
      const changed = new TimeRange(
        tileRange({ level: RAW_LEVEL, index: rawIndex(core.now().sub(span)) }).start,
        core.now(),
      );
      const { feed, release, reads } = createHeld(
        core,
        (tr, keys) => keys.includes(key) && tr.start.equals(changed.start),
      );
      const sub = feed.subscribe(raw([key], span));
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
      ["raw samples", EXACT],
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
      const { feed, release, reads } = createHeld(core, (_, keys) =>
        keys.includes(key),
      );
      const request: framer.SubscribeRequest = {
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
