// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { sync } from "@/sync";
import { TimeSpan } from "@/telem";

describe("withTimeout", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should return the value of a promise that settles in time", async () => {
    const result = sync.withTimeout(
      Promise.resolve(1),
      TimeSpan.seconds(1),
      () => new Error("late"),
    );
    expect(await result).toEqual(1);
    expect(vi.getTimerCount()).toEqual(0);
  });

  it("should throw the error of a promise that fails in time", async () => {
    const failed = sync.withTimeout(
      Promise.reject(new Error("fetch exploded")),
      TimeSpan.seconds(1),
      () => new Error("late"),
    );
    await expect(failed).rejects.toThrow(new Error("fetch exploded"));
    expect(vi.getTimerCount()).toEqual(0);
  });

  it("should throw the error of the factory when the timeout passes first", async () => {
    const request = Promise.withResolvers<number>();
    const late = sync.withTimeout(
      request.promise,
      TimeSpan.seconds(1),
      () => new RangeError("too slow"),
    );
    const settled = expect(late).rejects.toThrow(new RangeError("too slow"));
    await vi.advanceTimersByTimeAsync(1000);
    await settled;
  });

  it("should throw the error that an async factory resolves", async () => {
    const request = Promise.withResolvers<number>();
    const late = sync.withTimeout(
      request.promise,
      TimeSpan.seconds(1),
      async () => await Promise.resolve(new Error("names resolved")),
    );
    const settled = expect(late).rejects.toThrow(new Error("names resolved"));
    await vi.advanceTimersByTimeAsync(1000);
    await settled;
  });

  it("should not settle before the timeout passes", async () => {
    const request = Promise.withResolvers<number>();
    let settled = false;
    sync
      .withTimeout(request.promise, TimeSpan.seconds(1), () => new Error("late"))
      .catch(() => (settled = true));
    await vi.advanceTimersByTimeAsync(999);
    expect(settled).toBe(false);
    await vi.advanceTimersByTimeAsync(1);
    expect(settled).toBe(true);
  });

  it("should not surface a late rejection of the promise", async () => {
    const rejections: unknown[] = [];
    const onRejection = (reason: unknown) => rejections.push(reason);
    process.on("unhandledRejection", onRejection);
    try {
      const request = Promise.withResolvers<number>();
      const late = sync.withTimeout(
        request.promise,
        TimeSpan.seconds(1),
        () => new Error("too slow"),
      );
      const settled = expect(late).rejects.toThrow(new Error("too slow"));
      await vi.advanceTimersByTimeAsync(1000);
      await settled;
      request.reject(new Error("fetch exploded"));
      vi.useRealTimers();
      await new Promise((resolve) => setTimeout(resolve, 10));
    } finally {
      process.off("unhandledRejection", onRejection);
    }
    expect(rejections).toEqual([]);
  });
});
