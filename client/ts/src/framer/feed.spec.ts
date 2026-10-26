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
  array,
  DataType,
  dynamicWindow,
  id,
  Series,
  Size,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterAll, describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import { tileSpan } from "@/framer/cache/grid";
import { type TileSpec } from "@/framer/cache/tile";
import { Feed, type FeedProps, type RemoteLatestReader } from "@/framer/feed";
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

/** Retrieves every requested key as an index channel. */
const retrieveIndexes: ChannelRetriever = async (params) =>
  (array.toArray(params) as channel.Key[]).map(
    (key) =>
      new channel.Channel({
        key,
        name: `index-${key}`,
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      }).payload,
  );

/** Asserts a read returned every value written so far, each exactly once. */
const expectWrittenValues = (values: number[], writeCount: number): void => {
  const counts = new Map<number, number>();
  values.forEach((v) => counts.set(v, (counts.get(v) ?? 0) + 1));
  expect([...counts.keys()].sort((a, b) => a - b)).toEqual(
    Array.from({ length: writeCount }, (_, i) => i),
  );
  expect([...counts.entries()].filter(([, count]) => count > 1)).toEqual([]);
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
    // buffer into its raw block as a streamed entry.
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

  it("should return only the fetched form of a span the live buffer holds", async () => {
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
    expect(res.length).toBeGreaterThan(0);
    expect(received.some((s) => res.series.includes(s))).toBe(false);
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
    const values = await feed.read(tr, calc.key);
    const times = await feed.read(tr, calc.index);
    expect(Array.from(values)).toEqual([2, 4, 6]);
    expect(Array.from(times)).toEqual(stamps.map((t) => t.valueOf()));
    expect(values.series.map((s) => s.alignmentBounds)).toEqual(
      times.series.map((s) => s.alignmentBounds),
    );
  });

  it("should reject reads after the feed closes", async () => {
    const closable = client.openFeed();
    await closable.close();
    const tr = new TimeRange(TimeStamp.now(), TimeStamp.now().add(TimeSpan.seconds(1)));
    await expect(closable.read(tr, 123)).rejects.toThrow(UnexpectedError);
  });

  it("should forward fetchTimeout to the reader", async () => {
    const direct = new Feed({
      fetchTimeout: TimeSpan.milliseconds(20),
      retrieveChannels: retrieveIndexes,
      readRemote: async () => await new Promise<Frame>(() => {}),
      openStreamer: async () => {
        throw new UnexpectedError("streamer unused");
      },
      readLatest: async () => new Frame([], []),
    });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    await expect(direct.read(tr, 1)).rejects.toThrow(Unreachable);
    await direct.close();
  });

  it("should close while a stream open is still retrying", async () => {
    let opens = 0;
    const direct = new Feed({
      readRemote: async () => new Frame([], []),
      retrieveChannels: retrieveIndexes,
      openStreamer: async () => {
        opens++;
        throw new Unreachable();
      },
      readLatest: async () => new Frame([], []),
      onRetry: () => {},
    });
    direct.stream(() => {}, [1]);
    await expect.poll(() => opens).toBeGreaterThan(0);
    const closed = direct.close().then(() => true);
    const timedOut = sleep.sleep(TimeSpan.seconds(1)).then(() => false);
    expect(await Promise.race([closed, timedOut])).toBe(true);
  });

  describe("readTile", () => {
    const LEVEL = 12;
    const COUNT = 1000;

    // Writes COUNT samples one millisecond apart from the start of a finished tile.
    // Data sample i holds the value i.
    const writeTile = async () => {
      const channels = await createChannels();
      const span = tileSpan(LEVEL);
      const index = Number(TimeStamp.now().valueOf() / span.valueOf()) - 1;
      const start = new TimeStamp(BigInt(index) * span.valueOf());
      await client.write(start, {
        [channels.time.key]: Array.from({ length: COUNT }, (_, i) =>
          start.add(TimeSpan.milliseconds(i)),
        ),
        [channels.data.key]: Array.from({ length: COUNT }, (_, i) => i),
      });
      return { ...channels, index, start };
    };

    it("should read the min and max of each group of a tile", async () => {
      const { time, data, index, start } = await writeTile();
      const tile = { level: LEVEL, index, pointLimit: 100 };
      const aggregation = "min_max";
      const [times, values] = await Promise.all([
        feed.readTile({ ...tile, key: time.key, aggregation }),
        feed.readTile({ ...tile, key: data.key, aggregation }),
      ]);
      expect(Array.from(values)).toEqual(
        Array.from({ length: 50 }, (_, g) => [g * 20, g * 20 + 19]).flat(),
      );
      expect(times.length).toBe(100);
      expect(times.series[0].at(0)).toEqual(start.valueOf());
      expect(values.series.map((s) => s.alignment)).toEqual(
        times.series.map((s) => s.alignment),
      );
      values.series.forEach((s) => expect(s.alignmentMultiple).toBe(10n));
    });

    it("should read the mean of each group of a tile", async () => {
      const { data, index } = await writeTile();
      const values = await feed.readTile({
        key: data.key,
        level: LEVEL,
        index,
        pointLimit: 100,
        aggregation: "average",
      });
      expect(Array.from(values)).toEqual(
        Array.from({ length: 100 }, (_, g) => g * 10 + 4.5),
      );
      values.series.forEach((s) => expect(s.alignmentMultiple).toBe(10n));
    });

    it("should return every sample of a tile under the point limit", async () => {
      const { data, index } = await writeTile();
      const values = await feed.readTile({
        key: data.key,
        level: LEVEL,
        index,
        pointLimit: 2000,
        aggregation: "min_max",
      });
      expect(Array.from(values)).toEqual(Array.from({ length: COUNT }, (_, i) => i));
      values.series.forEach((s) => expect(s.alignmentMultiple).toBe(1n));
    });

    it("should serve a second read of a finished tile from the cache", async () => {
      const { data, index } = await writeTile();
      const spec: TileSpec = {
        key: data.key,
        level: LEVEL,
        index,
        pointLimit: 100,
        aggregation: "min_max",
      };
      const first = await feed.readTile(spec);
      expect(await feed.readTile(spec)).toBe(first);
    });

    it("should read a partial tile up to its end", async () => {
      const { data, index, start } = await writeTile();
      const values = await feed.readTile({
        key: data.key,
        level: LEVEL,
        index,
        pointLimit: 100,
        aggregation: "min_max",
        end: start.add(TimeSpan.milliseconds(500)),
      });
      expect(Array.from(values)).toEqual(
        Array.from({ length: 50 }, (_, g) => [g * 10, g * 10 + 9]).flat(),
      );
    });

    it("should read an empty tile for a channel with no data in it", async () => {
      const { data, index } = await writeTile();
      const values = await feed.readTile({
        key: data.key,
        level: LEVEL,
        index: index - 1,
        pointLimit: 100,
        aggregation: "min_max",
      });
      expect(values.length).toBe(0);
    });

    const TILE: TileSpec = {
      key: 1,
      level: LEVEL,
      index: 3,
      pointLimit: 100,
      aggregation: "min_max",
    };

    // Returns a feed whose remote read counts its calls and answers every key with one
    // float32 sample equal to the key.
    const createFeed = (props: Partial<FeedProps> = {}) => {
      const counter = { calls: 0 };
      const direct = new Feed({
        retrieveChannels: retrieveIndexes,
        readRemote: async (_, keys) => {
          counter.calls++;
          return new Frame(
            keys,
            keys.map((k) => new Series({ data: new Float32Array([k]) })),
          );
        },
        openStreamer: async () => {
          throw new UnexpectedError("streamer unused");
        },
        readLatest: async () => new Frame([], []),
        ...props,
      });
      return { direct, counter };
    };

    it("should forward fetchTimeout to the tile reader", async () => {
      const { direct } = createFeed({
        fetchTimeout: TimeSpan.milliseconds(20),
        readRemote: async () => await new Promise<Frame>(() => {}),
      });
      const read = direct.readTile(TILE);
      await expect(read).rejects.toThrow(Unreachable);
      await expect(read).rejects.toThrow("timed out after 20ms");
      await direct.close();
    });

    it("should forward budget to the block cache", async () => {
      const { direct, counter } = createFeed({ budget: Size.bytes(4) });
      await direct.readTile({ ...TILE, index: 1 });
      await direct.readTile({ ...TILE, index: 2 });
      await direct.readTile({ ...TILE, index: 1 });
      expect(counter.calls).toBe(3);
      await direct.close();
    });

    it("should reject tile reads after the feed closes", async () => {
      const { direct } = createFeed();
      await direct.close();
      await expect(direct.readTile(TILE)).rejects.toThrow("tile reader is closed");
    });
  });

  describe("readLatest", () => {
    const createFeed = (readLatest: RemoteLatestReader): Feed =>
      new Feed({
        readLatest,
        retrieveChannels: retrieveIndexes,
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
  });

  describe("subscribe", () => {
    it("should hold each sample once while a writer runs", async () => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const start = TimeStamp.now();
      const writer = await client.openWriter({
        start,
        channels: [index.key, data.key],
      });
      let written = 0;
      let running = true;
      // A 1 kHz writer: ten samples a millisecond apart, paced to the wall clock. Each
      // data sample holds its sample number.
      const writing = (async () => {
        while (running) {
          const numbers = Array.from({ length: 10 }, (_, i) => written + i);
          await writer.write({
            [index.key]: new BigInt64Array(
              numbers.map((n) => start.add(TimeSpan.milliseconds(n)).valueOf()),
            ),
            [data.key]: new Float64Array(numbers),
          });
          written += 10;
          const ahead = start.add(TimeSpan.milliseconds(written)).span(TimeStamp.now());
          if (start.add(TimeSpan.milliseconds(written)).after(TimeStamp.now()))
            await sleep.sleep(ahead);
        }
      })();
      const seamFeed = client.openFeed({ removalDelay: TimeSpan.ZERO });
      const span = TimeSpan.seconds(2);
      try {
        for (let i = 0; i < 20; i++) {
          const sub = seamFeed.subscribe({
            keys: [data.key],
            window: dynamicWindow(span),
            limit: { variant: "points", count: Infinity, aggregation: "min_max" },
          });
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
            const numbers = Array.from(sub.value(data.key).series) as number[];
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
  });
});
