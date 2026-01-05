// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { breaker } from "@/breaker";
import { TimeSpan } from "@/telem";

describe("breaker", () => {
  it("should allow first attempt without sleeping", async () => {
    const mockSleep = vi.fn();
    const brk = new breaker.Breaker({ sleepFn: mockSleep });
    const canRetry = await brk.wait();

    expect(canRetry).toBe(true);
    expect(mockSleep).toHaveBeenCalled();
  });

  it("should retry specified number of times before failing", async () => {
    const mockSleep = vi.fn();
    const brk = new breaker.Breaker({
      maxRetries: 2,
      baseInterval: TimeSpan.milliseconds(1),
      sleepFn: mockSleep,
    });

    // First attempt
    expect(await brk.wait()).toBe(true);
    // Second attempt
    expect(await brk.wait()).toBe(true);
    // Third attempt (should fail)
    expect(await brk.wait()).toBe(false);

    expect(mockSleep).toHaveBeenCalledTimes(2);
  });

  it("should increase delay between retries according to scale", async () => {
    const mockSleep = vi.fn();
    const brk = new breaker.Breaker({
      baseInterval: TimeSpan.seconds(1),
      maxRetries: 3,
      scale: 2,
      sleepFn: mockSleep,
    });

    await brk.wait(); // First attempt - 1s
    await brk.wait(); // Second attempt - 1s * 2 = 2s;
    await brk.wait(); // Third attempt - 2s *2 = 4s;

    expect(mockSleep).toHaveBeenNthCalledWith(1, TimeSpan.seconds(1));
    expect(mockSleep).toHaveBeenNthCalledWith(2, TimeSpan.seconds(2));
    expect(mockSleep).toHaveBeenNthCalledWith(3, TimeSpan.seconds(4));
  });

  it("should use custom sleep function when provided", async () => {
    const customSleep = vi.fn();
    const brk = new breaker.Breaker({
      baseInterval: TimeSpan.milliseconds(100),
      sleepFn: customSleep,
    });

    await brk.wait();
    await brk.wait();

    expect(customSleep).toHaveBeenCalledTimes(2);
  });
});
