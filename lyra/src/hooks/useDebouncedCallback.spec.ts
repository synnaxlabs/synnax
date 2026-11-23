// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useDebouncedCallback } from "@/hooks";

const WAIT = TimeSpan.milliseconds(10);

describe("useDebouncedCallback", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should keep a pending call through a re-render with the same deps", () => {
    const func = vi.fn();
    const { result, rerender } = renderHook(
      ({ dep }) => useDebouncedCallback(func, WAIT, [dep]),
      { initialProps: { dep: 1 } },
    );
    const first = result.current;
    first("a");
    rerender({ dep: 1 });
    expect(result.current).toBe(first);
    vi.advanceTimersByTime(WAIT.milliseconds);
    expect(func).toHaveBeenCalledExactlyOnceWith("a");
  });

  it("should return a new function when the deps change", () => {
    const { result, rerender } = renderHook(
      ({ dep }) => useDebouncedCallback(vi.fn(), WAIT, [dep]),
      { initialProps: { dep: 1 } },
    );
    const first = result.current;
    rerender({ dep: 2 });
    expect(result.current).not.toBe(first);
  });
});
