// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, Size, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { alignmentOf, type Block, RAW_LEVEL, rawKey } from "@/framer/cache/block";
import { tileRange } from "@/framer/cache/grid";
import { Store } from "@/framer/cache/store";
import { UnexpectedError } from "@/index";
import { ManualClock } from "@/testutil";

// Blocks this large make the fixed overhead of each block too small to change how many
// fit in a budget.
const BYTES = 1e4;

interface TestBlock extends Block {
  bytes: number;
  held: boolean;
}

const createBlock = (overrides: Partial<TestBlock> = {}): TestBlock => ({
  channel: 1,
  range: TimeRange.ZERO,
  pins: 0,
  fetchedAt: TimeStamp.now(),
  bytes: BYTES,
  held: false,
  sealed: false,
  ...overrides,
});

const BLOCK = tileRange({ level: RAW_LEVEL, index: 100 });

// Returns a piece of float32 values at the block offset, stamped one second apart from
// the start of raw block 100.
const createPiece = (offset: number, values: number[]): Series =>
  new Series({
    data: new Float32Array(values),
    alignment: alignmentOf(100, offset),
    timeRange: new TimeRange(
      BLOCK.start.add(TimeSpan.seconds(offset)),
      BLOCK.start.add(TimeSpan.seconds(offset + values.length)),
    ),
  });

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

    it("should evict a block past the budget once its fetch settles", async () => {
      const store = new Store({ budget: Size.bytes(1) });
      const block = store.raw(1, 100);
      let settle!: () => void;
      const fetch = store.hold(
        block,
        new Promise<void>((resolve) => (settle = resolve)),
      );
      store.raw(1, 101);
      expect(store.findRaw(1, 100)).toBe(block);
      settle();
      await fetch;
      expect(block.pending).toBeNull();
      expect(store.findRaw(1, 100)).toBeUndefined();
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
    it("should report a block fetched the stale threshold ago or earlier", () => {
      const clock = new ManualClock();
      const store = new Store({
        staleThreshold: TimeSpan.milliseconds(20),
        now: clock.now,
      });
      const block = createBlock({ fetchedAt: clock.now() });
      clock.advance(TimeSpan.milliseconds(19));
      expect(store.expired(block)).toBe(false);
      clock.advance(TimeSpan.milliseconds(1));
      expect(store.expired(block)).toBe(true);
    });

    it("should never expire a held block", () => {
      const clock = new ManualClock();
      const store = new Store({ now: clock.now });
      const block = createBlock({ fetchedAt: clock.now(), held: true });
      clock.advance(TimeSpan.hours(1));
      expect(store.expired(block)).toBe(false);
    });

    it("should keep only a sealed block valid while history signals stream", () => {
      const clock = new ManualClock();
      const store = new Store({ now: clock.now, node: () => 1 });
      store.setChannel(
        new channel.Channel({
          key: 1,
          name: "data",
          dataType: "float32",
          leaseholder: 1,
        }).payload,
      );
      store.setSignalKey(99);
      store.setAcked(new Set([99]));
      const sealed = createBlock({ fetchedAt: clock.now(), sealed: true });
      const unsealed = createBlock({ fetchedAt: clock.now() });
      clock.advance(TimeSpan.hours(1));
      expect(store.expired(sealed)).toBe(false);
      expect(store.expired(unsealed)).toBe(true);
    });

    it("should default the stale threshold to ten minutes", () => {
      const clock = new ManualClock();
      const store = new Store({ now: clock.now });
      const block = createBlock({ fetchedAt: clock.now() });
      clock.advance(TimeSpan.minutes(9));
      expect(store.expired(block)).toBe(false);
      clock.advance(TimeSpan.minutes(1));
      expect(store.expired(block)).toBe(true);
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

    it("should close the channel's open block when its data type changes", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      const block = store.raw(1, 100);
      store.retainDataType(1, DataType.FLOAT32);
      store.append(block, createPiece(0, [1]));
      expect(store.retainDataType(1, DataType.FLOAT64)).toBe(true);
      expect(block.buffer).toBeNull();
      expect(block.joined).toBe(false);
      expect(store.open(1)).toBeNull();
      expect(store.raw(1, 100)).not.toBe(block);
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

    it("should throw when the block under a raw key is not a raw block", () => {
      const store = new Store();
      store.set(rawKey(1, 100), createBlock());
      expect(() => store.raw(1, 100)).toThrow(UnexpectedError);
      expect(() => store.raw(1, 100)).toThrow("block 1/raw/100 is not a raw block");
    });

    it("should return an expired block without its fetched samples", () => {
      const clock = new ManualClock();
      const store = new Store({
        staleThreshold: TimeSpan.milliseconds(20),
        now: clock.now,
      });
      const block = store.raw(1, 100);
      block.fill(
        [new Series({ data: new Float32Array([1]), timeRange: BLOCK })],
        BLOCK.end,
        1,
      );
      expect(store.raw(1, 100).fetched).toHaveLength(1);
      clock.advance(TimeSpan.milliseconds(20));
      expect(store.raw(1, 100)).toBe(block);
      expect(block.fetched).toEqual([]);
      expect(block.coveredEnd).toEqual(BLOCK.start);
    });

    it("should keep the fetched samples of a pinned block past the stale threshold", () => {
      const clock = new ManualClock();
      const store = new Store({
        staleThreshold: TimeSpan.milliseconds(20),
        now: clock.now,
      });
      const series = new Series({ data: new Float32Array([1]), timeRange: BLOCK });
      store.raw(1, 100).fill([series], BLOCK.end, 1);
      store.raw(1, 100).pins++;
      clock.advance(TimeSpan.milliseconds(20));
      expect(store.raw(1, 100).fetched).toEqual([series]);
      store.raw(1, 100).pins--;
      expect(store.raw(1, 100).fetched).toEqual([]);
    });
  });

  describe("append", () => {
    it("should join the block and cover it to the end of the piece", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      const block = store.raw(1, 100);
      store.append(block, createPiece(0, [1, 2]));
      store.append(block, createPiece(2, [3]));
      expect(store.open(1)).toBe(block);
      expect(block.joined).toBe(true);
      expect(block.coveredCount).toBe(3);
      expect(block.coveredEnd).toEqual(BLOCK.start.add(TimeSpan.seconds(3)));
      expect(block.sealed).toBe(true);
      expect(Array.from(block.buffer?.leadingBuffer ?? [])).toEqual([1, 2, 3]);
    });

    it("should seal the open block complete when a piece goes past it", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      const first = store.raw(1, 100);
      store.append(first, createPiece(0, [1, 2]));
      const next = store.raw(1, 101);
      store.append(
        next,
        new Series({ ...createPiece(0, [3]), alignment: alignmentOf(101, 0) }),
      );
      expect(store.open(1)).toBe(next);
      expect(first.joined).toBe(false);
      expect(first.coveredEnd).toEqual(BLOCK.end);
      expect(Array.from(first.fetched[0])).toEqual([1, 2]);
    });

    it("should move the epoch when a block's series change, not when they grow", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      const block = store.raw(1, 100);
      const opened = store.epoch(1);
      store.append(block, createPiece(0, [1, 2]));
      const grown = store.epoch(1);
      expect(grown).toBeGreaterThan(opened);
      store.append(block, createPiece(2, [3]));
      expect(store.epoch(1)).toBe(grown);
      store.flush(1);
      expect(store.epoch(1)).toBeGreaterThan(grown);
      expect(store.epoch(2)).toBe(0);
    });

    it("should seal the open block incomplete on a flush", () => {
      const store = new Store({ dynamicBufferSize: 100 });
      const block = store.raw(1, 100);
      store.append(block, createPiece(0, [1, 2]));
      store.flush(1);
      expect(store.open(1)).toBeNull();
      expect(block.joined).toBe(false);
      expect(block.coveredEnd).toEqual(BLOCK.start.add(TimeSpan.seconds(2)));
      expect(block.coveredCount).toBe(2);
      expect(block.fetched.flatMap((s) => Array.from(s))).toEqual([1, 2]);
    });

    it("should do nothing on a flush of a channel with no open block", () => {
      const store = new Store();
      store.flush(1);
      expect(store.open(1)).toBeNull();
    });

    it("should never evict an open block", () => {
      const store = new Store({ budget: Size.bytes(1), dynamicBufferSize: 100 });
      store.append(store.raw(1, 100), createPiece(0, [1]));
      store.set("a", createBlock());
      store.set("b", createBlock());
      expect(store.get(rawKey(1, 100))).toBeDefined();
      expect(store.get("a")).toBeUndefined();
    });
  });

  it("should drop every block and buffer when it closes", () => {
    const store = new Store({ dynamicBufferSize: 100 });
    store.set("a", createBlock({ held: true }));
    store.append(store.raw(1, 100), createPiece(0, [1]));
    store.close();
    expect(store.get("a")).toBeUndefined();
    expect(store.get(rawKey(1, 100))).toBeUndefined();
    expect(store.open(1)).toBeNull();
  });
});
