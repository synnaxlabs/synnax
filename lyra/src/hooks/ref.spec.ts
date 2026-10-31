// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render, renderHook } from "@testing-library/react";
import { createElement, type RefObject, useLayoutEffect } from "react";
import { assert, describe, expect, it, vi } from "vitest";

import { useCombinedRefs, useCombinedStateAndRef, useSyncedRef } from "@/hooks";

describe("useSyncedRef", () => {
  it("should update the ref before a child's layout effect reads it", () => {
    const seen: number[] = [];
    interface ChildProps {
      synced: RefObject<number>;
      value: number;
    }
    const Child = ({ synced, value }: ChildProps) => {
      useLayoutEffect(() => {
        seen.push(synced.current);
      }, [synced, value]);
      return null;
    };
    const Parent = ({ value }: { value: number }) =>
      createElement(Child, { synced: useSyncedRef(value), value });
    const { rerender } = render(createElement(Parent, { value: 1 }));
    rerender(createElement(Parent, { value: 2 }));
    expect(seen).toEqual([1, 2]);
  });
});

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

describe("useCombinedRefs", () => {
  it("should set every ref, then clear each one when the element detaches", () => {
    const objectRef = { current: null as HTMLElement | null };
    const callbackRef = vi.fn();
    const { result } = renderHook(() =>
      useCombinedRefs<HTMLElement>(objectRef, callbackRef),
    );
    const el = document.createElement("div");
    const cleanup = result.current(el);
    expect(objectRef.current).toBe(el);
    expect(callbackRef).toHaveBeenCalledExactlyOnceWith(el);
    assert(typeof cleanup === "function");
    cleanup();
    expect(objectRef.current).toBeNull();
    expect(callbackRef).toHaveBeenLastCalledWith(null);
  });

  it("should run a callback ref's own cleanup instead of calling it with null", () => {
    const release = vi.fn();
    const callbackRef = vi.fn(() => release);
    const { result } = renderHook(() => useCombinedRefs<HTMLElement>(callbackRef));
    const cleanup = result.current(document.createElement("div"));
    assert(typeof cleanup === "function");
    cleanup();
    expect(release).toHaveBeenCalledOnce();
    expect(callbackRef).toHaveBeenCalledOnce();
  });
});
