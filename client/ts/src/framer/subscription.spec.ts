// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DataType,
  dynamicWindow,
  id,
  Rate,
  type Series,
  Size,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { rawIndex } from "@/framer/cache/block";
import { tileRange } from "@/framer/cache/grid";
import { type FeedOptions } from "@/framer/feed";
import {
  type Snapshot,
  type SubscribeRequest,
  type Subscription,
} from "@/framer/subscription";
import { createTestClient, SyntheticCore, type SyntheticCoreProps } from "@/testutil";

const RATE = Rate.hz(100);
const PERIOD = RATE.period.valueOf();
const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };
const MASK = (1n << 32n) - 1n;

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
// frames once.
const settled = async (
  core: SyntheticCore,
  sub: Subscription,
  key: channel.Key,
  span: TimeSpan,
): Promise<bigint[]> => {
  let times: bigint[] = [];
  await expect
    .poll(() => {
      const start = core.now().sub(span).valueOf();
      const recent = core.now().sub(TimeSpan.milliseconds(500)).valueOf();
      const snap = sub.value(key);
      times = timesOf(snap);
      if (snap.status.variant !== "success" || times.length < 2) return false;
      const steps = times.slice(1).every((t, i) => t - times[i] === PERIOD);
      return steps && times[0] <= start + PERIOD && times.at(-1)! >= recent;
    }, POLL)
    .toBe(true);
  return times;
};

describe("Feed.subscribe", () => {
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
          const block = Number(s.alignment >> 32n);
          const first = new TimeStamp(s.at(0, true) as bigint);
          const last = new TimeStamp(s.at(-1, true) as bigint);
          expect(rawIndex(first)).toBe(block);
          expect(rawIndex(last)).toBe(block);
          const prev = series.series[i - 1];
          if (prev == null) return;
          if (prev.alignment >> 32n === s.alignment >> 32n)
            expect(s.alignment).toBe(prev.alignment + BigInt(prev.length));
          else expect(s.alignment & MASK).toBe(0n);
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
      const block = Number(streamed.at(-2)!.alignment >> 32n);
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
        await expect
          .poll(() => {
            const types = new Set(
              sub.value(key).series.series.map((s) => s.dataType.toString()),
            );
            expect(types.size).toBeLessThanOrEqual(1);
            return [...types];
          }, POLL)
          .toEqual([DataType.FLOAT64.toString()]);
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
        const newest = (): bigint =>
          START.valueOf() +
          BigInt(
            Math.ceil(Number(core.now().sub(START).valueOf()) / Number(PERIOD)) - 1,
          ) *
            PERIOD;
        expect(sub.value(key).series.at(-1)).toBe(newest());
        expect(sub.value(key).series.length).toBe(1);
        core.advance(TimeSpan.seconds(1));
        await expect.poll(() => sub.value(key).series.at(-1), POLL).toBe(newest());
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
        expect(sub.value(key).series.length).toBe(20);
      } finally {
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
        await sleep.sleep(TimeSpan.milliseconds(150));
        core.advance(TimeSpan.milliseconds(100));
        await expect.poll(() => sub.value(key).series.length, POLL).toBe(10);
        expect(core.reads).toBe(reads);
      } finally {
        sub.close();
        await feed.close();
      }
    });
  });

  describe("update and status", () => {
    const createStatic = (key: channel.Key): SubscribeRequest => ({
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
        sub.update({ ...request, limit: { ...request.limit, view } as never });
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
        await sleep.sleep(TimeSpan.milliseconds(150));
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
        // Blocks fetched once the history changes stream are valid until one names them.
        await load(data);
        await sleep.sleep(TimeSpan.milliseconds(150));
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
  });

  describe("seam", () => {
    it("should hold each sample once from the window start while a writer runs", async () => {
      const client = createTestClient();
      const index = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const start = TimeStamp.now();
      const writer = await client.openWriter({ start, channels: [index.key] });
      let written = 0;
      let running = true;
      // A 1 kHz writer: ten samples a millisecond apart, paced to the wall clock.
      const writing = (async () => {
        while (running) {
          const times = Array.from({ length: 10 }, (_, i) =>
            start.add(TimeSpan.milliseconds(written + i)).valueOf(),
          );
          await writer.write(index.key, new BigInt64Array(times));
          written += 10;
          const ahead = start.add(TimeSpan.milliseconds(written)).span(TimeStamp.now());
          if (start.add(TimeSpan.milliseconds(written)).after(TimeStamp.now()))
            await sleep.sleep(ahead);
        }
      })();
      const feed = client.openFeed();
      const span = TimeSpan.seconds(2);
      try {
        for (let i = 0; i < 20; i++) {
          const sub = feed.subscribe(raw([index.key], span));
          try {
            await expect
              .poll(() => {
                const snap = sub.value(index.key);
                return snap.status.variant === "success" && snap.series.length > 0;
              }, POLL)
              .toBe(true);
            await sleep.sleep(TimeSpan.milliseconds(50 + 10 * i));
            const opened = TimeStamp.now().sub(span);
            const times = timesOf(sub.value(index.key));
            expect(new Set(times).size).toBe(times.length);
            const steps = new Set(times.slice(1).map((t, j) => t - times[j]));
            expect([...steps]).toEqual([TimeSpan.MILLISECOND.valueOf()]);
            const from = TimeStamp.max(start, opened).valueOf();
            expect(times[0] <= from + TimeSpan.milliseconds(1).valueOf()).toBe(true);
          } finally {
            sub.close();
          }
        }
      } finally {
        running = false;
        await writing;
        await writer.close();
        await feed.close();
      }
    });
  });
});
