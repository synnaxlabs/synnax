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
  describe("resolve", () => {
    it("should return the value directly if not a function", () => {
      expect(state.resolve(42, 0)).toBe(42);
      expect(state.resolve("hello", "")).toBe("hello");
      expect(state.resolve({ key: "value" }, {})).toEqual({ key: "value" });
    });

    it("should call the function with prev value if a function", () => {
      const setter = (prev: number) => prev + 10;
      expect(state.resolve(setter, 5)).toBe(15);
    });
  });

  describe("resolveInitial", () => {
    it("should return the value directly if not a function", () => {
      expect(state.resolveInitial(42)).toBe(42);
      expect(state.resolveInitial("hello")).toBe("hello");
    });

    it("should call the function only when resolved", () => {
      let called = false;
      const initializer = () => {
        called = true;
        return { data: "initialized" };
      };
      expect(called).toBe(false);
      const result = state.resolveInitial(initializer);
      expect(called).toBe(true);
      expect(result).toEqual({ data: "initialized" });
    });
  });
});
