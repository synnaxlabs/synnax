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
  id,
  MultiSeries,
  Rate,
  Series,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { NotFoundError, UnexpectedError } from "@/errors";
import { RAW_LEVEL, rawIndex } from "@/framer/cache/block";
import { tileRange } from "@/framer/cache/grid";
import { Reader, type ReaderProps } from "@/framer/cache/reader";
import { Store } from "@/framer/cache/store";
import { type Frame } from "@/framer/frame";
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
}

interface SetupProps extends Partial<ReaderProps> {
  /** Seconds after START on the Core's clock. */
  elapsed?: number;
  /** Runs before each fetch is answered. A rejection fails the fetch. */
  before?: (fetch: Fetch) => Promise<void>;
}

// Returns a reader over a Core with one index channel (1) and two data channels (2 and
// 3) on it, recording every fetch the reader makes.
const createReader = ({ elapsed = 300, before, ...props }: SetupProps = {}): Setup => {
  const core = new SyntheticCore({
    rate: Rate.hz(1),
    start: START,
    now: at(elapsed),
    channels: 2,
  });
  const fetches: Fetch[] = [];
  const store = props.store ?? new Store();
  const reader = new Reader({
    readRemote: async (tr, keys) => {
      const fetch = { tr, keys };
      fetches.push(fetch);
      await before?.(fetch);
      return core.read(tr, keys);
    },
    retrieveChannels: core.retrieveChannels,
    now: core.now,
    ...props,
    store,
  });
  return { core, reader, store, fetches };
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
      const { reader, fetches } = createReader();
      const res = await reader.read(new TimeRange(at(10), at(20)), 2);
      expect(fetches).toEqual([{ tr: blockRange(100), keys: [2, 1] }]);
      expect(res.length).toBe(66);
    });

    it("should serve a covered read from the cache", async () => {
      const { reader, fetches } = createReader();
      const tr = new TimeRange(at(10), at(20));
      const first = await reader.read(tr, 2);
      const second = await reader.read(tr, 2);
      expect(fetches).toHaveLength(1);
      expect(second.series).toEqual(first.series);
    });

    it("should fetch each block a read spans", async () => {
      const { reader, fetches } = createReader();
      const res = await reader.read(new TimeRange(at(60), at(70)), 1);
      expect(fetches.map(({ tr }) => tr)).toEqual([blockRange(100), blockRange(101)]);
      expect(timestamps(res)).toEqual(expectedTimestamps(0, 132));
    });

    it("should share one Core read among channels reading one block", async () => {
      const { reader, fetches } = createReader();
      const tr = new TimeRange(at(10), at(20));
      await Promise.all([reader.read(tr, 2), reader.read(tr, 3)]);
      expect(fetches).toHaveLength(1);
      expect(new Set(fetches[0].keys)).toEqual(new Set([1, 2, 3]));
    });

    it("should cover a block the Core answers with no data", async () => {
      const { reader, fetches } = createReader();
      const tr = new TimeRange(at(-30), at(-20));
      expect(await reader.read(tr, 2)).toHaveLength(0);
      expect(await reader.read(tr, 2)).toHaveLength(0);
      expect(fetches).toHaveLength(1);
    });

    it("should refetch the tail of a block too recent to commit on the Core's clock", async () => {
      const { core, reader, fetches } = createReader({ elapsed: 30 });
      await reader.read(new TimeRange(at(0), at(30)), 1);
      core.advance(TimeSpan.seconds(10));
      const res = await reader.read(new TimeRange(at(0), at(40)), 1);
      expect(fetches.map(({ tr }) => tr)).toEqual([
        blockRange(100),
        new TimeRange(at(29), blockRange(100).end),
      ]);
      expect(timestamps(res)).toEqual(expectedTimestamps(0, 40));
    });

    it("should refetch a block once its fetched samples expire", async () => {
      const { reader, fetches } = createReader({
        store: new Store({ staleThreshold: TimeSpan.milliseconds(50) }),
      });
      const tr = new TimeRange(at(10), at(20));
      await reader.read(tr, 2);
      await reader.read(tr, 2);
      expect(fetches).toHaveLength(1);
      await sleep.sleep(TimeSpan.milliseconds(60));
      expect(await reader.read(tr, 2)).toHaveLength(66);
      expect(fetches).toHaveLength(2);
    });

    it("should run at most four fetches at once", async () => {
      const gate = createGate();
      let active = 0;
      let most = 0;
      const { reader, fetches } = createReader({
        before: async () => {
          most = Math.max(most, ++active);
          await gate.wait();
          active--;
        },
      });
      const read = reader.read(new TimeRange(at(0), blockRange(106).start), 2);
      await expect.poll(() => fetches.length).toBe(4);
      await sleep.sleep(TimeSpan.milliseconds(20));
      expect(fetches).toHaveLength(4);
      gate.open();
      await read;
      expect(fetches).toHaveLength(6);
      expect(most).toBe(4);
    });
  });

  describe("batching", () => {
    it("should fire a batch even while reads keep arriving", async () => {
      const { reader, fetches } = createReader({
        batchDebounce: TimeSpan.milliseconds(30),
      });
      const pending = [reader.read(new TimeRange(at(0), at(1)), 2)];
      for (let i = 1; i <= 5; i++) {
        await sleep.sleep(TimeSpan.milliseconds(10));
        pending.push(reader.read(new TimeRange(at(i * 70), at(i * 70 + 1)), 2));
      }
      // A timer reset by each read would still be waiting for 30ms of silence.
      expect(fetches.length).toBeGreaterThan(0);
      await Promise.all(pending);
    });

    it("should join a read to the fetch pending for its block", async () => {
      const gate = createGate();
      const { reader, fetches } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        before: gate.wait,
      });
      const tr = new TimeRange(at(10), at(20));
      const first = reader.read(tr, 2);
      await expect.poll(() => fetches.length).toBe(1);
      const second = reader.read(tr, 2);
      await sleep.sleep(TimeSpan.milliseconds(30));
      gate.open();
      expect((await first).series).toEqual((await second).series);
      expect(fetches).toHaveLength(1);
    });

    it("should skip a block that a fetch covered while the request waited", async () => {
      const gate = createGate();
      const { reader, fetches } = createReader({
        batchDebounce: TimeSpan.milliseconds(20),
        before: gate.wait,
      });
      const tr = new TimeRange(at(10), at(20));
      const data = reader.read(tr, 2);
      await expect.poll(() => fetches.length).toBe(1);
      // The index's block is not pending, so its read queues a fetch of its own.
      const index = reader.read(tr, 1);
      gate.open();
      await data;
      expect(timestamps(await index)).toEqual(expectedTimestamps(0, 66));
      expect(fetches).toHaveLength(1);
    });
  });

  describe("alignment", () => {
    it("should give fetched series block alignment", async () => {
      const { reader } = createReader();
      const tr = new TimeRange(at(70), at(80));
      const [index, data] = await Promise.all([reader.read(tr, 1), reader.read(tr, 2)]);
      expect(index.series.map((s) => s.alignment)).toEqual([blockAlignment(101, 0)]);
      expect(data.series.map((s) => s.alignment)).toEqual([blockAlignment(101, 0)]);
      expect(timestamps(index)[0]).toEqual(at(66).valueOf());
    });

    it("should continue a block's offsets in a tail fetch", async () => {
      const { core, reader } = createReader({ elapsed: 30 });
      const first = await reader.read(new TimeRange(at(0), at(30)), 1);
      core.advance(TimeSpan.seconds(10));
      const res = await reader.read(new TimeRange(at(0), at(40)), 1);
      expect(res.series.map((s) => s.alignment)).toEqual([
        blockAlignment(100, 0),
        blockAlignment(100, 30),
      ]);
      expect(res.series[0]).toBe(first.series[0]);
    });

    it("should align a channel's tail with the tail of its index", async () => {
      const { core, reader } = createReader({ elapsed: 30 });
      await reader.read(new TimeRange(at(0), at(30)), 2);
      core.advance(TimeSpan.seconds(10));
      const tr = new TimeRange(at(0), at(40));
      const data = await reader.read(tr, 2);
      const index = await reader.read(tr, 1);
      expect(data.series.map((s) => s.alignmentBounds)).toEqual(
        index.series.map((s) => s.alignmentBounds),
      );
    });
  });

  describe("streamed samples", () => {
    const createStreamed = (from: number, to: number): MultiSeries =>
      new MultiSeries([
        new Series({
          data: new Float32Array(Array.from({ length: to - from }, (_, i) => i)),
          alignment: 1n << 60n,
          timeRange: new TimeRange(at(from), at(to)),
        }),
      ]);

    it("should return streamed samples the fetched ones do not overlap, in time order", async () => {
      const { reader, store, fetches } = createReader({ elapsed: 30 });
      store.write(2, createStreamed(40, 42));
      const res = await reader.read(new TimeRange(at(0), at(60)), 2);
      expect(fetches).toHaveLength(1);
      expect(res.series).toHaveLength(2);
      expect(res.series[1]).toBe(store.leadingBuffer(2));
    });

    it("should fetch a span the stream holds", async () => {
      const { reader, store, fetches } = createReader({ elapsed: 30 });
      store.write(2, createStreamed(40, 42));
      const tr = new TimeRange(at(40), at(42));
      await reader.read(tr, 2);
      await reader.read(tr, 2);
      expect(fetches).toHaveLength(2);
    });

    it("should leave out a streamed series that a fetched series overlaps", async () => {
      const { reader, store } = createReader({ elapsed: 30 });
      store.write(2, createStreamed(10, 12));
      const res = await reader.read(new TimeRange(at(0), at(20)), 2);
      expect(res.series).toHaveLength(1);
      expect(res.series[0]).not.toBe(store.leadingBuffer(2));
      expect(res.length).toBe(30);
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
        reader.read(new TimeRange(at(10), at(20)), 2),
        reader.read(new TimeRange(at(70), at(80)), 3),
      ]);
      expect(ok.status).toBe("fulfilled");
      expect(bad).toEqual({ status: "rejected", reason: new Error("read exploded") });
    });

    it("should reject a read after its last fetch lands when one fails", async () => {
      const gate = createGate();
      const { reader } = createReader({
        before: async (fetch) => {
          await failBlock(101)(fetch);
          await gate.wait();
        },
      });
      let settled = false;
      const read = reader.read(new TimeRange(at(60), at(70)), 2).finally(() => {
        settled = true;
      });
      await sleep.sleep(TimeSpan.milliseconds(100));
      expect(settled).toBe(false);
      gate.open();
      await expect(read).rejects.toThrow("read exploded");
    });

    it("should refetch a block whose fetch failed", async () => {
      const { reader, fetches } = createReader({ before: failBlock(100) });
      const tr = new TimeRange(at(10), at(20));
      await expect(reader.read(tr, 2)).rejects.toThrow("read exploded");
      await expect(reader.read(tr, 2)).rejects.toThrow("read exploded");
      expect(fetches).toHaveLength(2);
    });

    it("should reject a read of a channel that does not exist", async () => {
      const { reader, fetches } = createReader();
      const read = reader.read(new TimeRange(at(10), at(20)), 99);
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
      const hung = reader.read(new TimeRange(at(10), at(20)), 2);
      await expect(hung).rejects.toSatisfy((e) => Unreachable.matches(e));
      await expect(hung).rejects.toThrow(
        `block read for ${blockRange(100).toString()} timed out after 50ms`,
      );
    });

    it("should serve other blocks while a fetch hangs", async () => {
      const { reader } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        fetchTimeout: TimeSpan.seconds(5),
        before: hang(100),
      });
      const hung = reader.read(new TimeRange(at(10), at(20)), 2);
      await sleep.sleep(TimeSpan.milliseconds(30));
      expect(await reader.read(new TimeRange(at(70), at(80)), 3)).toHaveLength(66);
      await reader.close();
      await expect(hung).rejects.toThrow(UnexpectedError);
    });
  });

  describe("close", () => {
    it("should reject pending reads", async () => {
      const { reader } = createReader({ batchDebounce: TimeSpan.seconds(10) });
      const pending = reader.read(new TimeRange(at(10), at(20)), 2);
      await sleep.sleep(TimeSpan.milliseconds(5));
      await reader.close();
      await expect(pending).rejects.toThrow(UnexpectedError);
      await expect(pending).rejects.toThrow("telemetry reader is closed");
    });

    it("should reject reads issued after close", async () => {
      const { reader } = createReader();
      await reader.close();
      await expect(reader.read(new TimeRange(at(10), at(20)), 2)).rejects.toThrow(
        "telemetry reader is closed",
      );
    });

    it("should discard a fetch that lands after close", async () => {
      const gate = createGate();
      const { reader, store, fetches } = createReader({
        batchDebounce: TimeSpan.milliseconds(10),
        before: gate.wait,
      });
      const hung = reader.read(new TimeRange(at(10), at(20)), 2);
      await expect.poll(() => fetches.length).toBe(1);
      await reader.close();
      await expect(hung).rejects.toThrow(UnexpectedError);
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
      const fetched: Frame[] = [];
      const reader = new Reader({
        readRemote: async (tr, keys) => {
          const frame = await client.read(tr, keys);
          fetched.push(frame);
          return frame;
        },
        retrieveChannels: async (keys) =>
          (await client.channels.retrieve(keys as channel.Key[])).map((c) => c.payload),
        store: new Store(),
      });
      const empty = blockRange(rawIndex(start) - 3);
      const tr = new TimeRange(empty.start, empty.start.add(TimeSpan.seconds(10)));
      expect(await reader.read(tr, data.key)).toHaveLength(0);
      expect(await reader.read(tr, data.key)).toHaveLength(0);
      expect(fetched).toHaveLength(1);
    });
  });
});
