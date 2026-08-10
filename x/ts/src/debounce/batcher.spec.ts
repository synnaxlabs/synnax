// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Batcher, type Entry } from "@/debounce/batcher";
import { TimeSpan } from "@/telem/telem";

const resolveAll = (entries: Array<Entry<number, number>>): void =>
  entries.forEach(({ req, resolve }) => resolve(req * 2));

describe("Batcher", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should coalesce enqueues within the window into one exec call", async () => {
    const exec = vi.fn(resolveAll);
    const batcher = new Batcher<number, number>({
      interval: TimeSpan.milliseconds(100),
      exec,
    });
    const results = Promise.all([batcher.enqueue(1), batcher.enqueue(2)]);
    expect(exec).toHaveBeenCalledTimes(0);
    await vi.advanceTimersByTimeAsync(100);
    expect(exec).toHaveBeenCalledTimes(1);
    expect(exec.mock.calls[0][0].map(({ req }) => req)).toEqual([1, 2]);
    expect(await results).toEqual([2, 4]);
  });

  it("should not reset the window on later enqueues", async () => {
    const exec = vi.fn(resolveAll);
    const batcher = new Batcher<number, number>({
      interval: TimeSpan.milliseconds(100),
      exec,
    });
    const first = batcher.enqueue(1);
    await vi.advanceTimersByTimeAsync(60);
    const second = batcher.enqueue(2);
    await vi.advanceTimersByTimeAsync(40);
    expect(exec).toHaveBeenCalledTimes(1);
    expect(await Promise.all([first, second])).toEqual([2, 4]);
  });

  it("should open a new window after the previous one fires", async () => {
    const exec = vi.fn(resolveAll);
    const batcher = new Batcher<number, number>({
      interval: TimeSpan.milliseconds(100),
      exec,
    });
    const first = batcher.enqueue(1);
    await vi.advanceTimersByTimeAsync(100);
    const second = batcher.enqueue(2);
    await vi.advanceTimersByTimeAsync(100);
    expect(exec).toHaveBeenCalledTimes(2);
    expect(await Promise.all([first, second])).toEqual([2, 4]);
  });

  it("should reject unsettled entries when exec throws", async () => {
    const batcher = new Batcher<number, number>({
      interval: TimeSpan.milliseconds(100),
      exec: (entries) => {
        entries[0].resolve(42);
        throw new Error("exec exploded");
      },
    });
    const first = expect(batcher.enqueue(1)).resolves.toEqual(42);
    const second = expect(batcher.enqueue(2)).rejects.toThrow("exec exploded");
    await vi.advanceTimersByTimeAsync(100);
    await first;
    await second;
  });

  it("should reject entries when an async exec rejects", async () => {
    const batcher = new Batcher<number>({
      interval: TimeSpan.milliseconds(100),
      exec: async () => {
        throw new Error("fetch exploded");
      },
    });
    const pending = expect(batcher.enqueue(1)).rejects.toThrow("fetch exploded");
    await vi.advanceTimersByTimeAsync(100);
    await pending;
  });

  it("should reject queued entries on close and stay usable", async () => {
    const exec = vi.fn(resolveAll);
    const batcher = new Batcher<number, number>({
      interval: TimeSpan.milliseconds(100),
      exec,
    });
    const orphan = expect(batcher.enqueue(1)).rejects.toThrow("closed");
    batcher.close(new Error("closed"));
    await orphan;
    await vi.advanceTimersByTimeAsync(100);
    expect(exec).toHaveBeenCalledTimes(0);
    const revived = batcher.enqueue(2);
    await vi.advanceTimersByTimeAsync(100);
    expect(await revived).toEqual(4);
  });
});
