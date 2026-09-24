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

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type Transform } from "@/framer/cache/transform";
import { Feed, type FeedOptions, type LatestReader } from "@/framer/feed";
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

  it("should report no last write before a frame lands for the key", async () => {
    const { data } = await createChannels();
    const sub = feed.stream(() => {}, [data.key]);
    expect(sub.lastWrite(data.key)).toBeNull();
    sub.close();
  });

  it("should report the Core's stamp for the last streamed write", async () => {
    const { time, data } = await createChannels();
    let received = 0;
    const sub = feed.stream(
      (res) => {
        received += res.get(data.key)?.length ?? 0;
      },
      [data.key],
    );
    const next = createClock();
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: [time.key, data.key],
    });
    try {
      await expect
        .poll(
          async () => {
            await writer.write({ [time.key]: [next()], [data.key]: [1] });
            return received > 0;
          },
          { timeout: 10000, interval: 250 },
        )
        .toBe(true);
      // The stream has converged, so this write is the one whose stamp to observe.
      const last = next();
      await writer.write({ [time.key]: [last], [data.key]: [2] });
      // The Core stamps a streamed series with its last timestamp plus one nanosecond.
      await expect
        .poll(() => sub.lastWrite(data.key)?.valueOf(), { timeout: 5000 })
        .toEqual(last.valueOf() + 1n);
    } finally {
      await writer.close();
    }
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
      readLatest: async () => new Frame([], []),
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

  describe("readLatest", () => {
    const createFeed = (readLatest: LatestReader, transform?: Transform): Feed =>
      new Feed({
        transform,
        readLatest,
        readRemote: async () => {
          throw new UnexpectedError("reader unused");
        },
        openStreamer: async () => {
          throw new UnexpectedError("streamer unused");
        },
      });

    const latestFrame = (keys: channel.Key[]): Frame =>
      new Frame(
        keys,
        keys.map((k) => new Series({ data: new Float32Array([k]) })),
      );

    it("should return the latest stored sample", async () => {
      const { time, data } = await createChannels();
      const start = TimeStamp.now();
      await client.write(start, {
        [time.key]: [start, start.add(TimeSpan.milliseconds(1))],
        [data.key]: [1, 2],
      });
      expect(Array.from(await feed.readLatest(data.key))).toEqual([2]);
    });

    it("should coalesce concurrent reads into one request", async () => {
      const calls: channel.Key[][] = [];
      const direct = createFeed(async (keys) => {
        calls.push(keys);
        return latestFrame(keys);
      });
      const [a, b, c] = await Promise.all([
        direct.readLatest(1),
        direct.readLatest(2),
        direct.readLatest(1),
      ]);
      expect(calls).toEqual([[1, 2]]);
      expect(Array.from(a)).toEqual([1]);
      expect(Array.from(b)).toEqual([2]);
      expect(Array.from(c)).toEqual([1]);
      await direct.close();
    });

    it("should open a new window once the first has fired", async () => {
      let calls = 0;
      const direct = createFeed(async (keys) => {
        calls++;
        return latestFrame(keys);
      });
      await direct.readLatest(1);
      await direct.readLatest(1);
      expect(calls).toBe(2);
      await direct.close();
    });

    it("should apply the configured transform", async () => {
      const transform: Transform = {
        resolveDataType: () => DataType.FLOAT32,
        convert: (series) => series.convert(DataType.FLOAT32),
      };
      const direct = createFeed(
        async (keys) =>
          new Frame(
            keys,
            keys.map(() => new Series({ data: new Int32Array([9]) })),
          ),
        transform,
      );
      const res = await direct.readLatest(1);
      expect(res.series[0].dataType.equals(DataType.FLOAT32)).toBe(true);
      expect(Array.from(res)).toEqual([9]);
      await direct.close();
    });

    it("should return an empty result for a channel with no samples", async () => {
      const direct = createFeed(async () => new Frame([], []));
      expect((await direct.readLatest(1)).length).toBe(0);
      await direct.close();
    });

    it("should reject every read in a failed batch", async () => {
      const direct = createFeed(async () => {
        throw new Error("iterator exploded");
      });
      const results = await Promise.allSettled([
        direct.readLatest(1),
        direct.readLatest(2),
      ]);
      expect(results.map((r) => r.status)).toEqual(["rejected", "rejected"]);
      await direct.close();
    });

    it("should reject pending and later reads once the feed closes", async () => {
      const direct = createFeed(async (keys) => latestFrame(keys));
      const pending = direct.readLatest(1);
      await direct.close();
      await expect(pending).rejects.toThrow(UnexpectedError);
      await expect(direct.readLatest(1)).rejects.toThrow(UnexpectedError);
    });

    it("should reject in-flight reads as soon as the feed closes", async () => {
      let release = (): void => {};
      const gate = new Promise<void>((resolve) => (release = resolve));
      const direct = createFeed(async (keys) => {
        await gate;
        return latestFrame(keys);
      });
      const reads = [direct.readLatest(1), direct.readLatest(2)];
      await sleep.sleep(TimeSpan.milliseconds(60));
      await direct.close();
      const results = await Promise.allSettled(reads);
      expect(results.map((r) => r.status)).toEqual(["rejected", "rejected"]);
      release();
    });

    it("should report the close rather than a later request failure", async () => {
      let release = (): void => {};
      const gate = new Promise<void>((resolve) => (release = resolve));
      const direct = createFeed(async () => {
        await gate;
        throw new Error("iterator exploded");
      });
      const read = direct.readLatest(1);
      await sleep.sleep(TimeSpan.milliseconds(60));
      await direct.close();
      release();
      await expect(read).rejects.toThrow(UnexpectedError);
    });

    describe("while the channel is streamed", () => {
      // A feed over the test client that counts latest reads and can hold them.
      const createCountingFeed = (options: FeedOptions = {}) => {
        let calls = 0;
        let gate: Promise<void> | null = null;
        const counting = new Feed({
          ...options,
          readRemote: async (tr, keys) => await client.read(tr, keys),
          openStreamer: async (config) => await client.openStreamer(config),
          readLatest: async (keys) => {
            calls++;
            if (gate != null) await gate;
            return await client.readLatest(keys, 1);
          },
        });
        return {
          feed: counting,
          calls: () => calls,
          hold: (): (() => void) => {
            let release = (): void => {};
            gate = new Promise<void>((resolve) => (release = resolve));
            return () => {
              gate = null;
              release();
            };
          },
        };
      };

      const createStreamedChannel = async (counting: Feed) => {
        const { time, data } = await createChannels();
        const start = TimeStamp.now();
        await client.write(start, {
          [time.key]: [start, start.add(TimeSpan.milliseconds(1))],
          [data.key]: [1, 2],
        });
        const sub = counting.stream(() => {}, [data.key]);
        await expect
          .poll(() => sub.status(data.key).variant, { timeout: 5000 })
          .toBe("success");
        return { time, data, sub };
      };

      it("should serve the second read without a request", async () => {
        const { feed: counting, calls } = createCountingFeed();
        try {
          const { data, sub } = await createStreamedChannel(counting);
          expect(Array.from(await counting.readLatest(data.key))).toEqual([2]);
          expect(Array.from(await counting.readLatest(data.key))).toEqual([2]);
          expect(calls()).toBe(1);
          sub.close();
        } finally {
          await counting.close();
        }
      });

      it("should read again once a live write lands", async () => {
        const { feed: counting, calls } = createCountingFeed();
        try {
          const { time, data, sub } = await createStreamedChannel(counting);
          await counting.readLatest(data.key);
          expect(calls()).toBe(1);
          const next = createClock();
          const writer = await client.openWriter({
            start: TimeStamp.now(),
            channels: [time.key, data.key],
          });
          try {
            await expect
              .poll(
                async () => {
                  await writer.write({ [time.key]: [next()], [data.key]: [3] });
                  return sub.lastWrite(data.key) != null;
                },
                { timeout: 10000, interval: 250 },
              )
              .toBe(true);
          } finally {
            await writer.close();
          }
          await counting.readLatest(data.key);
          expect(calls()).toBe(2);
          sub.close();
        } finally {
          await counting.close();
        }
      });

      it("should not store a read that a live write overtook", async () => {
        const { feed: counting, calls, hold } = createCountingFeed();
        try {
          const { time, data, sub } = await createStreamedChannel(counting);
          const release = hold();
          const overtaken = counting.readLatest(data.key);
          await expect.poll(calls, { timeout: 5000 }).toBe(1);
          const next = createClock();
          const writer = await client.openWriter({
            start: TimeStamp.now(),
            channels: [time.key, data.key],
          });
          try {
            await expect
              .poll(
                async () => {
                  await writer.write({ [time.key]: [next()], [data.key]: [3] });
                  return sub.lastWrite(data.key) != null;
                },
                { timeout: 10000, interval: 250 },
              )
              .toBe(true);
          } finally {
            await writer.close();
          }
          release();
          await overtaken;
          await counting.readLatest(data.key);
          expect(calls()).toBe(2);
          sub.close();
        } finally {
          await counting.close();
        }
      });

      it("should read again once the channel leaves the stream", async () => {
        const { feed: counting, calls } = createCountingFeed({
          removalDelay: TimeSpan.milliseconds(100),
        });
        try {
          const { data, sub } = await createStreamedChannel(counting);
          await counting.readLatest(data.key);
          await counting.readLatest(data.key);
          expect(calls()).toBe(1);
          // Outlasts the removal delay and the reconcile that shrinks the stream.
          sub.close();
          await sleep.sleep(TimeSpan.milliseconds(500));
          await counting.readLatest(data.key);
          await counting.readLatest(data.key);
          expect(calls()).toBe(3);
        } finally {
          await counting.close();
        }
      });
    });
  });
});
