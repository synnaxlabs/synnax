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
  MultiSeries,
  Series,
  Size,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { RAW_LEVEL, rawIndex, rawKey } from "@/framer/cache/block";
import { tileRange } from "@/framer/cache/grid";
import { type Block, Store } from "@/framer/cache/store";

// Blocks this large make the fixed overhead of each block too small to change how many
// fit in a budget.
const BYTES = 1e4;

interface TestBlock extends Block {
  bytes: number;
  held: boolean;
}

const createBlock = (overrides: Partial<TestBlock> = {}): TestBlock => ({
  channel: 1,
  fetchedAt: TimeStamp.now(),
  bytes: BYTES,
  held: false,
  ...overrides,
});

const BLOCK = tileRange({ level: RAW_LEVEL, index: 100 });

// Returns a streamed write of float32 values stamped one second apart from the start of
// raw block 100, plus offset seconds.
const createWrite = (offset: number, values: number[], alignment: bigint) =>
  new MultiSeries([
    new Series({
      data: new Float32Array(values),
      alignment,
      timeRange: new TimeRange(
        BLOCK.start.add(TimeSpan.seconds(offset)),
        BLOCK.start.add(TimeSpan.seconds(offset + values.length)),
      ),
    }),
  ]);

describe("Store", () => {
  it("should return the block set under a key", () => {
    const store = new Store();
    const block = createBlock();
    store.set("a", block);
    expect(store.get("a")).toBe(block);
    expect(store.get("b")).toBeUndefined();
  });

  it("should replace the block set under a key", () => {
    const store = new Store();
    const next = createBlock();
    store.set("a", createBlock());
    store.set("a", next);
    expect(store.get("a")).toBe(next);
  });

  it("should remove a deleted block", () => {
    const store = new Store();
    store.set("a", createBlock());
    store.delete("a");
    expect(store.get("a")).toBeUndefined();
  });

  describe("budget", () => {
    it("should evict the least recently used block past the budget", () => {
      const store = new Store({ budget: Size.bytes(2.5 * BYTES) });
      store.set("a", createBlock());
      store.set("b", createBlock());
      store.get("a");
      store.set("c", createBlock());
      expect(store.get("a")).toBeDefined();
      expect(store.get("b")).toBeUndefined();
      expect(store.get("c")).toBeDefined();
    });

    it("should never evict the block it sets", () => {
      const store = new Store({ budget: Size.bytes(1) });
      const block = createBlock();
      store.set("a", block);
      expect(store.get("a")).toBe(block);
    });

    it("should keep a held block past the budget", () => {
      const store = new Store({ budget: Size.bytes(1.5 * BYTES) });
      store.set("a", createBlock({ held: true }));
      store.set("b", createBlock());
      store.set("c", createBlock());
      expect(store.get("a")).toBeDefined();
      expect(store.get("b")).toBeUndefined();
    });

    it("should make a held block it skips the most recently used", () => {
      const store = new Store({ budget: Size.bytes(2.5 * BYTES) });
      const held = createBlock({ held: true });
      store.set("a", held);
      store.set("b", createBlock());
      store.set("c", createBlock());
      held.held = false;
      store.set("d", createBlock());
      expect(store.get("a")).toBe(held);
      expect(store.get("c")).toBeUndefined();
    });

    it("should count a block's new size when it is set again", () => {
      const store = new Store({ budget: Size.bytes(2.5 * BYTES) });
      const grown = createBlock();
      store.set("a", grown);
      store.set("b", createBlock());
      grown.bytes = 2 * BYTES;
      store.set("a", grown);
      expect(store.get("a")).toBe(grown);
      expect(store.get("b")).toBeUndefined();
    });

    it("should free the bytes of a deleted block", () => {
      const store = new Store({ budget: Size.bytes(2.5 * BYTES) });
      store.set("a", createBlock());
      store.set("b", createBlock());
      store.delete("a");
      store.set("c", createBlock());
      expect(store.get("b")).toBeDefined();
      expect(store.get("c")).toBeDefined();
    });

    it("should count a fixed overhead for a block with no samples", () => {
      const store = new Store({ budget: Size.bytes(1000) });
      const keys = Array.from({ length: 10 }, (_, i) => `${i}`);
      keys.forEach((key) => store.set(key, createBlock({ bytes: 0 })));
      expect(keys.filter((key) => store.get(key) != null).length).toBeLessThan(10);
    });
  });

  describe("expired", () => {
    it("should report a block fetched before the stale threshold", async () => {
      const store = new Store({ staleThreshold: TimeSpan.milliseconds(20) });
      const block = createBlock();
      expect(store.expired(block)).toBe(false);
      await sleep.sleep(TimeSpan.milliseconds(30));
      expect(store.expired(block)).toBe(true);
    });

    it("should default the stale threshold to ten minutes", () => {
      const store = new Store();
      const fetchedAt = TimeStamp.now().sub(TimeSpan.minutes(9));
      expect(store.expired(createBlock({ fetchedAt }))).toBe(false);
      const old = TimeStamp.now().sub(TimeSpan.minutes(11));
      expect(store.expired(createBlock({ fetchedAt: old }))).toBe(true);
    });
  });

  describe("retainDataType", () => {
    const createStore = (): Store => {
      const store = new Store();
      const channels: channel.Key[] = [1, 1, 2];
      channels.forEach((ch, i) => store.set(`${i}`, createBlock({ channel: ch })));
      store.retainDataType(1, DataType.FLOAT32);
      store.retainDataType(2, DataType.FLOAT32);
      return store;
    };

    it("should keep a channel's blocks while its data type holds", () => {
      const store = createStore();
      store.retainDataType(1, DataType.FLOAT32);
      expect(store.get("0")).toBeDefined();
      expect(store.get("1")).toBeDefined();
    });

    it("should drop every block of a channel whose data type changes", () => {
      const store = createStore();
      store.retainDataType(1, DataType.INT64);
      expect(store.get("0")).toBeUndefined();
      expect(store.get("1")).toBeUndefined();
      expect(store.get("2")).toBeDefined();
    });
  });

  describe("raw", () => {
    it("should cache an empty raw block on first access", () => {
      const store = new Store();
      const block = store.raw(1, 100);
      expect(block.range).toEqual(BLOCK);
      expect(block.fetched).toEqual([]);
      expect(store.get(rawKey(1, 100))).toBe(block);
      expect(store.raw(1, 100)).toBe(block);
    });

    it("should return an expired block without its fetched samples", async () => {
      const store = new Store({ staleThreshold: TimeSpan.milliseconds(20) });
      const block = store.raw(1, 100);
      block.fill(
        [new Series({ data: new Float32Array([1]), timeRange: BLOCK })],
        BLOCK.end,
        1,
      );
      expect(store.raw(1, 100).fetched).toHaveLength(1);
      await sleep.sleep(TimeSpan.milliseconds(30));
      expect(store.raw(1, 100)).toBe(block);
      expect(block.fetched).toEqual([]);
      expect(block.coveredEnd).toEqual(BLOCK.start);
    });
  });

  describe("write", () => {
    it("should write streamed series into the open block of their time", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      store.write(1, createWrite(1, [1, 2], 0n));
      const block = store.raw(1, 100);
      expect(block.open).not.toBeNull();
      expect(store.leadingBuffer(1)).toBe(block.open?.leadingBuffer);
      expect(Array.from(store.leadingBuffer(1) ?? [])).toEqual([1, 2]);
    });

    it("should return only the buffers a write allocates", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      const first = store.write(1, createWrite(1, [1, 2], 0n));
      expect(first.series).toEqual([store.leadingBuffer(1)]);
      const second = store.write(1, createWrite(3, [3], 2n));
      expect(second.series).toEqual([]);
      expect(Array.from(store.leadingBuffer(1) ?? [])).toEqual([1, 2, 3]);
    });

    it("should close the open block when a series starts in the next one", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      store.write(1, createWrite(60, [1, 2], 0n));
      const first = store.raw(1, 100);
      const buffer = store.leadingBuffer(1);
      const allocated = store.write(1, createWrite(70, [3], 2n));
      expect(first.open).toBeNull();
      expect(first.streamed).toEqual([buffer]);
      expect(buffer?.timeRange.end).toEqual(BLOCK.start.add(TimeSpan.seconds(62)));
      const next = store.raw(1, 101);
      expect(next.open).not.toBeNull();
      expect(allocated.series).toEqual([store.leadingBuffer(1)]);
    });

    it("should keep a series that crosses a block edge whole in its first block", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      store.write(1, createWrite(64, [1, 2, 3, 4], 0n));
      expect(store.raw(1, 100).open).not.toBeNull();
      expect(store.raw(1, 101).open).toBeNull();
      expect(Array.from(store.leadingBuffer(1) ?? [])).toEqual([1, 2, 3, 4]);
    });

    it("should move to an earlier block when a series starts before the open one", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      store.write(1, createWrite(1, [1], 0n));
      store.write(1, createWrite(-60, [2], 1n));
      expect(store.raw(1, 100).open).toBeNull();
      expect(store.raw(1, 99).open).not.toBeNull();
    });

    it("should write a series with no time range into the block of the wall clock", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      store.write(1, new MultiSeries([new Series(new Float32Array([1]))]));
      expect(store.raw(1, rawIndex(TimeStamp.now())).open).not.toBeNull();
    });

    it("should never evict an open block", () => {
      const store = new Store({ budget: Size.bytes(1), dynamicBufferSize: 100 });
      store.write(1, createWrite(1, [1], 0n));
      store.set("a", createBlock());
      store.set("b", createBlock());
      expect(store.get(rawKey(1, 100))).toBeDefined();
      expect(store.get("a")).toBeUndefined();
    });

    it("should drop a channel's buffer and blocks when its data type changes", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      store.write(1, createWrite(1, [1], 0n));
      const block = store.raw(1, 100);
      store.write(
        1,
        new MultiSeries([
          new Series({
            data: new Float64Array([2]),
            alignment: 1n,
            timeRange: BLOCK.start.range(BLOCK.start.add(TimeSpan.seconds(1))),
          }),
        ]),
      );
      expect(block.open).toBeNull();
      expect(store.raw(1, 100)).not.toBe(block);
      expect(store.leadingBuffer(1)?.dataType).toEqual(DataType.FLOAT64);
      expect(Array.from(store.leadingBuffer(1) ?? [])).toEqual([2]);
    });
  });

  describe("flush", () => {
    it("should close the open block and keep its buffer readable", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      store.write(1, createWrite(1, [1, 2], 0n));
      const buffer = store.leadingBuffer(1);
      store.flush(1);
      const block = store.raw(1, 100);
      expect(store.leadingBuffer(1)).toBeNull();
      expect(block.open).toBeNull();
      expect(block.streamed).toEqual([buffer]);
      expect(block.covers(BLOCK)).toBe(false);
    });

    it("should do nothing for a channel with no open block", () => {
      const store = new Store();
      store.flush(1);
      expect(store.leadingBuffer(1)).toBeNull();
    });
  });

  it("should drop every block and buffer when it closes", () => {
    const store = new Store({ dynamicBufferSize: 100 });
    store.set("a", createBlock({ held: true }));
    store.write(1, createWrite(1, [1], 0n));
    store.close();
    expect(store.get("a")).toBeUndefined();
    expect(store.get(rawKey(1, 100))).toBeUndefined();
    expect(store.leadingBuffer(1)).toBeNull();
  });
});
