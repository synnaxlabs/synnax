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

import { Connection } from "@/platform/connection";

describe("useCountdown", () => {
  beforeEach(() => {
    vi.useFakeTimers({ now: new Date(2026, 0, 1, 12) });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should drop a second as soon as one fewer remains", () => {
    const at = TimeStamp.now().add(TimeSpan.milliseconds(5300));
    const { result } = renderHook(() => Connection.useCountdown(at));
    expect(result.current).toBe(6);
    act(() => {
      vi.advanceTimersByTime(299);
    });
    expect(result.current).toBe(6);
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(result.current).toBe(5);
  });

  it("should stop at zero", () => {
    const at = TimeStamp.now().add(TimeSpan.seconds(2));
    const { result } = renderHook(() => Connection.useCountdown(at));
    act(() => {
      vi.advanceTimersByTime(TimeSpan.seconds(3).milliseconds);
    });
    expect(result.current).toBe(0);
    expect(vi.getTimerCount()).toBe(0);
  });
});
