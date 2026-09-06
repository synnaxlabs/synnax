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
  id,
  Series,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterAll, describe, expect, it } from "vitest";

import { UnexpectedError } from "@/errors";
import { type Transform } from "@/framer/cache/transform";
import { Feed } from "@/framer/feed";
import { Frame } from "@/framer/frame";
import { createTestClient } from "@/testutil";

const client = createTestClient();
const feed = client.openFeed();
afterAll(async () => await feed.close());

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

/**
 * Asserts a read returned every value written so far and nothing else. A span crossing
 * the live boundary can come back in both its streamed and its fetched representation,
 * so a value may appear twice, but never more.
 */
const expectWrittenValues = (values: number[], writeCount: number): void => {
  const counts = new Map<number, number>();
  values.forEach((v) => counts.set(v, (counts.get(v) ?? 0) + 1));
  expect([...counts.keys()].sort((a, b) => a - b)).toEqual(
    Array.from({ length: writeCount }, (_, i) => i),
  );
  expect([...counts.entries()].filter(([, count]) => count > 2)).toEqual([]);
};

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

describe("feed", () => {
  it("should read written samples through the feed cache", async () => {
    const { time, data } = await createChannels();
    const start = TimeStamp.now();
    await client.write(start, {
      [time.key]: [start, start.add(TimeSpan.milliseconds(1))],
      [data.key]: [1, 2],
    });
    const tr = new TimeRange(start, start.add(TimeSpan.seconds(1)));
    const res = await feed.read(tr, data.key);
    expect(Array.from(res)).toEqual([1, 2]);
  });

  it("should apply the configured transform to cached series", async () => {
    const transform: Transform = {
      resolveDataType: () => DataType.FLOAT64,
      convert: (series) =>
        new Series({
          data: new Float64Array((Array.from(series) as number[]).map((v) => v * 2)),
          timeRange: series.timeRange,
          alignment: series.alignment,
        }),
    };
    const transformed = client.openFeed({ transform });
    const { time, data } = await createChannels();
    const start = TimeStamp.now();
    await client.write(start, { [time.key]: [start], [data.key]: [3] });
    const tr = new TimeRange(start, start.add(TimeSpan.seconds(1)));
    const res = await transformed.read(tr, data.key);
    await transformed.close();
    expect(res.dataType.equals(DataType.FLOAT64)).toBe(true);
    expect(Array.from(res)).toEqual([6]);
  });

  it("should deliver written frames to a subscribed stream handler", async () => {
    const { time, data } = await createChannels();
    const received: number[] = [];
    const sub = feed.stream(
      (res) => {
        const series = res.get(data.key);
        if (series != null) received.push(...(Array.from(series) as number[]));
      },
      [data.key],
    );
    // The writer must stream as well as persist: the plain write convenience is
    // persist-only and the relay never broadcasts it.
    const next = createClock();
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: [time.key, data.key],
    });
    try {
      // The stream converges to the registered demand in the background, so writes
      // repeat until one lands on the open stream.
      await expect
        .poll(
          async () => {
            await writer.write({ [time.key]: [next()], [data.key]: [42] });
            return received.length > 0;
          },
          { timeout: 10000, interval: 250 },
        )
        .toBe(true);
    } finally {
      await writer.close();
    }
    expect(received).toContain(42);
    sub.close();
  });

  it("should return every written sample across the live boundary", async () => {
    const { time, data } = await createChannels();
    const received: number[] = [];
    const sub = feed.stream(
      (res) => {
        const series = res.get(data.key);
        if (series != null) received.push(...(Array.from(series) as number[]));
      },
      [data.key],
    );
    const start = TimeStamp.now();
    const next = createClock();
    let value = 0;
    const writeUntilStreamed = async (
      w: Awaited<ReturnType<typeof client.openWriter>>,
    ) => {
      const before = received.length;
      await expect
        .poll(
          async () => {
            await w.write({ [time.key]: [next()], [data.key]: [value++] });
            return received.length > before;
          },
          { timeout: 10000, interval: 250 },
        )
        .toBe(true);
    };
    const w1 = await client.openWriter({ start, channels: [time.key, data.key] });
    try {
      await writeUntilStreamed(w1);
    } finally {
      await w1.close();
    }
    // A read that spans the live leading buffer includes it, and may also return
    // the fetched form of the same samples: the two representations carry
    // different alignments and both stay visible until the fetched form wins.
    const tr = new TimeRange(start, TimeStamp.now().add(TimeSpan.seconds(1)));
    const res = await feed.read(tr, data.key);
    expectWrittenValues(Array.from(res) as number[], value);
    // A second writer opens a new alignment domain, which flushes the old leading
    // buffer into the static cache as a streamed entry.
    const w2 = await client.openWriter({
      start: TimeStamp.now(),
      channels: [time.key, data.key],
    });
    try {
      await writeUntilStreamed(w2);
    } finally {
      await w2.close();
    }
    const tr2 = new TimeRange(start, TimeStamp.now().add(TimeSpan.seconds(1)));
    const res2 = await feed.read(tr2, data.key);
    expectWrittenValues(Array.from(res2) as number[], value);
    sub.close();
  });

  it("should include the same series object in reads that the stream delivers", async () => {
    const { time, data } = await createChannels();
    const received: Series[] = [];
    const sub = feed.stream(
      (res) => {
        const series = res.get(data.key);
        if (series != null) received.push(...series.series);
      },
      [data.key],
    );
    const start = TimeStamp.now();
    const next = createClock();
    const writer = await client.openWriter({ start, channels: [time.key, data.key] });
    try {
      await expect
        .poll(
          async () => {
            await writer.write({ [time.key]: [next()], [data.key]: [1] });
            return received.length > 0;
          },
          { timeout: 10000, interval: 250 },
        )
        .toBe(true);
    } finally {
      await writer.close();
    }
    const tr = new TimeRange(start, TimeStamp.now().add(TimeSpan.seconds(1)));
    const res = await feed.read(tr, data.key);
    // Identity equality lets consumers that both read and stream deduplicate the
    // live buffer.
    expect(received.some((s) => res.series.includes(s))).toBe(true);
    sub.close();
  });

  it("should backload history that pairs with a warm index by alignment", async () => {
    const { time, data } = await createChannels();
    const short = client.openFeed({ removalDelay: TimeSpan.milliseconds(100) });
    try {
      const start = TimeStamp.now();
      const next = createClock();
      const w = await client.openWriter({
        start,
        channels: [time.key, data.key],
        enableAutoCommit: true,
      });
      let value = 0;
      try {
        // Another consumer streams the index while history accumulates, so the
        // index cache covers the window with leading-alignment data only.
        const sub = short.stream(() => {}, [time.key]);
        for (let i = 0; i < 40; i++) {
          await w.write({ [time.key]: [next()], [data.key]: [value++] });
          await sleep.sleep(TimeSpan.milliseconds(5));
        }
        // A plot opens on the never-streamed channel: it reads the channel and its
        // index over the same window.
        const tr = new TimeRange(start, TimeStamp.now());
        const yRes = await short.read(tr, data.key);
        const xRes = await short.read(tr, time.key);
        // The line renderer pairs x and y series by alignment overlap, so every
        // backloaded y series must sit inside the union of x alignment spans.
        const xSpans = xRes.series
          .map((s) => s.alignmentBounds)
          .sort((a, b) => Number(a.lower - b.lower));
        const merged: Array<{ lower: bigint; upper: bigint }> = [];
        for (const s of xSpans) {
          const last = merged.at(-1);
          if (last != null && s.lower <= last.upper) {
            if (s.upper > last.upper) last.upper = s.upper;
          } else merged.push({ lower: s.lower, upper: s.upper });
        }
        const uncovered = yRes.series.filter(
          (y) =>
            !merged.some(
              (m) =>
                y.alignmentBounds.lower >= m.lower &&
                y.alignmentBounds.upper <= m.upper,
            ),
        );
        expect(
          uncovered.map((s) => ({
            alignment: s.alignment.toString(16),
            length: s.length,
          })),
        ).toHaveLength(0);
        sub.close();
      } finally {
        await w.close();
      }
    } finally {
      await short.close();
    }
  });

  it("should keep reads complete across leading buffer rollovers", async () => {
    const { time, data } = await createChannels();
    // A small buffer forces frequent flush and realloc cycles.
    const small = client.openFeed({ dynamicBufferSize: 50 });
    try {
      const start = TimeStamp.now();
      const next = createClock();
      const w = await client.openWriter({
        start,
        channels: [time.key, data.key],
        enableAutoCommit: true,
      });
      try {
        const sub = small.stream(() => {}, [time.key, data.key]);
        let value = 0;
        for (let i = 0; i < 10; i++) {
          for (let j = 0; j < 30; j++)
            await w.write({
              [time.key]: [next(), next()],
              [data.key]: [value++, value++],
            });
          const tr = new TimeRange(start, TimeStamp.now());
          // The fresh baseline reads first: anything committed by now must also
          // reach the later cached read, so autocommit lag cannot flake this.
          const fresh = client.openFeed();
          try {
            const all = Array.from(await fresh.read(tr, data.key)) as number[];
            const got = new Set(Array.from(await small.read(tr, data.key)) as number[]);
            const missing = all.filter((v) => !got.has(v));
            expect(missing).toHaveLength(0);
          } finally {
            await fresh.close();
          }
        }
        sub.close();
      } finally {
        await w.close();
      }
    } finally {
      await small.close();
    }
  });

  it("should refetch samples written while a channel was unstreamed", async () => {
    const { time, data } = await createChannels();
    const short = client.openFeed({ removalDelay: TimeSpan.milliseconds(100) });
    try {
      const start = TimeStamp.now();
      const next = createClock();
      const w = await client.openWriter({ start, channels: [time.key, data.key] });
      let value = 0;
      try {
        const received: number[] = [];
        const sub = short.stream(
          (res) => {
            const series = res.get(data.key);
            if (series != null) received.push(...(Array.from(series) as number[]));
          },
          [data.key],
        );
        await expect
          .poll(
            async () => {
              await w.write({ [time.key]: [next()], [data.key]: [value++] });
              return received.length > 0;
            },
            { timeout: 10000, interval: 100 },
          )
          .toBe(true);
        // Ends the channel's streaming demand; the wait outlasts the removal delay
        // and the reconcile that shrinks the stream.
        sub.close();
        await sleep.sleep(TimeSpan.milliseconds(500));
        for (let i = 0; i < 10; i++)
          await w.write({ [time.key]: [next()], [data.key]: [value++] });
        await w.commit();
      } finally {
        await w.close();
      }
      const tr = new TimeRange(start, TimeStamp.now());
      const cached = new Set(Array.from(await short.read(tr, data.key)) as number[]);
      const fresh = client.openFeed();
      try {
        const all = Array.from(await fresh.read(tr, data.key)) as number[];
        expect(all.length).toBeGreaterThanOrEqual(value - 1);
        const missing = all.filter((v) => !cached.has(v));
        expect(missing).toHaveLength(0);
      } finally {
        await fresh.close();
      }
    } finally {
      await short.close();
    }
  });

  it("should keep far-past streamed data out of reads of the present", async () => {
    const { time, data } = await createChannels();
    const received: number[] = [];
    const sub = feed.stream(
      (res) => {
        const series = res.get(data.key);
        if (series != null) received.push(...(Array.from(series) as number[]));
      },
      [data.key],
    );
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
            return received.length > 0;
          },
          { timeout: 10000, interval: 100 },
        )
        .toBe(true);
    } finally {
      await writer.close();
    }
    try {
      // The live buffer holds only epoch-era samples, so a read of the recent past must
      // come back empty instead of serving them.
      const now = TimeStamp.now();
      const recent = new TimeRange(now.sub(TimeSpan.seconds(30)), now);
      expect((await feed.read(recent, data.key)).length).toBe(0);
      // A read that targets the buffer's own era still serves it.
      const past = new TimeRange(TimeStamp.ZERO, TimeStamp.seconds(60));
      expect((await feed.read(past, data.key)).length).toBeGreaterThan(0);
    } finally {
      sub.close();
    }
  });

  it("should reject reads after the feed closes", async () => {
    const closable = client.openFeed();
    await closable.close();
    const tr = new TimeRange(TimeStamp.now(), TimeStamp.now().add(TimeSpan.seconds(1)));
    await expect(closable.read(tr, 123)).rejects.toThrow(UnexpectedError);
  });

  it("should forward staleCoverageThreshold to the cache", async () => {
    let calls = 0;
    const direct = new Feed({
      staleCoverageThreshold: TimeSpan.milliseconds(50),
      readRemote: async () => {
        calls++;
        return new Frame([], []);
      },
      openStreamer: async () => {
        throw new UnexpectedError("streamer unused");
      },
    });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    await direct.read(tr, 1);
    await direct.read(tr, 1);
    expect(calls).toBe(1);
    await sleep.sleep(TimeSpan.milliseconds(60));
    await direct.read(tr, 1);
    expect(calls).toBe(2);
    await direct.close();
  });
});
