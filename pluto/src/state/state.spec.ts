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

import {
  executeInitialSetter,
  executeSetter,
  isInitialSetter,
  isSetter,
  skipNull,
  skipUndefined,
  usePassthrough,
  usePurePassthrough,
} from "@/state/state";

describe("state", () => {
  describe("isSetter", () => {
    it("should return true for a function", () => {
      const setter = (prev: number) => prev + 1;
      expect(isSetter(setter)).toBe(true);
    });

    it("should return false for a value", () => {
      expect(isSetter(42)).toBe(false);
      expect(isSetter("hello")).toBe(false);
      expect(isSetter({ key: "value" })).toBe(false);
    });
  });

  describe("executeSetter", () => {
    it("should return the value directly if not a function", () => {
      expect(executeSetter(42, 0)).toBe(42);
      expect(executeSetter("hello", "")).toBe("hello");
    });

    it("should call the function with prev value if a function", () => {
      const setter = (prev: number) => prev + 10;
      expect(executeSetter(setter, 5)).toBe(15);
    });
  });

  describe("skipUndefined", () => {
    it("should return undefined if the input is undefined", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = skipUndefined(setter);
      expect(wrapped(undefined)).toBe(undefined);
    });

    it("should return undefined if the input is null", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = skipUndefined(setter);
      expect(wrapped(null as unknown as undefined)).toBe(undefined);
    });

    it("should apply the function if the input is defined", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = skipUndefined(setter);
      expect(wrapped(5)).toBe(6);
    });

    it("should work with object state", () => {
      const setter = (prev: { count: number }) => ({ count: prev.count + 1 });
      const wrapped = skipUndefined(setter);
      expect(wrapped({ count: 10 })).toEqual({ count: 11 });
      expect(wrapped(undefined)).toBe(undefined);
    });
  });

  describe("skipNull", () => {
    it("should return null if the input is null", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = skipNull(setter);
      expect(wrapped(null)).toBe(null);
    });

    it("should return null if the input is undefined", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = skipNull(setter);
      expect(wrapped(undefined as unknown as null)).toBe(null);
    });

    it("should apply the function if the input is not null", () => {
      const setter = (prev: number) => prev + 1;
      const wrapped = skipNull(setter);
      expect(wrapped(5)).toBe(6);
    });

    it("should work with object state", () => {
      const setter = (prev: { name: string }) => ({ name: prev.name.toUpperCase() });
      const wrapped = skipNull(setter);
      expect(wrapped({ name: "test" })).toEqual({ name: "TEST" });
      expect(wrapped(null)).toBe(null);
    });
  });

  describe("isInitialSetter", () => {
    it("should return true for a function", () => {
      const initializer = () => 42;
      expect(isInitialSetter(initializer)).toBe(true);
    });

    it("should return false for a value", () => {
      expect(isInitialSetter(42)).toBe(false);
      expect(isInitialSetter("hello")).toBe(false);
    });
  });

  describe("usePassthrough", () => {
    it("should return the controlled value and onChange when both are given", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        usePassthrough<number>({ initial: 0, value: 5, onChange }),
      );
      const [value, set] = result.current;
      expect(value).toBe(5);
      act(() => set(6));
      expect(onChange).toHaveBeenCalledWith(6);
      const [after] = result.current;
      expect(after).toBe(5);
    });

    it("should update internal state and notify onChange when uncontrolled", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        usePassthrough<number>({ initial: 0, onChange }),
      );
      act(() => result.current[1](3));
      expect(result.current[0]).toBe(3);
      expect(onChange).toHaveBeenCalledTimes(1);
      expect(onChange).toHaveBeenCalledWith(3);
    });

    it("should forward function updaters to onChange when uncontrolled", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        usePassthrough<number>({ initial: 1, onChange }),
      );
      const updater = (prev: number) => prev + 1;
      act(() => result.current[1](updater));
      expect(result.current[0]).toBe(2);
      expect(onChange).toHaveBeenCalledWith(updater);
    });

    it("should keep a stable setter identity across state updates", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        usePassthrough<number>({ initial: 0, onChange }),
      );
      const first = result.current[1];
      act(() => result.current[1](1));
      expect(result.current[1]).toBe(first);
    });

    it("should work without onChange", () => {
      const { result } = renderHook(() =>
        usePassthrough<number>({ initial: 0, onChange: undefined }),
      );
      act(() => result.current[1](2));
      expect(result.current[0]).toBe(2);
    });
  });

  describe("usePurePassthrough", () => {
    it("should return the controlled value and onChange when both are given", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        usePurePassthrough<string>({ initialValue: "", value: "a", onChange }),
      );
      expect(result.current[0]).toBe("a");
      act(() => result.current[1]("b"));
      expect(onChange).toHaveBeenCalledWith("b");
      expect(result.current[0]).toBe("a");
    });

    it("should update internal state and notify onChange when uncontrolled", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        usePurePassthrough<string>({ initialValue: "", onChange }),
      );
      act(() => result.current[1]("b"));
      expect(result.current[0]).toBe("b");
      expect(onChange).toHaveBeenCalledTimes(1);
      expect(onChange).toHaveBeenCalledWith("b");
    });

    it("should keep a stable setter identity across state updates", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        usePurePassthrough<string>({ initialValue: "", onChange }),
      );
      const first = result.current[1];
      act(() => result.current[1]("b"));
      expect(result.current[1]).toBe(first);
    });
  });

  describe("executeInitialSetter", () => {
    it("should return the value directly if not a function", () => {
      expect(executeInitialSetter(42)).toBe(42);
      expect(executeInitialSetter("hello")).toBe("hello");
    });

    it("should call the function if a function", () => {
      const initializer = () => 42;
      expect(executeInitialSetter(initializer)).toBe(42);
    });

    it("should work with lazy initialization", () => {
      let called = false;
      const initializer = () => {
        called = true;
        return { data: "initialized" };
      };
      expect(called).toBe(false);
      const result = executeInitialSetter(initializer);
      expect(called).toBe(true);
      expect(result).toEqual({ data: "initialized" });
    });
  });
});
