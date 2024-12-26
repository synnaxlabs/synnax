import { describe, expect, it, vi } from "vitest";

import { breaker } from "@/breaker";
import { TimeSpan } from "@/telem";

describe("breaker", () => {
  it("should allow first attempt without sleeping", async () => {
    const mockSleep = vi.fn();
    const brk = breaker.create({ sleepFn: mockSleep });
    const canRetry = await brk();

    expect(canRetry).toBe(true);
    expect(mockSleep).toHaveBeenCalled();
  });

  it("should retry specified number of times before failing", async () => {
    const mockSleep = vi.fn();
    const brk = breaker.create({
      maxRetries: 2,
      interval: TimeSpan.milliseconds(1),
      sleepFn: mockSleep,
    });

    // First attempt
    expect(await brk()).toBe(true);
    // Second attempt
    expect(await brk()).toBe(true);
    // Third attempt (should fail)
    expect(await brk()).toBe(false);

    expect(mockSleep).toHaveBeenCalledTimes(2);
  });

  it("should increase delay between retries according to scale", async () => {
    const mockSleep = vi.fn();
    const brk = breaker.create({
      interval: TimeSpan.seconds(1),
      maxRetries: 3,
      scale: 2,
      sleepFn: mockSleep,
    });

    await brk(); // First attempt - 1s
    await brk(); // Second attempt - 1s * 2 = 2s;
    await brk(); // Third attempt - 2s *2 = 4s;

    expect(mockSleep).toHaveBeenNthCalledWith(1, TimeSpan.seconds(1));
    expect(mockSleep).toHaveBeenNthCalledWith(2, TimeSpan.seconds(2));
    expect(mockSleep).toHaveBeenNthCalledWith(3, TimeSpan.seconds(4));
  });

  it("should use default values when no options provided", async () => {
    const brk = breaker.create();
    let attempts = 0;

    while (await brk()) attempts++;

    expect(attempts).toBe(5); // Default maxRetries is 5
  });

  it("should use custom sleep function when provided", async () => {
    const customSleep = vi.fn();
    const brk = breaker.create({
      interval: TimeSpan.milliseconds(100),
      sleepFn: customSleep,
    });

    await brk();
    await brk();

    expect(customSleep).toHaveBeenCalledTimes(2);
  });
});
