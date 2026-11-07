// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { scheduler } from "@/scheduler";
import { TimeSpan } from "@/telem";

describe("throttle", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should run the first call on the next task", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    expect(fn).not.toHaveBeenCalled();
    vi.advanceTimersByTime(0);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(10);
  });

  it("should run the calls of one task once with the latest arguments", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    throttled(20);
    throttled(30);
    vi.advanceTimersByTime(0);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(30);
  });

  it("should throttle repeated calls during the wait period", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    vi.advanceTimersByTime(0);
    throttled(20);
    throttled(30);
    throttled(40);
    vi.advanceTimersByTime(99);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it("should run a trailing call with the latest arguments", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    vi.advanceTimersByTime(0);
    throttled(20);
    throttled(30);
    throttled(40);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenLastCalledWith(40);
  });

  it("should schedule the trailing call after only the remaining wait time", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    vi.advanceTimersByTime(90);
    throttled(20);
    vi.advanceTimersByTime(9);
    expect(fn).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(1);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenLastCalledWith(20);
  });

  it("should not schedule multiple trailing calls", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    vi.advanceTimersByTime(0);
    throttled(20);
    throttled(30);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(1, 10);
    expect(fn).toHaveBeenNthCalledWith(2, 30);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(2);
  });

  it("should run on the next task once the wait period has passed", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    vi.advanceTimersByTime(100);
    throttled(20);
    expect(fn).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(0);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(2, 20);
  });

  it("should run on the next task after the clock moves backwards", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    vi.advanceTimersByTime(0);
    vi.setSystemTime(Date.now() - 1000);
    throttled(20);
    vi.advanceTimersByTime(0);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenLastCalledWith(20);
  });

  it("should preserve null arguments", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(null);
    vi.advanceTimersByTime(0);
    throttled(20);
    throttled(null);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(1, null);
    expect(fn).toHaveBeenNthCalledWith(2, null);
  });

  it("should clear pending state before invoking the function", () => {
    const fn = vi.fn();
    fn.mockImplementation((value: number) => {
      if (value === 20) throttled(30);
    });
    const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    vi.advanceTimersByTime(0);
    throttled(20);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(2, 20);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(3);
    expect(fn).toHaveBeenNthCalledWith(3, 30);
  });

  it("should invoke the function synchronously when the wait period is 0", () => {
    const fn = vi.fn();
    const throttled = scheduler.throttle(fn, TimeSpan.ZERO);
    throttled(10);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(10);
  });

  describe("cancel", () => {
    it("should drop a pending call", () => {
      const fn = vi.fn();
      const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
      throttled(10);
      throttled.cancel();
      vi.advanceTimersByTime(200);
      expect(fn).not.toHaveBeenCalled();
    });

    it("should drop a pending trailing call", () => {
      const fn = vi.fn();
      const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
      throttled(10);
      vi.advanceTimersByTime(0);
      throttled(20);
      throttled.cancel();
      vi.advanceTimersByTime(200);
      expect(fn).toHaveBeenCalledTimes(1);
      expect(fn).toHaveBeenLastCalledWith(10);
    });

    it("should allow a new call after cancellation", () => {
      const fn = vi.fn();
      const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
      throttled(10);
      throttled.cancel();
      throttled(20);
      vi.advanceTimersByTime(0);
      expect(fn).toHaveBeenCalledTimes(1);
      expect(fn).toHaveBeenLastCalledWith(20);
    });

    it("should be a no-op for a zero-wait throttle", () => {
      const fn = vi.fn();
      const throttled = scheduler.throttle(fn, TimeSpan.ZERO);
      expect(() => throttled.cancel()).not.toThrow();
      throttled(10);
      expect(fn).toHaveBeenCalledTimes(1);
    });
  });

  describe("flush", () => {
    it("should run a pending call at once with the latest arguments", () => {
      const fn = vi.fn();
      const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
      throttled(10);
      throttled(20);
      throttled.flush();
      expect(fn).toHaveBeenCalledTimes(1);
      expect(fn).toHaveBeenLastCalledWith(20);
      vi.advanceTimersByTime(200);
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it("should run a pending trailing call without waiting for the period", () => {
      const fn = vi.fn();
      const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
      throttled(10);
      vi.advanceTimersByTime(0);
      throttled(20);
      throttled.flush();
      expect(fn).toHaveBeenCalledTimes(2);
      expect(fn).toHaveBeenLastCalledWith(20);
    });

    it("should start the wait period at the flushed call", () => {
      const fn = vi.fn();
      const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
      throttled(10);
      throttled.flush();
      throttled(20);
      vi.advanceTimersByTime(99);
      expect(fn).toHaveBeenCalledTimes(1);
      vi.advanceTimersByTime(1);
      expect(fn).toHaveBeenCalledTimes(2);
      expect(fn).toHaveBeenLastCalledWith(20);
    });

    it("should be a no-op when no call is pending", () => {
      const fn = vi.fn();
      const throttled = scheduler.throttle(fn, TimeSpan.milliseconds(100));
      throttled.flush();
      expect(fn).not.toHaveBeenCalled();
    });
  });
});
