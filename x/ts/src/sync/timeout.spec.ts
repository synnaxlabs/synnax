// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { withTimeout } from "@/sync/timeout";
import { TimeSpan } from "@/telem";

const deadline = (): Error => new Error("deadline");

describe("withTimeout", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("should return the value when the promise settles in time", async () => {
    await expect(
      withTimeout(Promise.resolve(1), TimeSpan.seconds(1), deadline),
    ).resolves.toEqual(1);
  });

  it("should propagate the promise's own rejection", async () => {
    await expect(
      withTimeout(Promise.reject(new Error("original")), TimeSpan.seconds(1), deadline),
    ).rejects.toThrow("original");
  });

  it("should reject with the timeout error when the deadline passes", async () => {
    const res = withTimeout(new Promise(() => {}), TimeSpan.seconds(1), deadline);
    const settled = expect(res).rejects.toThrow("deadline");
    await vi.advanceTimersByTimeAsync(1_500);
    await settled;
  });

  it("should wait forever on a non-positive span", async () => {
    let resolve!: (v: number) => void;
    const pending = new Promise<number>((r) => (resolve = r));
    const res = withTimeout(pending, TimeSpan.ZERO, deadline);
    await vi.advanceTimersByTimeAsync(TimeSpan.hours(1).milliseconds);
    resolve(2);
    await expect(res).resolves.toEqual(2);
  });

  it("should not leave a rejected loser unhandled", async () => {
    const unhandled = vi.fn();
    process.on("unhandledRejection", unhandled);
    const late = new Promise<never>((_, reject) =>
      setTimeout(() => reject(new Error("late")), 2_000),
    );
    const res = withTimeout(late, TimeSpan.seconds(1), deadline);
    const settled = expect(res).rejects.toThrow("deadline");
    await vi.advanceTimersByTimeAsync(3_000);
    await settled;
    await vi.advanceTimersByTimeAsync(0);
    expect(unhandled).not.toHaveBeenCalled();
    process.off("unhandledRejection", unhandled);
  });
});
