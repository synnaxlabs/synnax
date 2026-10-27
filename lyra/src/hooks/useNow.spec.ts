// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useNow } from "@/hooks";

describe("useNow", () => {
  beforeEach(() => {
    vi.useFakeTimers({ now: new Date(2026, 0, 1, 12) });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should advance at the instant next returns", () => {
    const { result } = renderHook(() => useNow((now) => now.add(TimeSpan.SECOND)));
    const start = result.current;
    act(() => {
      vi.advanceTimersByTime(TimeSpan.SECOND.milliseconds - 1);
    });
    expect(result.current).toBe(start);
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(result.current.equals(start.add(TimeSpan.SECOND))).toBe(true);
  });

  it("should hold the time when next returns null", () => {
    const { result } = renderHook(() => useNow(() => null));
    const start = result.current;
    act(() => {
      vi.advanceTimersByTime(TimeSpan.DAY.milliseconds);
    });
    expect(result.current).toBe(start);
    expect(vi.getTimerCount()).toBe(0);
  });

  it("should catch up before paint when the instant has already passed", () => {
    const start = TimeStamp.now();
    const initialProps: { at: TimeStamp | null } = { at: null };
    const { result, rerender } = renderHook(
      ({ at }) => useNow((now) => (at?.after(now) === true ? at : null)),
      { initialProps },
    );
    vi.setSystemTime(start.add(TimeSpan.HOUR).date());
    rerender({ at: start.add(TimeSpan.minutes(30)) });
    expect(result.current.equals(start.add(TimeSpan.HOUR))).toBe(true);
  });

  it("should re-arm a far instant instead of firing at once", () => {
    const at = TimeStamp.now().add(TimeSpan.days(30));
    const { result } = renderHook(() => useNow((now) => (at.after(now) ? at : null)));
    const start = result.current;
    act(() => {
      vi.advanceTimersByTime(TimeSpan.MINUTE.milliseconds);
    });
    expect(result.current).toBe(start);
    act(() => {
      vi.advanceTimersByTime(TimeSpan.days(25).milliseconds);
    });
    expect(result.current.before(at)).toBe(true);
    act(() => {
      vi.advanceTimersToNextTimer();
    });
    expect(result.current.equals(at)).toBe(true);
  });
});
