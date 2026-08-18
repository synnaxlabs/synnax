// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { aether } from "@/aether/aether";

const flushed = (): Promise<void> => new Promise((resolve) => queueMicrotask(resolve));

describe("Batcher", () => {
  it("should deliver everything buffered in one tick as a single batch", async () => {
    const flushTo = vi.fn();
    const batcher = new aether.Batcher<string>(flushTo);
    batcher.send("a");
    batcher.send("b");
    expect(flushTo).not.toHaveBeenCalled();
    await flushed();
    expect(flushTo).toHaveBeenCalledTimes(1);
    expect(flushTo).toHaveBeenCalledWith(["a", "b"], []);
  });

  it("should collect the transferables of every send in the batch", async () => {
    const flushTo = vi.fn();
    const batcher = new aether.Batcher<string>(flushTo);
    const first = new ArrayBuffer(8);
    const second = new ArrayBuffer(8);
    batcher.send("a", [first]);
    batcher.send("b");
    batcher.send("c", [second]);
    await flushed();
    expect(flushTo).toHaveBeenCalledWith(["a", "b", "c"], [first, second]);
  });

  it("should start a new batch for sends made after a flush", async () => {
    const flushTo = vi.fn();
    const batcher = new aether.Batcher<string>(flushTo);
    batcher.send("a");
    await flushed();
    batcher.send("b");
    await flushed();
    expect(flushTo).toHaveBeenCalledTimes(2);
    expect(flushTo).toHaveBeenNthCalledWith(1, ["a"], []);
    expect(flushTo).toHaveBeenNthCalledWith(2, ["b"], []);
  });

  describe("clear", () => {
    it("should drop buffered values without flushing them", async () => {
      const flushTo = vi.fn();
      const batcher = new aether.Batcher<string>(flushTo);
      batcher.send("a");
      batcher.clear();
      await flushed();
      expect(flushTo).not.toHaveBeenCalled();
    });

    it("should leave the pending flush scheduled so a later send does not double it", async () => {
      const flushTo = vi.fn();
      const beforeFlush = vi.fn();
      const batcher = new aether.Batcher<string>(flushTo, beforeFlush);
      batcher.send("a");
      batcher.clear();
      batcher.send("b");
      await flushed();
      expect(beforeFlush).toHaveBeenCalledTimes(1);
      expect(flushTo).toHaveBeenCalledTimes(1);
      expect(flushTo).toHaveBeenCalledWith(["b"], []);
    });
  });

  describe("beforeFlush", () => {
    it("should send into the batch it precedes", async () => {
      const flushTo = vi.fn();
      const batcher: aether.Batcher<string> = new aether.Batcher<string>(flushTo, () =>
        batcher.send("leading"),
      );
      batcher.send("buffered");
      await flushed();
      expect(flushTo).toHaveBeenCalledTimes(1);
      expect(flushTo).toHaveBeenCalledWith(["buffered", "leading"], []);
    });

    it("should run on a schedule with an empty buffer without calling flushTo", async () => {
      const flushTo = vi.fn();
      const beforeFlush = vi.fn();
      const batcher = new aether.Batcher<string>(flushTo, beforeFlush);
      batcher.schedule();
      await flushed();
      expect(beforeFlush).toHaveBeenCalledTimes(1);
      expect(flushTo).not.toHaveBeenCalled();
    });

    it("should collapse repeated schedules within a tick into one flush", async () => {
      const flushTo = vi.fn();
      const beforeFlush = vi.fn();
      const batcher = new aether.Batcher<string>(flushTo, beforeFlush);
      batcher.schedule();
      batcher.schedule();
      batcher.send("a");
      await flushed();
      expect(beforeFlush).toHaveBeenCalledTimes(1);
      expect(flushTo).toHaveBeenCalledTimes(1);
    });
  });

  it("should defer a send made from inside flushTo to the next batch", async () => {
    const batches: string[][] = [];
    let resent = false;
    const batcher: aether.Batcher<string> = new aether.Batcher<string>((values) => {
      batches.push(values);
      if (resent) return;
      resent = true;
      batcher.send("re-sent");
    });
    batcher.send("first");
    await flushed();
    await flushed();
    expect(batches).toEqual([["first"], ["re-sent"]]);
  });
});
