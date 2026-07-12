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
  describe("isSetter", () => {
    it("should return true for a function", () => {
      const setter = (prev: number) => prev + 1;
      expect(state.isSetter(setter)).toBe(true);
    });

    it("should return false for a value", () => {
      expect(state.isSetter(42)).toBe(false);
      expect(state.isSetter("hello")).toBe(false);
      expect(state.isSetter({ key: "value" })).toBe(false);
    });
  });

  describe("executeSetter", () => {
    it("should return the value directly if not a function", () => {
      expect(state.executeSetter(42, 0)).toBe(42);
      expect(state.executeSetter("hello", "")).toBe("hello");
    });

    it("should call the function with prev value if a function", () => {
      const setter = (prev: number) => prev + 10;
      expect(state.executeSetter(setter, 5)).toBe(15);
    });
  });

  describe("skipUndefined", () => {
    it("should return undefined if the input is undefined", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = state.skipUndefined(setter);
      expect(wrapped(undefined)).toBe(undefined);
    });

    it("should return undefined if the input is null", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = state.skipUndefined(setter);
      expect(wrapped(null as unknown as undefined)).toBe(undefined);
    });

    it("should apply the function if the input is defined", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = state.skipUndefined(setter);
      expect(wrapped(5)).toBe(6);
    });

    it("should work with object state", () => {
      const setter = (prev: { count: number }) => ({ count: prev.count + 1 });
      const wrapped = state.skipUndefined(setter);
      expect(wrapped({ count: 10 })).toEqual({ count: 11 });
      expect(wrapped(undefined)).toBe(undefined);
    });
  });

  describe("skipNull", () => {
    it("should return null if the input is null", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = state.skipNull(setter);
      expect(wrapped(null)).toBe(null);
    });

    it("should return null if the input is undefined", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = state.skipNull(setter);
      expect(wrapped(undefined as unknown as null)).toBe(null);
    });

    it("should apply the function if the input is not null", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = state.skipNull(setter);
      expect(wrapped(5)).toBe(6);
    });

    it("should work with object state", () => {
      const setter = (prev: { name: string }) => ({ name: prev.name.toUpperCase() });
      const wrapped = state.skipNull(setter);
      expect(wrapped({ name: "test" })).toEqual({ name: "TEST" });
      expect(wrapped(null)).toBe(null);
    });
  });

  describe("isInitialSetter", () => {
    it("should return true for a function", () => {
      const initializer = () => 42;
      expect(state.isInitialSetter(initializer)).toBe(true);
    });

    it("should return false for a value", () => {
      expect(state.isInitialSetter(42)).toBe(false);
      expect(state.isInitialSetter("hello")).toBe(false);
    });
  });

  describe("executeInitialSetter", () => {
    it("should return the value directly if not a function", () => {
      expect(state.executeInitialSetter(42)).toBe(42);
      expect(state.executeInitialSetter("hello")).toBe("hello");
    });

    it("should call the function if a function", () => {
      const initializer = () => 42;
      expect(state.executeInitialSetter(initializer)).toBe(42);
    });

    it("should work with lazy initialization", () => {
      let called = false;
      const initializer = () => {
        called = true;
        return { data: "initialized" };
      };
      expect(called).toBe(false);
      const result = state.executeInitialSetter(initializer);
      expect(called).toBe(true);
      expect(result).toEqual({ data: "initialized" });
    });
  });

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

    it("should not notify onChange when value is absent even if onChange is set", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        state.usePassthrough<number>({ initial: 3, onChange }),
      );
      expect(result.current[0]).toBe(3);
      act(() => result.current[1](8));
      expect(result.current[0]).toBe(8);
      expect(onChange).not.toHaveBeenCalled();
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
