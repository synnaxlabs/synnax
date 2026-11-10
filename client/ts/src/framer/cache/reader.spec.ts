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
  DataType,
  id,
  MultiSeries,
  Rate,
  Size,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { framer } from "@/framer";
import { RAW_LEVEL, rawIndex } from "@/framer/cache/block";
import { indexes, tileRange } from "@/framer/cache/grid";
import { Reader, type ReaderProps } from "@/framer/cache/reader";
import { Store } from "@/framer/cache/store";
import { NotFoundError, UnexpectedError } from "@/index";
import { createTestClient, SyntheticCore } from "@/testutil";

const blockRange = (index: number): TimeRange => tileRange({ level: RAW_LEVEL, index });

const blockAlignment = (index: number, offset: number): bigint =>
  (BigInt(index) << 32n) | BigInt(offset);

// Every Core below samples at 1 Hz from the start of raw block 100.
const START = blockRange(100).start;

const at = (seconds: number): TimeStamp => START.add(TimeSpan.seconds(seconds));

interface Fetch {
  tr: TimeRange;
  keys: channel.Key[];
}

interface Setup {
  core: SyntheticCore;
  reader: Reader;
  store: Store;
  fetches: Fetch[];
  /** Fills the channel's blocks over tr, and returns their samples that overlap it. */
  read: (tr: TimeRange, ch: channel.Key) => Promise<MultiSeries>;
}

const readBlocks = async (
  reader: Reader,
  store: Store,
  tr: TimeRange,
  ch: channel.Key,
): Promise<MultiSeries> => {
  await reader.fill(tr, ch);
  return new MultiSeries(
    indexes(tr, RAW_LEVEL).flatMap((i) => store.findRaw(ch, i)?.seriesIn(tr) ?? []),
  );
};

interface SetupProps extends Partial<ReaderProps> {
  /** Seconds after START on the Core's clock. */
  elapsed?: number;
  /** Runs before each fetch is answered. A rejection fails the fetch. */
  before?: (fetch: Fetch) => Promise<void>;
  /** Changes each frame the Core answers with. */
  answer?: (frame: framer.Frame) => framer.Frame;
  /** Runs before each channel retrieval is answered. */
  beforeRetrieve?: () => Promise<void>;
  /** The stale threshold of the store, on the Core's clock. */
  staleThreshold?: TimeSpan;
}

// Returns a reader over a Core with one index channel (1) and two data channels (2 and
// 3) on it, recording every fetch the reader makes.
const createReader = ({
  elapsed = 300,
  before,
  answer = (frame) => frame,
  beforeRetrieve,
  staleThreshold,
  ...props
}: SetupProps = {}): Setup => {
  const core = new SyntheticCore({
    rate: Rate.hz(1),
    start: START,
    now: at(elapsed),
    channels: 2,
  });
  const fetches: Fetch[] = [];
  const store = props.store ?? new Store({ staleThreshold, now: core.now });
  const reader = new Reader({
    readRemote: async (tr, keys) => {
      const fetch = { tr, keys };
      fetches.push(fetch);
      await before?.(fetch);
      return answer(core.read(tr, keys));
    },
    retrieveChannels: async (keys) => {
      await beforeRetrieve?.();
      return await core.retrieveChannels(keys);
    },
    now: core.now,
    ...props,
    store,
  });
  const read = async (tr: TimeRange, ch: channel.Key) =>
    await readBlocks(reader, store, tr, ch);
  return { core, reader, store, fetches, read };
};

// Returns a gate that holds every fetch until it opens.
const createGate = () => {
  const { promise, resolve } = Promise.withResolvers<void>();
  return { wait: async () => await promise, open: () => resolve() };
};

const timestamps = (res: MultiSeries): bigint[] => Array.from(res) as bigint[];

const expectedTimestamps = (from: number, to: number): bigint[] =>
  Array.from({ length: to - from }, (_, i) => at(from + i).valueOf());

describe("Reader", () => {
  describe("fetching", () => {
    it("should fetch the whole block a read needs, with the channel's index", async () => {
      const { fetches, read } = createReader();
      const res = await read(new TimeRange(at(10), at(20)), 2);
      expect(fetches).toEqual([{ tr: blockRange(100), keys: [2, 1] }]);
      expect(res.length).toBe(66);
    });

    it("should serve a covered read from the cache", async () => {
      const { fetches, read } = createReader();
      const tr = new TimeRange(at(10), at(20));
      const first = await read(tr, 2);
      const second = await read(tr, 2);
      expect(fetches).toHaveLength(1);
      expect(second.series).toEqual(first.series);
    });

    it("should fetch the blocks a read spans in one Core read", async () => {
      const { fetches, read } = createReader();
      const res = await read(new TimeRange(at(60), at(70)), 1);
      expect(fetches.map(({ tr }) => tr)).toEqual([
        new TimeRange(blockRange(100).start, blockRange(101).end),
      ]);
      expect(timestamps(res)).toEqual(expectedTimestamps(0, 132));
      expect(res.series.map((s) => s.alignment)).toEqual([
        blockAlignment(100, 0),
        blockAlignment(101, 0),
      ]);
    });

    it("should fetch the blocks on each side of a covered block apart", async () => {
      const { fetches, read } = createReader();
      await read(new TimeRange(at(70), at(80)), 2);
      const res = await read(new TimeRange(at(10), at(140)), 2);
      expect(fetches.map(({ tr }) => tr)).toEqual([
        blockRange(101),
        blockRange(100),
        blockRange(102),
      ]);
      expect(res.length).toBe(197);
    });

    it("should share one Core read among channels reading one block", async () => {
      const { fetches, read } = createReader();
      const tr = new TimeRange(at(10), at(20));
      await Promise.all([read(tr, 2), read(tr, 3)]);
      expect(fetches).toHaveLength(1);
      expect(new Set(fetches[0].keys)).toEqual(new Set([1, 2, 3]));
    });

    it("should cover a block the Core answers with no data", async () => {
      const { fetches, read } = createReader();
      const tr = new TimeRange(at(-30), at(-20));
      expect(await read(tr, 2)).toHaveLength(0);
      expect(await read(tr, 2)).toHaveLength(0);
      expect(fetches).toHaveLength(1);
    });

    it("should refetch the tail of a block too recent to commit on the Core's clock", async () => {
      const { core, fetches, read } = createReader({ elapsed: 30 });
      await read(new TimeRange(at(0), at(30)), 1);
      core.advance(TimeSpan.seconds(10));
      const res = await read(new TimeRange(at(0), at(40)), 1);
      expect(fetches.map(({ tr }) => tr)).toEqual([
        blockRange(100),
        new TimeRange(at(29), blockRange(100).end),
      ]);
      expect(timestamps(res)).toEqual(expectedTimestamps(0, 40));
    });

    it("should refetch a block once its fetched samples expire", async () => {
      const { core, fetches, read } = createReader({
        staleThreshold: TimeSpan.minutes(1),
      });
      const tr = new TimeRange(at(10), at(20));
      await read(tr, 2);
      await read(tr, 2);
      expect(fetches).toHaveLength(1);
      core.advance(TimeSpan.minutes(1));
      expect(await read(tr, 2)).toHaveLength(66);
      expect(fetches).toHaveLength(2);
    });
  });

  describe("catch-up", () => {
    it("should cover a channel up to its own newest sample", async () => {
      const { reader, store } = createReader({
        elapsed: 20,
        // The Core committed the index two samples past its data channel.
        answer: (frame) =>
          new framer.Frame(
            frame.map((k, s) => [k, k === 2 ? s.slice(0, s.length - 2) : s]),
          ),
      });
      await Promise.all([reader.catchUp(1, 100), reader.catchUp(2, 100)]);
      expect(store.raw(1, 100).coveredCount).toBe(20);
      expect(store.raw(2, 100).coveredCount).toBe(18);
      expect(store.raw(2, 100).coveredEnd).toEqual(at(17).add(1));
    });
  });

  describe("back-fill", () => {
    it.each([TimeSpan.minutes(10), TimeSpan.days(1)])(
      "should back-fill %s in one Core read split at block edges",
      async (span) => {
        const { core, fetches, read } = createReader({ elapsed: span.seconds });
        const res = await read(new TimeRange(at(0), core.now()), 2);
        expect(fetches).toHaveLength(1);
        expect(res.length).toBe(span.seconds);
        res.series.forEach((s) => {
          const index = rawIndex(s.timeRange.start);
          const block = blockRange(index);
          expect(s.alignment >> 32n).toBe(BigInt(index));
          expect(s.timeRange.end.beforeEq(block.end)).toBe(true);
        });
      },
    );
  });

  describe("catch-up", () => {
    it("should fill only the blocks that hold samples from a block long before now", async () => {
      const now = at(300).add(TimeSpan.days(7));
      const { reader, store, fetches, read } = createReader({ now: () => now });
      await reader.catchUp(2, 100);
      const held = (ch: channel.Key) =>
        indexes(new TimeRange(START, now), RAW_LEVEL).filter(
          (i) => store.findRaw(ch, i) != null,
        );
      expect(held(2)).toEqual([100, 101, 102, 103, 104]);
      expect(held(1)).toEqual([100, 101, 102, 103, 104]);
      expect(await read(new TimeRange(START, at(299)), 2)).toHaveLength(300);
      expect(fetches.map(({ tr }) => tr)).toEqual([
        new TimeRange(START, TimeStamp.MAX),
      ]);
    });

    it("should merge into a later block that a read filled first", async () => {
      const { core, reader, store, read } = createReader({ elapsed: 100 });
      await read(new TimeRange(at(70), at(80)), 1);
      core.advance(TimeSpan.seconds(200));
      await reader.catchUp(1, 100);
      const range = blockRange(101);
      const expected = expectedTimestamps(0, 300).filter(
        (t) => t >= range.start.valueOf() && t < range.end.valueOf(),
      );
      const block = store.findRaw(1, 101);
      expect(timestamps(new MultiSeries([...(block?.fetched ?? [])]))).toEqual(
        expected,
      );
      expect(block?.coveredCount).toBe(expected.length);
      expect(block?.coveredEnd).toEqual(range.end);
    });

    it("should keep the coverage of a later block that a newer read filled", async () => {
      const gate = createGate();
      let reads = 0;
      const setup = createReader({
        elapsed: 100,
        batchDebounce: TimeSpan.milliseconds(10),
        // The first read answers with the samples of when it ran, after the gate opens.
        readRemote: async (tr, keys) => {
          const frame = setup.core.read(tr, keys);
          if (reads++ === 0) await gate.wait();
          return frame;
        },
      });
      const { core, reader, store, read } = setup;
      const caught = reader.catchUp(1, 100);
      await expect.poll(() => reads).toBe(1);
      core.advance(TimeSpan.seconds(20));
      await read(new TimeRange(at(70), at(80)), 1);
      const block = store.raw(1, 101);
      const newer = [block.coveredEnd, block.coveredCount, block.sealed];
      gate.open();
      await caught;
      expect([block.coveredEnd, block.coveredCount, block.sealed]).toEqual(newer);
    });
  });

  describe("budget", () => {
    it("should keep every block it fills past the budget while they are pinned", async () => {
      const { store, read } = createReader({
        store: new Store({ budget: Size.bytes(1) }),
      });
      const tr = new TimeRange(at(60), at(70));
      // Points pins the blocks of its raw range before it fills them.
      indexes(tr, RAW_LEVEL).forEach((i) => store.raw(1, i).pins++);
      const res = await read(tr, 1);
      expect(timestamps(res)).toEqual(expectedTimestamps(0, 132));
    });

    it("should evict the blocks of a back-fill past the budget once it lands", async () => {
      const store = new Store({ budget: Size.bytes(1) });
      const { core, reader } = createReader({ elapsed: 600, store });
      await reader.fill(new TimeRange(at(0), core.now()), 2);
      const cached = [1, 2]
        .flatMap((ch) =>
          Array.from({ length: 12 }, (_, i) => store.findRaw(ch, 100 + i)),
        )
        .filter((b) => b != null);
      expect(cached).toHaveLength(0);
    });
  });

  describe("data type", () => {
    // Converts the series of channel 2 to float64 once the returned flag is set.
    const createConversion = () => {
      const state = { converted: false };
      const answer = (frame: framer.Frame): framer.Frame =>
        frame.map((k, s) => [
          k,
          state.converted && k === 2 ? s.convert(DataType.FLOAT64) : s,
        ]);
      return { state, answer };
    };

    it("should return the samples of a fetch that changes the channel's data type", async () => {
      const { store, read } = createReader();
      store.retainDataType(2, DataType.FLOAT64);
      const res = await read(new TimeRange(at(10), at(20)), 2);
      expect(res.length).toBe(66);
      expect(res.series.map((s) => s.dataType)).toEqual([DataType.FLOAT32]);
    });

    it("should fetch a block again from its start when its data type changes", async () => {
      const { state, answer } = createConversion();
      const { core, fetches, read } = createReader({ elapsed: 30, answer });
      await read(new TimeRange(at(0), at(30)), 2);
      state.converted = true;
      core.advance(TimeSpan.seconds(10));
      const res = await read(new TimeRange(at(0), at(40)), 2);
      expect(fetches.map(({ tr }) => tr)).toEqual([
        blockRange(100),
        new TimeRange(at(29), blockRange(100).end),
        blockRange(100),
      ]);
      expect(res.series.map((s) => s.dataType)).toEqual([DataType.FLOAT64]);
      expect(res.series.map((s) => s.alignment)).toEqual([blockAlignment(100, 0)]);
      expect(res.length).toBe(40);
    });

    it("should drop cached blocks when a fetch changes the data type", async () => {
      const { state, answer } = createConversion();
      const { reader, store, fetches, read } = createReader({ answer });
      await read(new TimeRange(at(0), at(30)), 2);
      state.converted = true;
      const tr = new TimeRange(at(0), at(100));
      await reader.fill(tr, 2);
      expect(store.findRaw(2, 100)).toBeUndefined();
      const res = await read(tr, 2);
      expect(fetches.map(({ tr }) => tr)).toEqual([
        blockRange(100),
        blockRange(101),
        blockRange(100),
      ]);
      expect(res.series.map((s) => s.dataType.toString())).toEqual([
        "float64",
        "float64",
      ]);
      expect(res.length).toBe(132);
    });
  });

  describe("batching", () => {
    it("should fire a batch even while reads keep arriving", async () => {
      const { fetches, read } = createReader({
        batchDebounce: TimeSpan.milliseconds(30),
      });
      const pending = [read(new TimeRange(at(0), at(1)), 2)];
      for (let i = 1; i <= 5; i++) {
        await sleep.sleep(TimeSpan.milliseconds(10));
        pending.push(read(new TimeRange(at(i * 70), at(i * 70 + 1)), 2));
      }
      // A timer reset by each read would still be waiting for 30ms of silence.
      expect(fetches.length).toBeGreaterThan(0);
      await Promise.all(pending);
    });

    it("should join a read to the fetch pending for its block", async () => {
      const gate = createGate();
      const { fetches, read } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        before: gate.wait,
      });
      const tr = new TimeRange(at(10), at(20));
      const first = read(tr, 2);
      await expect.poll(() => fetches.length).toBe(1);
      const second = read(tr, 2);
      await sleep.sleep(TimeSpan.milliseconds(30));
      gate.open();
      expect((await first).series).toEqual((await second).series);
      expect(fetches).toHaveLength(1);
    });

    it("should skip a block that a fetch covered while the request waited", async () => {
      const gate = createGate();
      const { fetches, read } = createReader({
        batchDebounce: TimeSpan.milliseconds(20),
        before: gate.wait,
      });
      const tr = new TimeRange(at(10), at(20));
      const data = read(tr, 2);
      await expect.poll(() => fetches.length).toBe(1);
      // The index's block is not pending, so its read queues a fetch of its own.
      const index = read(tr, 1);
      gate.open();
      await data;
      expect(timestamps(await index)).toEqual(expectedTimestamps(0, 66));
      expect(fetches).toHaveLength(1);
    });
  });

  describe("alignment", () => {
    it("should give fetched series block alignment", async () => {
      const { read } = createReader();
      const tr = new TimeRange(at(70), at(80));
      const [index, data] = await Promise.all([read(tr, 1), read(tr, 2)]);
      expect(index.series.map((s) => s.alignment)).toEqual([blockAlignment(101, 0)]);
      expect(data.series.map((s) => s.alignment)).toEqual([blockAlignment(101, 0)]);
      expect(timestamps(index)[0]).toEqual(at(66).valueOf());
    });

    it("should continue a block's offsets in a tail fetch", async () => {
      const { core, read } = createReader({ elapsed: 30 });
      const first = await read(new TimeRange(at(0), at(30)), 1);
      core.advance(TimeSpan.seconds(10));
      const res = await read(new TimeRange(at(0), at(40)), 1);
      expect(res.series.map((s) => s.alignment)).toEqual([
        blockAlignment(100, 0),
        blockAlignment(100, 30),
      ]);
      expect(res.series[0]).toBe(first.series[0]);
    });

    it("should leave out samples the Core answers before their index", async () => {
      const { read } = createReader({
        // The Core committed the data channel two samples past its index.
        answer: (frame) =>
          new framer.Frame(
            frame.map((k, s) => [k, k === 1 ? s.slice(0, s.length - 2) : s]),
          ),
      });
      const tr = new TimeRange(at(0), at(10));
      const [index, data] = await Promise.all([read(tr, 1), read(tr, 2)]);
      expect(data.length).toBe(64);
      expect(data.series.map((s) => s.alignmentBounds)).toEqual(
        index.series.map((s) => s.alignmentBounds),
      );
    });

    it("should align a channel's tail with the tail of its index", async () => {
      const { core, read } = createReader({ elapsed: 30 });
      await read(new TimeRange(at(0), at(30)), 2);
      core.advance(TimeSpan.seconds(10));
      const tr = new TimeRange(at(0), at(40));
      const data = await read(tr, 2);
      const index = await read(tr, 1);
      expect(data.series.map((s) => s.alignmentBounds)).toEqual(
        index.series.map((s) => s.alignmentBounds),
      );
    });
  });

  describe("failures", () => {
    const failBlock = (index: number) => async (fetch: Fetch) => {
      if (fetch.tr.start.equals(blockRange(index).start))
        throw new Error("read exploded");
    };

    it("should fail only the reads whose fetch failed", async () => {
      const { reader } = createReader({ before: failBlock(101) });
      const [ok, bad] = await Promise.allSettled([
        reader.fill(new TimeRange(at(10), at(20)), 2),
        reader.fill(new TimeRange(at(70), at(80)), 3),
      ]);
      expect(ok.status).toBe("fulfilled");
      expect(bad).toEqual({ status: "rejected", reason: new Error("read exploded") });
    });

    it("should reject a read after its last fetch lands when one fails", async () => {
      const gate = createGate();
      let gated = false;
      const { reader } = createReader({
        before: async (fetch) => {
          if (!gated) return;
          await failBlock(102)(fetch);
          await gate.wait();
        },
      });
      // A covered block 101 splits the read into two fetches.
      await reader.fill(new TimeRange(at(70), at(80)), 2);
      gated = true;
      let settled = false;
      const read = reader.fill(new TimeRange(at(10), at(140)), 2).finally(() => {
        settled = true;
      });
      await sleep.sleep(TimeSpan.milliseconds(100));
      expect(settled).toBe(false);
      gate.open();
      await expect(read).rejects.toThrow("read exploded");
    });

    it("should read each channel alone when a read of several fails", async () => {
      const { read, fetches } = createReader({
        before: async ({ keys }) => {
          if (keys.includes(3)) throw new Error("read exploded");
        },
      });
      const tr = new TimeRange(at(10), at(20));
      const [ok, bad] = await Promise.allSettled([read(tr, 2), read(tr, 3)]);
      expect(ok.status).toBe("fulfilled");
      expect(bad).toEqual({ status: "rejected", reason: new Error("read exploded") });
      expect(fetches.map(({ keys }) => [...keys].sort())).toEqual([
        [1, 2, 3],
        ...[
          [1, 2],
          [1, 3],
        ].map((keys) => expect.arrayContaining(keys)),
      ]);
    });

    it("should fail every channel of a read that could not reach the Core", async () => {
      const { read, fetches } = createReader({
        before: async () => {
          throw new Unreachable({ message: "Core is down" });
        },
      });
      const tr = new TimeRange(at(10), at(20));
      const results = await Promise.allSettled([read(tr, 2), read(tr, 3)]);
      const reason = new Unreachable({ message: "Core is down" });
      expect(results).toEqual([
        { status: "rejected", reason },
        { status: "rejected", reason },
      ]);
      expect(fetches).toHaveLength(1);
    });

    it("should refetch a block whose fetch failed", async () => {
      const { reader, fetches } = createReader({ before: failBlock(100) });
      const tr = new TimeRange(at(10), at(20));
      await expect(reader.fill(tr, 2)).rejects.toThrow("read exploded");
      await expect(reader.fill(tr, 2)).rejects.toThrow("read exploded");
      expect(fetches).toHaveLength(2);
    });

    it("should reject a read of a channel that does not exist", async () => {
      const { reader, fetches } = createReader();
      const read = reader.fill(new TimeRange(at(10), at(20)), 99);
      await expect(read).rejects.toThrow(NotFoundError);
      await expect(read).rejects.toThrow("channel 99 not found");
      expect(fetches).toHaveLength(0);
    });
  });

  describe("hung fetch", () => {
    const hang = (index: number) => async (fetch: Fetch) => {
      if (fetch.tr.start.equals(blockRange(index).start))
        await new Promise<never>(() => {});
    };

    it("should reject a hung read with Unreachable after the fetch timeout", async () => {
      const { reader } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        fetchTimeout: TimeSpan.milliseconds(50),
        before: hang(100),
      });
      const hung = reader.fill(new TimeRange(at(10), at(20)), 2);
      await expect(hung).rejects.toSatisfy((e) => Unreachable.matches(e));
      await expect(hung).rejects.toThrow(
        `block read for ${blockRange(100).toString()} timed out after 50ms`,
      );
    });

    it("should reject a read whose channel retrieve hangs", async () => {
      const { reader, fetches } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        fetchTimeout: TimeSpan.milliseconds(50),
        beforeRetrieve: async () => await new Promise<never>(() => {}),
      });
      const hung = reader.fill(new TimeRange(at(10), at(20)), 2);
      await expect(hung).rejects.toSatisfy((e) => Unreachable.matches(e));
      await expect(hung).rejects.toThrow("retrieve of channels 2 timed out after 50ms");
      expect(fetches).toHaveLength(0);
    });

    it("should change no block with a fetch that lands after its timeout", async () => {
      const gate = createGate();
      const { reader, store } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        fetchTimeout: TimeSpan.milliseconds(50),
        before: gate.wait,
      });
      const late = reader.fill(new TimeRange(at(10), at(20)), 2);
      await expect(late).rejects.toSatisfy((e) => Unreachable.matches(e));
      gate.open();
      await sleep.sleep(TimeSpan.milliseconds(10));
      expect(store.raw(2, 100).coveredEnd).toEqual(blockRange(100).start);
    });

    it("should serve other blocks while a fetch hangs", async () => {
      const { reader, read } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        fetchTimeout: TimeSpan.seconds(5),
        before: hang(100),
      });
      const hung = reader.fill(new TimeRange(at(10), at(20)), 2);
      await sleep.sleep(TimeSpan.milliseconds(30));
      expect(await read(new TimeRange(at(70), at(80)), 3)).toHaveLength(66);
      await reader.close();
      await expect(hung).rejects.toThrow(UnexpectedError);
      await expect(hung).rejects.toThrow("telemetry reader is closed");
    });
  });

  describe("close", () => {
    it("should reject pending reads", async () => {
      const { reader } = createReader({ batchDebounce: TimeSpan.seconds(10) });
      const pending = reader.fill(new TimeRange(at(10), at(20)), 2);
      await sleep.sleep(TimeSpan.milliseconds(5));
      await reader.close();
      await expect(pending).rejects.toThrow(UnexpectedError);
      await expect(pending).rejects.toThrow("telemetry reader is closed");
    });

    it("should reject reads issued after close", async () => {
      const { reader } = createReader();
      await reader.close();
      await expect(reader.fill(new TimeRange(at(10), at(20)), 2)).rejects.toThrow(
        "telemetry reader is closed",
      );
    });

    it("should make no Core read for a fetch that waits on its channels at close", async () => {
      const gate = createGate();
      let retrieving = false;
      const { reader, fetches } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        beforeRetrieve: async () => {
          retrieving = true;
          await gate.wait();
        },
      });
      const read = reader.fill(new TimeRange(at(10), at(20)), 2);
      await expect.poll(() => retrieving).toBe(true);
      await reader.close();
      gate.open();
      await expect(read).rejects.toThrow("telemetry reader is closed");
      await sleep.sleep(TimeSpan.milliseconds(10));
      expect(fetches).toHaveLength(0);
    });

    it("should discard a fetch that lands after close", async () => {
      const gate = createGate();
      const { reader, store, fetches } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        before: gate.wait,
      });
      const hung = reader.fill(new TimeRange(at(10), at(20)), 2);
      await expect.poll(() => fetches.length).toBe(1);
      await reader.close();
      await expect(hung).rejects.toThrow(UnexpectedError);
      await expect(hung).rejects.toThrow("telemetry reader is closed");
      gate.open();
      await sleep.sleep(TimeSpan.milliseconds(10));
      expect(store.raw(2, 100).covers(blockRange(100))).toBe(false);
    });
  });

  describe("Core", () => {
    it("should cover a span the Core answers with no data after one fetch", async () => {
      const client = createTestClient();
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
      const start = TimeStamp.now().sub(TimeSpan.minutes(5));
      await client.write(start, { [time.key]: [start], [data.key]: [1] });
      const fetched: framer.Frame[] = [];
      const store = new Store();
      const reader = new Reader({
        readRemote: async (tr, keys) => {
          const frame = await client.read(tr, keys);
          fetched.push(frame);
          return frame;
        },
        retrieveChannels: async (keys) =>
          (await client.channels.retrieve(keys as channel.Key[])).map((c) => c.payload),
        store,
      });
      const empty = blockRange(rawIndex(start) - 3);
      const tr = new TimeRange(empty.start, empty.start.add(TimeSpan.seconds(10)));
      expect(await readBlocks(reader, store, tr, data.key)).toHaveLength(0);
      expect(await readBlocks(reader, store, tr, data.key)).toHaveLength(0);
      expect(fetched).toHaveLength(1);
    });

    it("should seal only the blocks that the channel's samples pass", async () => {
      const client = createTestClient();
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
      const first = rawIndex(TimeStamp.now().sub(TimeSpan.minutes(10)));
      // One sample per second from 5 seconds into the first block to inside the third.
      const times = Array.from({ length: 146 }, (_, i) =>
        blockRange(first).start.add(TimeSpan.seconds(5 + i)),
      );
      await client.write(times[0], {
        [time.key]: times,
        [data.key]: times.map((_, i) => i),
      });
      const store = new Store();
      const reader = new Reader({
        readRemote: async (tr, keys) => await client.read(tr, keys),
        retrieveChannels: async (keys) =>
          (await client.channels.retrieve(keys as channel.Key[])).map((c) => c.payload),
        store,
      });
      await reader.fill(blockRange(first), data.key);
      await reader.fill(blockRange(first + 2), data.key);
      expect(store.findRaw(data.key, first)?.sealed).toBe(true);
      expect(store.findRaw(data.key, first + 2)?.sealed).toBe(false);
    });
  });
});
