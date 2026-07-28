// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

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
});
