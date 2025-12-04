// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  executeInitialSetter,
  executeSetter,
  isInitialSetter,
  isSetter,
  skipNull,
  skipUndefined,
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
