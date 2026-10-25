// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Size, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
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

    it("should keep a held block past the budget", () => {
      const store = new Store({ budget: Size.bytes(1.5 * BYTES) });
      store.set("a", createBlock({ held: true }));
      store.set("b", createBlock());
      expect(store.get("a")).toBeDefined();
      expect(store.get("b")).toBeUndefined();
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

  it("should drop every block when it closes", () => {
    const store = new Store();
    store.set("a", createBlock({ held: true }));
    store.close();
    expect(store.get("a")).toBeUndefined();
  });
});
