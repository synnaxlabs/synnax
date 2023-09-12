// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import * as bounds from "@/spatial/bounds";

describe("Bounds", () => {
  describe("construction", () => {
    type T = [string, bounds.Crude];
    const TESTS: T[] = [
      ["from couple", [1, 2]],
      ["from bounds", { lower: 1, upper: 2 }],
      ["from invalid bounds", { upper: 1, lower: 2 }],
    ];
    TESTS.forEach(([name, arg]) => {
      test(name, () => {
        const bound = bounds.construct(arg);
        expect(bound.lower).toEqual(1);
        expect(bound.upper).toEqual(2);
      });
    });
    it("should consider a single argument as the upper bound", () => {
      const bound = bounds.construct(1);
      expect(bound.lower).toEqual(0);
      expect(bound.upper).toEqual(1);
    });
  });
  describe("contains", () => {
    it("should return true if the value is within the bounds", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.contains(b, 1.5)).toEqual(true);
    });
    it("should return false if the value is outside the bounds", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.contains(b, 2.5)).toEqual(false);
    });
    it("should return true if the value is equal to the lower bound", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.contains(b, 1)).toEqual(true);
    });
    it("should return false if the value is equal to the upper bound", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.contains(b, 2)).toEqual(false);
    });
  });
  describe("span", () => {
    it("should return the span of the bound", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.span(b)).toEqual(1);
    });
  });
  test("isZero", () => {
    const b = bounds.construct([0, 0]);
    expect(bounds.isZero(b)).toEqual(true);
  });
  test("spanIsZero", () => {
    const b = bounds.construct([1, 1]);
    expect(bounds.spanIsZero(b)).toEqual(true);
  });
  describe("max", () => {
    it("should return the bound with the maximum possible span", () => {
      const args: bounds.Crude[] = [
        [1, 2],
        [-1, 1],
      ];
      const bound = bounds.max(args);
      expect(bound.lower).toEqual(-1);
      expect(bound.upper).toEqual(2);
    });
  });
  describe("min", () => {
    it("should return the bound with the minimum possible span", () => {
      const args: bounds.Crude[] = [
        [1, 2],
        [-1, 1],
      ];
      const bound = bounds.min(args);
      expect(bound.lower).toEqual(1);
      expect(bound.upper).toEqual(1);
    });
  });
  describe("isFinite", () => {
    it("should return false if either bound is infinite", () => {
      const b = bounds.construct([1, Infinity]);
      expect(bounds.isFinite(b)).toEqual(false);
    });
    it("should return true if both bounds are finite", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.isFinite(b)).toEqual(true);
    });
  });
});
