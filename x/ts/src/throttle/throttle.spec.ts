// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TimeSpan } from "@/telem/telem";
import { throttle } from "@/throttle/throttle";

describe("throttle", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should execute immediately on the leading edge", () => {
    const fn = vi.fn();
    const throttled = throttle(fn, TimeSpan.milliseconds(100));
    expect(fn).toHaveBeenCalledTimes(0);
    throttled(10);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(10);
  });

  it("should throttle repeated calls during the wait period", () => {
    const fn = vi.fn();
    const throttled = throttle(fn, TimeSpan.milliseconds(100));
    expect(fn).toHaveBeenCalledTimes(0);
    throttled(10);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(10);
    throttled(20);
    throttled(30);
    throttled(40);
    expect(fn).toHaveBeenCalledTimes(1);
  });

  it("should run a trailing call with the latest arguments", () => {
    const fn = vi.fn();
    const throttled = throttle(fn, TimeSpan.milliseconds(100));
    expect(fn).toHaveBeenCalledTimes(0);
    throttled(10);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(10);
    throttled(20);
    throttled(30);
    throttled(40);
    expect(fn).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenLastCalledWith(40);
  });

  it("should schedule the trailing call after only the remaining wait time", () => {
    const fn = vi.fn();
    const throttled = throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(fn).toHaveBeenLastCalledWith(10);
    vi.advanceTimersByTime(90);
    throttled(20);
    expect(fn).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(9);
    expect(fn).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(1);
    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenLastCalledWith(20);
  });

  it("should not schedule multiple trailing calls", () => {
    const fn = vi.fn();
    const throttled = throttle(fn, TimeSpan.milliseconds(100));
    throttled(10);
    throttled(20);
    throttled(30);

    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(1, 10);
    expect(fn).toHaveBeenNthCalledWith(2, 30);

    vi.advanceTimersByTime(100);
    expect(fn).toHaveBeenCalledTimes(2);
  });

  it("should execute again immediately after the wait period has passed", () => {
    const fn = vi.fn();
    const throttled = throttle(fn, TimeSpan.milliseconds(100));

    throttled(10);
    vi.advanceTimersByTime(100);
    throttled(20);

    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(1, 10);
    expect(fn).toHaveBeenNthCalledWith(2, 20);
  });

  it("should preserve null arguments", () => {
    const fn = vi.fn();
    const throttled = throttle(fn, TimeSpan.milliseconds(100));
    throttled(null);
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

    const throttled = throttle(fn, TimeSpan.milliseconds(100));

    throttled(10);
    throttled(20);

    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledTimes(2);
    expect(fn).toHaveBeenNthCalledWith(1, 10);
    expect(fn).toHaveBeenNthCalledWith(2, 20);

    vi.advanceTimersByTime(100);

    expect(fn).toHaveBeenCalledTimes(3);
    expect(fn).toHaveBeenNthCalledWith(3, 30);
  });

  it("should not throttle the execution of the given function if the time is 0", () => {
    const fn = vi.fn();
    const throttled = throttle(fn, TimeSpan.ZERO);
    expect(throttled).toBe(fn);
  });
});
