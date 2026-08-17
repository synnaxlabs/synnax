// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useHeldChecking } from "@/platform/connection/useHeldChecking";

describe("useHeldChecking", () => {
  beforeEach(() => vi.useFakeTimers());

  afterEach(() => vi.useRealTimers());

  it("reports a check as soon as one starts", () => {
    const { result, rerender } = renderHook(
      (checking: boolean) => useHeldChecking(checking),
      {
        initialProps: false,
      },
    );
    expect(result.current).toBe(false);
    rerender(true);
    expect(result.current).toBe(true);
  });

  it("holds a check that ends immediately", () => {
    const { result, rerender } = renderHook(
      (checking: boolean) => useHeldChecking(checking),
      {
        initialProps: true,
      },
    );
    rerender(false);
    expect(result.current).toBe(true);
    act(() => void vi.advanceTimersByTime(1000));
    expect(result.current).toBe(true);
    act(() => void vi.advanceTimersByTime(500));
    expect(result.current).toBe(false);
  });

  it("keeps reporting a check that starts again inside the hold", () => {
    const { result, rerender } = renderHook(
      (checking: boolean) => useHeldChecking(checking),
      {
        initialProps: true,
      },
    );
    rerender(false);
    act(() => void vi.advanceTimersByTime(1000));
    rerender(true);
    act(() => void vi.advanceTimersByTime(5000));
    expect(result.current).toBe(true);
  });
});
