// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import {
  dynamicWindow,
  id,
  Rate,
  sleep,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";
import { afterAll, describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { type framer } from "@/framer";
import { UnexpectedError } from "@/index";
import { createTestClient, SyntheticCore } from "@/testutil";

// More points than any window here holds samples, so every sample draws.
const EXACT = 2 ** 20;

const client = createTestClient();
const feed = client.openFeed();
afterAll(async () => await feed.close());

const POLL = { timeout: 10000, interval: 50 };
const START = TimeStamp.seconds(1_000_000);

const OPTIONS: framer.FeedOptions = {
  batchDebounce: TimeSpan.milliseconds(5),
  removalDelay: TimeSpan.ZERO,
  breaker: {
    baseInterval: TimeSpan.milliseconds(10),
    maxInterval: TimeSpan.milliseconds(50),
  },
};

const createCore = (): SyntheticCore =>
  new SyntheticCore({
    rate: Rate.hz(100),
    start: START,
    now: START.add(TimeSpan.minutes(10)),
    channels: 2,
  });

/**
 * Returns a source of strictly increasing timestamps. `TimeStamp.now()` resolves to the
 * millisecond, so a tight write loop repeats one, and an index channel must never
 * repeat a timestamp.
 */
const createClock = (): (() => TimeStamp) => {
  let last = TimeStamp.now();
  return () => {
    const now = TimeStamp.now();
    last = now.after(last) ? now : last.add(TimeSpan.microseconds(1));
    return last;
  };
};

const raw = (keys: channel.Key[], window: Window): framer.PointsRequest => ({
  keys,
  window,
  limit: { variant: "points", count: EXACT, aggregation: "min_max" },
});

const latest = (keys: channel.Key[], count = 1): framer.LatestRequest => ({
  keys,
  limit: { variant: "latest", count },
});

const valuesOf = (sub: framer.Subscription, key: channel.Key): number[] =>
  Array.from(sub.value(key).series) as number[];

/** Returns the samples of the channel once the subscription loads them. */
const loaded = async (
  sub: framer.Subscription,
  key: channel.Key,
): Promise<number[]> => {
  await expect.poll(() => sub.value(key).status.variant, POLL).toBe("success");
  return valuesOf(sub, key);
};

/** @returns true when values hold every number below count, each once. */
const holdsWritten = (values: number[], count: number): boolean =>
  values.length === count && [...values].sort((a, b) => a - b).every((v, i) => v === i);

const createChannels = async () => {
  const time = await client.channels.create({
    name: id.create(),
    dataType: "timestamp",
    isIndex: true,
  });
  const data = await client.channels.create({
    name: id.create(),
    dataType: "float32",
    index: time.key,
  });
  return { time, data };
};

describe("Feed", () => {
  it("should serve the written samples of a static window", async () => {
    const { time, data } = await createChannels();
    const start = TimeStamp.now();
    await client.write(start, {
      [time.key]: [start, start.add(TimeSpan.milliseconds(1))],
      [data.key]: [1, 2],
    });
    const tr = new TimeRange(start, start.add(TimeSpan.seconds(1)));
    const sub = feed.subscribe(raw([data.key], staticWindow(tr)));
    try {
      expect(await loaded(sub, data.key)).toEqual([1, 2]);
    } finally {
      sub.close();
    }
  });

  it("should serve frames written after it subscribes", async () => {
    const { time, data } = await createChannels();
    const sub = feed.subscribe(latest([data.key], 100));
    // The writer must stream as well as persist: the plain write convenience is
    // persist-only and the relay never broadcasts it.
    const next = createClock();
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: [time.key, data.key],
    });
    try {
      // The stream converges to the demand in the background, so writes repeat until
      // one lands on the open stream.
      await expect
        .poll(
          async () => {
            await writer.write({ [time.key]: [next()], [data.key]: [42] });
            return valuesOf(sub, data.key).includes(42);
          },
          { timeout: 10000, interval: 250 },
        )
        .toBe(true);
    } finally {
      await writer.close();
      sub.close();
    }
  });

  it("should hold each sample once while a writer runs", async () => {
    const { time, data } = await createChannels();
    const start = TimeStamp.now();
    const writer = await client.openWriter({ start, channels: [time.key, data.key] });
    let written = 0;
    let running = true;
    // A 1 kHz writer: ten samples a millisecond apart, paced to the wall clock. Each
    // data sample holds its sample number.
    const writing = (async () => {
      while (running) {
        const numbers = Array.from({ length: 10 }, (_, i) => written + i);
        await writer.write({
          [time.key]: new BigInt64Array(
            numbers.map((n) => start.add(TimeSpan.milliseconds(n)).valueOf()),
          ),
          [data.key]: new Float32Array(numbers),
        });
        written += 10;
        const next = start.add(TimeSpan.milliseconds(written));
        if (next.after(TimeStamp.now())) await sleep.sleep(next.span(TimeStamp.now()));
      }
    })();
    const seamFeed = client.openFeed({ removalDelay: TimeSpan.ZERO });
    const span = TimeSpan.seconds(2);
    try {
      for (let i = 0; i < 20; i++) {
        const sub = seamFeed.subscribe(raw([data.key], dynamicWindow(span)));
        try {
          await expect
            .poll(
              () => {
                const snap = sub.value(data.key);
                return snap.status.variant === "success" && snap.series.length > 0;
              },
              { timeout: 5000, interval: 5 },
            )
            .toBe(true);
          await sleep.sleep(TimeSpan.milliseconds(50 + 10 * i));
          const opened = TimeStamp.now().sub(span);
          const numbers = valuesOf(sub, data.key);
          const steps = new Set(numbers.slice(1).map((n, j) => n - numbers[j]));
          expect([...steps]).toEqual([1]);
          const from = TimeStamp.max(start, opened);
          expect(numbers[0]).toBeLessThanOrEqual(
            Number(from.span(start).milliseconds) + 1,
          );
        } finally {
          sub.close();
        }
      }
    } finally {
      running = false;
      await writing;
      await writer.close();
      await seamFeed.close();
    }
  });

  it("should read samples written while a channel was unstreamed", async () => {
    const { time, data } = await createChannels();
    const short = client.openFeed({ removalDelay: TimeSpan.milliseconds(100) });
    try {
      const start = TimeStamp.now();
      const next = createClock();
      const w = await client.openWriter({ start, channels: [time.key, data.key] });
      let value = 0;
      try {
        const live = short.subscribe(
          raw([data.key], dynamicWindow(TimeSpan.minutes(1))),
        );
        await expect
          .poll(
            async () => {
              await w.write({ [time.key]: [next()], [data.key]: [value++] });
              return valuesOf(live, data.key).length > 0;
            },
            { timeout: 10000, interval: 100 },
          )
          .toBe(true);
        // Ends the channel's streaming demand. The wait outlasts the removal delay and
        // the reconcile that shrinks the stream.
        live.close();
        await sleep.sleep(TimeSpan.milliseconds(500));
        for (let i = 0; i < 10; i++)
          await w.write({ [time.key]: [next()], [data.key]: [value++] });
        await w.commit();
      } finally {
        await w.close();
      }
      const tr = new TimeRange(start, TimeStamp.now());
      const sub = short.subscribe(raw([data.key], staticWindow(tr)));
      try {
        expect(holdsWritten(await loaded(sub, data.key), value)).toBe(true);
      } finally {
        sub.close();
      }
    } finally {
      await short.close();
    }
  });

  it("should keep far-past streamed samples out of a present window", async () => {
    const { time, data } = await createChannels();
    const live = feed.subscribe(latest([data.key]));
    // Epoch-anchored stamps: the incident's Arc defect streamed samples whose index
    // timestamps sat decades in the past while writes marched forward in real time.
    let epoch = TimeStamp.seconds(10);
    const writer = await client.openWriter({
      start: epoch,
      channels: [time.key, data.key],
    });
    try {
      await expect
        .poll(
          async () => {
            epoch = epoch.add(TimeSpan.milliseconds(1));
            await writer.write({ [time.key]: [epoch], [data.key]: [1] });
            return valuesOf(live, data.key).length > 0;
          },
          { timeout: 10000, interval: 100 },
        )
        .toBe(true);
    } finally {
      await writer.close();
    }
    const now = TimeStamp.now();
    const recent = feed.subscribe(
      raw([data.key], staticWindow(new TimeRange(now.sub(TimeSpan.seconds(30)), now))),
    );
    const past = feed.subscribe(
      raw(
        [data.key],
        staticWindow(new TimeRange(TimeStamp.ZERO, TimeStamp.seconds(60))),
      ),
    );
    try {
      // The live buffer holds only epoch-era samples, so a window of the recent past
      // must come back empty instead of serving them.
      expect(await loaded(recent, data.key)).toEqual([]);
      expect((await loaded(past, data.key)).length).toBeGreaterThan(0);
    } finally {
      live.close();
      recent.close();
      past.close();
    }
  });

  it("should give a calculated channel and its index one block alignment", async () => {
    const time = await client.channels.create({
      name: id.create(),
      dataType: "timestamp",
      isIndex: true,
    });
    const data = await client.channels.create({
      name: id.create(),
      dataType: "float64",
      index: time.key,
    });
    const calc = await client.channels.create({
      name: id.create(),
      dataType: "float64",
      virtual: true,
      expression: `return ${data.name} * 2`,
    });
    const start = TimeStamp.now().sub(TimeSpan.seconds(10));
    const stamps = [0, 1, 2].map((i) => start.add(TimeSpan.milliseconds(i)));
    await client.write(start, { [time.key]: stamps, [data.key]: [1, 2, 3] });
    const tr = new TimeRange(start, start.add(TimeSpan.seconds(1)));
    const sub = feed.subscribe(raw([calc.key, calc.index], staticWindow(tr)));
    try {
      expect(await loaded(sub, calc.key)).toEqual([2, 4, 6]);
      expect(await loaded(sub, calc.index)).toEqual(stamps.map((t) => t.valueOf()));
      const bounds = (key: channel.Key) =>
        sub.value(key).series.series.map((s) => s.alignmentBounds);
      expect(bounds(calc.key)).toEqual(bounds(calc.index));
    } finally {
      sub.close();
    }
  });

  it("should throw when a closed feed subscribes", async () => {
    const closed = client.openFeed();
    await closed.close();
    expect(() => closed.subscribe(latest([1]))).toThrow(
      new UnexpectedError("telemetry feed is closed"),
    );
  });

  it("should forward fetchTimeout to the reader", async () => {
    const core = createCore();
    const direct = core.openFeed({
      ...OPTIONS,
      fetchTimeout: TimeSpan.milliseconds(20),
      readRemote: async () => await new Promise<framer.Frame>(() => {}),
    });
    const key = core.index.key;
    const tr = new TimeRange(START, START.add(TimeSpan.seconds(1)));
    const sub = direct.subscribe(raw([key], staticWindow(tr)));
    try {
      await expect.poll(() => sub.value(key).status.variant, POLL).toBe("error");
      expect(sub.value(key).status).toMatchObject({
        message: "Failed to read channel data",
        description: expect.stringContaining("timed out after 20ms"),
      });
    } finally {
      sub.close();
      await direct.close();
    }
  });

  it("should close while a stream open is still retrying", async () => {
    let opens = 0;
    const core = createCore();
    const direct = core.openFeed({
      ...OPTIONS,
      openStreamer: async () => {
        opens++;
        throw new Unreachable();
      },
      onRetry: () => {},
    });
    direct.subscribe(latest([core.index.key]));
    await expect.poll(() => opens).toBeGreaterThan(0);
    const closed = direct.close().then(() => true);
    const timedOut = sleep.sleep(TimeSpan.seconds(1)).then(() => false);
    expect(await Promise.race([closed, timedOut])).toBe(true);
  });

  describe("latest reads", () => {
    it("should batch the reads of subscriptions that load together", async () => {
      const core = createCore();
      const calls: channel.Key[][] = [];
      const direct = core.openFeed({
        ...OPTIONS,
        readLatest: async (keys, count, tr) => {
          calls.push(keys);
          return core.readLatest(keys, count, tr);
        },
      });
      const [a, b] = core.data.map((ch) => ch.key);
      const subs = [a, b, a].map((key) => ({
        key,
        sub: direct.subscribe(latest([key])),
      }));
      try {
        await expect
          .poll(
            () =>
              subs.every(({ key, sub }) => sub.value(key).status.variant === "success"),
            POLL,
          )
          .toBe(true);
        expect(calls).toEqual([[a, b]]);
        const later = direct.subscribe(latest([b]));
        subs.push({ key: b, sub: later });
        await loaded(later, b);
        expect(calls).toEqual([[a, b], [b]]);
      } finally {
        subs.forEach(({ sub }) => sub.close());
        await direct.close();
      }
    });

    it("should fail every subscription of a failed batch", async () => {
      const core = createCore();
      const direct = core.openFeed({
        ...OPTIONS,
        readLatest: async () => {
          throw new Error("iterator exploded");
        },
      });
      const keys = core.data.map((ch) => ch.key);
      const subs = keys.map((key) => direct.subscribe(latest([key])));
      try {
        await expect
          .poll(() => subs.map((sub, i) => sub.value(keys[i]).status.variant), POLL)
          .toEqual(["error", "error"]);
        subs.forEach((sub, i) =>
          expect(sub.value(keys[i]).status).toMatchObject({
            message: "Failed to read channel data",
            description: "iterator exploded",
          }),
        );
      } finally {
        subs.forEach((sub) => sub.close());
        await direct.close();
      }
    });
  });
});
