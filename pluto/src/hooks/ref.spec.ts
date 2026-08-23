// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useCombinedStateAndRef } from "@/hooks/ref";

describe("useCombinedStateAndRef", () => {
  it("should seed both the state and the ref from the initial value", () => {
    const { result } = renderHook(() => useCombinedStateAndRef<number>(1));
    const [value, , ref] = result.current;
    expect(value).toEqual(1);
    expect(ref.current).toEqual(1);
  });

  it("should run a lazy initializer exactly once", () => {
    const initializer = vi.fn(() => 1);
    const { rerender } = renderHook(() => useCombinedStateAndRef<number>(initializer));
    rerender();
    expect(initializer).toHaveBeenCalledOnce();
  });

  it("should update the state", () => {
    const { result } = renderHook(() => useCombinedStateAndRef<number>(1));
    act(() => result.current[1](2));
    expect(result.current[0]).toEqual(2);
  });

  // React computes the first update of a batch eagerly, so a ref assigned by the state
  // updater rather than the setter reads correctly after a single call and only goes
  // stale once an update is already pending. The pending cases below are the ones that
  // hold the contract.
  describe("ref currency", () => {
    it("should update the ref before the setter returns", () => {
      const { result } = renderHook(() => useCombinedStateAndRef<number>(1));
      const [, setValue, ref] = result.current;
      act(() => {
        setValue(2);
        expect(ref.current).toEqual(2);
      });
    });

    it("should update the ref before the setter returns with an update pending", () => {
      const { result } = renderHook(() => useCombinedStateAndRef<number>(1));
      const [, setValue, ref] = result.current;
      act(() => {
        setValue(2);
        setValue(3);
        expect(ref.current).toEqual(3);
      });
      expect(result.current[0]).toEqual(3);
    });

    it("should compose sequential function updaters within one handler", () => {
      const { result } = renderHook(() => useCombinedStateAndRef<number>(1));
      const [, setValue, ref] = result.current;
      act(() => {
        setValue((prev) => prev + 1);
        setValue((prev) => prev + 1);
        expect(ref.current).toEqual(3);
      });
      expect(result.current[0]).toEqual(3);
    });
  });
});
