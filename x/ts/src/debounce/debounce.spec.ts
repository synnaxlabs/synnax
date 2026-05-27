// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { debounce } from "@/debounce/debounce";
import { TimeSpan } from "@/telem/telem";

describe("debounce", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should not execute immediately", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(10);

    expect(fn).toHaveBeenCalledTimes(0);
  });

  it("should execute after the wait period", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(10);

    vi.advanceTimersByTime(99);
    expect(fn).toHaveBeenCalledTimes(0);

    vi.advanceTimersByTime(1);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(10);
  });

  it("should reset the timer on repeated calls", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(10);
    vi.advanceTimersByTime(90);

    debounced(20);
    vi.advanceTimersByTime(90);

    expect(fn).toHaveBeenCalledTimes(0);

    vi.advanceTimersByTime(10);

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(20);
  });

  it("should run with the latest arguments", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(10);
    debounced(20);
    debounced(30);
    debounced(40);

    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(40);
  });

  it("should not execute before the full wait period since the latest call", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(10);
    vi.advanceTimersByTime(50);

    debounced(20);
    vi.advanceTimersByTime(99);

    expect(fn).toHaveBeenCalledTimes(0);

    vi.advanceTimersByTime(1);

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(20);
  });

  it("should only execute once for a burst of calls", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(10);
    vi.advanceTimersByTime(25);
    debounced(20);
    vi.advanceTimersByTime(25);
    debounced(30);
    vi.advanceTimersByTime(25);
    debounced(40);

    vi.advanceTimersByTime(99);
    expect(fn).toHaveBeenCalledTimes(0);

    vi.advanceTimersByTime(1);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(40);

    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it("should execute again after a previous debounce cycle completes", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(10);
    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenNthCalledWith(1, 10);

    debounced(20);
    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(2, 20);
  });

  it("should preserve null arguments", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(10);
    debounced(null);
    debounced(20);
    debounced(null);

    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(null);
  });

  it("should preserve multiple arguments", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(1, "a", false);
    debounced(2, "b", true);

    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(2, "b", true);
  });

  it("should clear pending state before invoking the function", () => {
    const fn = vi.fn();
    fn.mockImplementation((value: number) => {
      if (value === 20) debounced(30);
    });

    const debounced = debounce(fn, TimeSpan.milliseconds(100));

    debounced(10);
    debounced(20);

    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenNthCalledWith(1, 20);

    vi.advanceTimersByTime(99);
    expect(fn).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(1);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(2, 30);
  });

  it("should invoke the function synchronously when the wait period is 0", () => {
    const fn = vi.fn();
    const debounced = debounce(fn, TimeSpan.ZERO);

    debounced(10);

    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(10);
  });

  describe("cancel", () => {
    it("should drop a pending invocation", () => {
      const fn = vi.fn();
      const debounced = debounce(fn, TimeSpan.milliseconds(100));

      debounced(10);
      debounced.cancel();

      vi.advanceTimersByTime(200);

      expect(fn).not.toHaveBeenCalled();
    });

    it("should be a no-op when no invocation is pending", () => {
      const fn = vi.fn();
      const debounced = debounce(fn, TimeSpan.milliseconds(100));

      expect(() => debounced.cancel()).not.toThrow();

      debounced(10);
      vi.advanceTimersByTime(100);

      expect(fn).toHaveBeenCalledTimes(1);
      expect(fn).toHaveBeenLastCalledWith(10);
    });

    it("should be a no-op for a zero-wait debounce", () => {
      const fn = vi.fn();
      const debounced = debounce(fn, TimeSpan.ZERO);

      expect(() => debounced.cancel()).not.toThrow();

      debounced(10);
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it("should allow scheduling a new invocation after cancellation", () => {
      const fn = vi.fn();
      const debounced = debounce(fn, TimeSpan.milliseconds(100));

      debounced(10);
      debounced.cancel();
      vi.advanceTimersByTime(100);
      expect(fn).not.toHaveBeenCalled();

      debounced(20);
      vi.advanceTimersByTime(100);

      expect(fn).toHaveBeenCalledTimes(1);
      expect(fn).toHaveBeenLastCalledWith(20);
    });
  });

  describe("flush", () => {
    it("should invoke a pending call immediately with the latest arguments", () => {
      const fn = vi.fn();
      const debounced = debounce(fn, TimeSpan.milliseconds(100));

      debounced(10);
      debounced(20);
      debounced.flush();

      expect(fn).toHaveBeenCalledTimes(1);
      expect(fn).toHaveBeenLastCalledWith(20);
    });

    it("should be a no-op when no invocation is pending", () => {
      const fn = vi.fn();
      const debounced = debounce(fn, TimeSpan.milliseconds(100));

      debounced.flush();

      expect(fn).not.toHaveBeenCalled();
    });

    it("should clear the pending timer so the function does not fire again later", () => {
      const fn = vi.fn();
      const debounced = debounce(fn, TimeSpan.milliseconds(100));

      debounced(10);
      debounced.flush();
      vi.advanceTimersByTime(200);

      expect(fn).toHaveBeenCalledTimes(1);
    });
  });
});
