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

import { useDestructors } from "@/hooks/useDestructors";

describe("useDestructors", () => {
  it("should call cleanup on unmount", () => {
    const destructor = vi.fn();
    const { result, unmount } = renderHook(() => useDestructors());

    result.current.set(destructor);
    expect(destructor).not.toHaveBeenCalled();

    unmount();
    expect(destructor).toHaveBeenCalledTimes(1);
  });

  it("should handle multiple destructors", () => {
    const destructor1 = vi.fn();
    const destructor2 = vi.fn();
    const destructor3 = vi.fn();

    const { result, unmount } = renderHook(() => useDestructors());

    result.current.set(destructor1);
    result.current.set([destructor2, destructor3]);

    expect(destructor1).not.toHaveBeenCalled();
    expect(destructor2).not.toHaveBeenCalled();
    expect(destructor3).not.toHaveBeenCalled();

    unmount();

    expect(destructor1).toHaveBeenCalledTimes(1);
    expect(destructor2).toHaveBeenCalledTimes(1);
    expect(destructor3).toHaveBeenCalledTimes(1);
  });

  it("should clear destructors after cleanup", () => {
    const destructor = vi.fn();
    const { result } = renderHook(() => useDestructors());

    result.current.set(destructor);
    result.current.cleanup();

    expect(destructor).toHaveBeenCalledTimes(1);

    result.current.cleanup();
    expect(destructor).toHaveBeenCalledTimes(1);
  });

  it("should handle undefined destructors", () => {
    const { result, unmount } = renderHook(() => useDestructors());

    result.current.set(undefined);
    expect(() => unmount()).not.toThrow();
  });

  it("should handle null destructors", () => {
    const { result, unmount } = renderHook(() => useDestructors());

    result.current.set(null as any);
    expect(() => unmount()).not.toThrow();
  });

  it("should maintain the same cleanup function reference across renders", () => {
    const { result, rerender } = renderHook(() => useDestructors());

    const cleanup1 = result.current.cleanup;
    const set1 = result.current.set;

    rerender();

    const cleanup2 = result.current.cleanup;
    const set2 = result.current.set;

    expect(cleanup1).toBe(cleanup2);
    expect(set1).toBe(set2);
  });

  it("should accumulate destructors from multiple calls", () => {
    const destructor1 = vi.fn();
    const destructor2 = vi.fn();
    const { result, unmount } = renderHook(() => useDestructors());

    result.current.set(destructor1);
    result.current.set(destructor2);

    unmount();

    expect(destructor1).toHaveBeenCalledTimes(1);
    expect(destructor2).toHaveBeenCalledTimes(1);
  });

  it("should execute destructors in the order they were added", () => {
    const order: number[] = [];
    const destructor1 = vi.fn(() => order.push(1));
    const destructor2 = vi.fn(() => order.push(2));
    const destructor3 = vi.fn(() => order.push(3));

    const { result, unmount } = renderHook(() => useDestructors());

    result.current.set(destructor1);
    result.current.set(destructor2);
    result.current.set(destructor3);

    unmount();

    expect(order).toEqual([1, 2, 3]);
  });

  it("should handle errors in destructors", () => {
    const errorDestructor = vi.fn(() => {
      throw new Error("Test error");
    });
    const normalDestructor = vi.fn();

    const { result, unmount } = renderHook(() => useDestructors());

    result.current.set(normalDestructor);
    result.current.set(errorDestructor);

    expect(() => unmount()).toThrow("Test error");

    expect(normalDestructor).toHaveBeenCalledTimes(1);
    expect(errorDestructor).toHaveBeenCalledTimes(1);
  });

  it("should handle manual cleanup call", () => {
    const destructor1 = vi.fn();
    const destructor2 = vi.fn();
    const { result } = renderHook(() => useDestructors());

    result.current.set(destructor1);
    result.current.cleanup();
    expect(destructor1).toHaveBeenCalledTimes(1);

    result.current.set(destructor2);
    result.current.cleanup();
    expect(destructor1).toHaveBeenCalledTimes(1);
    expect(destructor2).toHaveBeenCalledTimes(1);
  });
});
