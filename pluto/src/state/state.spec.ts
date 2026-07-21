// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { state } from "@/state";

describe("state", () => {
  describe("usePassthrough", () => {
    it("should return the internal state and setter when uncontrolled", () => {
      const { result } = renderHook(() => state.usePassthrough<number>({ initial: 5 }));
      expect(result.current[0]).toBe(5);
      act(() => result.current[1](10));
      expect(result.current[0]).toBe(10);
    });

    it("should lazily evaluate a function initial", () => {
      const initial = vi.fn(() => 7);
      const { result } = renderHook(() => state.usePassthrough<number>({ initial }));
      expect(initial).toHaveBeenCalledTimes(1);
      expect(result.current[0]).toBe(7);
    });

    it("should return the controlled value and onChange when both are provided", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        state.usePassthrough<number>({ initial: 0, value: 42, onChange }),
      );
      expect(result.current[0]).toBe(42);
      expect(result.current[1]).toBe(onChange);
    });

    it("should update internal state and notify onChange when uncontrolled", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        state.usePassthrough<number>({ initial: 3, onChange }),
      );
      act(() => result.current[1](8));
      expect(result.current[0]).toBe(8);
      expect(onChange).toHaveBeenCalledTimes(1);
      expect(onChange).toHaveBeenCalledWith(8);
    });

    it("should forward function updaters to onChange when uncontrolled", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        state.usePassthrough<number>({ initial: 1, onChange }),
      );
      const updater = (prev: number) => prev + 1;
      act(() => result.current[1](updater));
      expect(result.current[0]).toBe(2);
      expect(onChange).toHaveBeenCalledWith(updater);
    });

    it("should keep a stable setter identity across state updates", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        state.usePassthrough<number>({ initial: 0, onChange }),
      );
      const first = result.current[1];
      act(() => result.current[1](1));
      expect(result.current[1]).toBe(first);
    });
  });

  describe("usePurePassthrough", () => {
    it("should return the internal state and setter when uncontrolled", () => {
      const { result } = renderHook(() =>
        state.usePurePassthrough<number>({ initialValue: 5 }),
      );
      expect(result.current[0]).toBe(5);
      act(() => result.current[1](10));
      expect(result.current[0]).toBe(10);
    });

    it("should evaluate a function initial value", () => {
      const { result } = renderHook(() =>
        state.usePurePassthrough<number>({ initialValue: () => 9 }),
      );
      expect(result.current[0]).toBe(9);
    });

    it("should update internal state and notify onChange when uncontrolled", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        state.usePurePassthrough<number>({ initialValue: 0, onChange }),
      );
      act(() => result.current[1](11));
      expect(result.current[0]).toBe(11);
      expect(onChange).toHaveBeenCalledWith(11);
    });

    it("should return the controlled value and onChange when both are provided", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        state.usePurePassthrough<number>({ initialValue: 0, value: 42, onChange }),
      );
      expect(result.current[0]).toBe(42);
      expect(result.current[1]).toBe(onChange);
    });

    it("should keep a stable setter identity across state updates", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        state.usePurePassthrough<number>({ initialValue: 0, onChange }),
      );
      const first = result.current[1];
      act(() => result.current[1](1));
      expect(result.current[1]).toBe(first);
    });
  });

  describe("usePersisted", () => {
    beforeEach(() => localStorage.clear());

    it("should use the initial value when nothing is persisted", () => {
      const { result } = renderHook(() => state.usePersisted(5, "missing-key"));
      expect(result.current[0]).toBe(5);
    });

    it("should read a previously persisted value on mount", () => {
      localStorage.setItem("stored-key", JSON.stringify(42));
      const { result } = renderHook(() => state.usePersisted(5, "stored-key"));
      expect(result.current[0]).toBe(42);
    });

    it("should persist updates to localStorage", () => {
      const { result } = renderHook(() => state.usePersisted<number>(1, "write-key"));
      act(() => result.current[1](7));
      expect(result.current[0]).toBe(7);
      expect(localStorage.getItem("write-key")).toBe(JSON.stringify(7));
    });

    it("should surface a persisted value to a freshly mounted hook", () => {
      const first = renderHook(() => state.usePersisted<number>(1, "shared-key"));
      act(() => first.result.current[1](99));
      const second = renderHook(() => state.usePersisted<number>(1, "shared-key"));
      expect(second.result.current[0]).toBe(99);
    });

    it("should evaluate a function initial when nothing is persisted", () => {
      const { result } = renderHook(() =>
        state.usePersisted(() => ({ count: 3 }), "lazy-key"),
      );
      expect(result.current[0]).toEqual({ count: 3 });
    });
  });
});
