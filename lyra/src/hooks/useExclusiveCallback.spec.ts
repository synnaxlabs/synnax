// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useExclusiveCallback } from "@/hooks";

describe("useExclusiveCallback", () => {
  it("should skip a call while an earlier one is pending, across a re-render", async () => {
    const { promise, resolve } = Promise.withResolvers<void>();
    const func = vi.fn(async () => await promise);
    const { result, rerender } = renderHook(() => useExclusiveCallback(func));
    const first = result.current();
    rerender();
    await result.current();
    expect(func).toHaveBeenCalledOnce();
    resolve();
    await first;
  });

  it("should run a call with its arguments after an earlier one resolves", async () => {
    const func = vi.fn(async (_: string) => {});
    const { result } = renderHook(() => useExclusiveCallback(func));
    await result.current("a");
    await result.current("b");
    expect(func.mock.calls).toEqual([["a"], ["b"]]);
  });

  it("should pass a failure on and run the next call", async () => {
    const func = vi
      .fn<() => Promise<void>>()
      .mockRejectedValueOnce(new Error("add failed"))
      .mockResolvedValueOnce(undefined);
    const { result } = renderHook(() => useExclusiveCallback(func));
    await expect(result.current()).rejects.toThrow("add failed");
    await result.current();
    expect(func).toHaveBeenCalledTimes(2);
  });
});
