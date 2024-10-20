// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { math } from "@/math";

describe("math", () => {
  describe("sub", () => {
    it("should subtract two numbers", () => {
      expect(math.sub(1, 2)).toBe(-1);
    });
    it("should subtract two bigints", () => {
      expect(math.sub(1n, 2n)).toBe(-1n);
    });
    it("should subtract a number and a bigint", () => {
      expect(math.sub(1, 2n)).toBe(-1);
    });
    it("should subtract a bigint and a number", () => {
      expect(math.sub(1n, 2)).toBe(-1n);
    });
  });
  describe("add", () => {
    it("should add two numbers", () => {
      expect(math.add(1, 2)).toBe(3);
    });
    it("should add two bigints", () => {
      expect(math.add(1n, 2n)).toBe(3n);
    });
    it("should add a number and a bigint", () => {
      expect(math.add(1, 2n)).toBe(3);
    });
    it("should add a bigint and a number", () => {
      expect(math.add(1n, 2)).toBe(3n);
    });
  });
  describe("closeTo", () => {
    it("should return true if two numbers are close", () => {
      expect(math.closeTo(1, 1.0001)).toBe(true);
    });
    it("should return false if two numbers are not close", () => {
      expect(math.closeTo(1, 1.1)).toBe(false);
    });
  });
  describe("equal", () => {
    it("should return true if two numbers are equal", () => {
      expect(math.equal(1, 1)).toBe(true);
    });
    it("should return false if two numbers are not equal", () => {
      expect(math.equal(1, 2)).toBe(false);
    });
    it("should return true if two bigints are equal", () => {
      expect(math.equal(1n, 1n)).toBe(true);
    });
    it("should return false if two bigints are not equal", () => {
      expect(math.equal(1n, 2n)).toBe(false);
    });
    it("should return true if a number and a bigint are equal", () => {
      expect(math.equal(1, 1n)).toBe(true);
    });
    it("should return false if a number and a bigint are not equal", () => {
      expect(math.equal(1, 2n)).toBe(false);
    });
    it("should return true if a bigint and a number are equal", () => {
      expect(math.equal(1n, 1)).toBe(true);
    });
    it("should return false if a bigint and a number are not equal", () => {
      expect(math.equal(1n, 2)).toBe(false);
    });
  });
  describe("roundToNearestMagnitude", () => {
    it("should round to the nearest magnitude of 10", () => {
      expect(math.roundToNearestMagnitude(1234)).toBe(1000);
      expect(math.roundToNearestMagnitude(12345)).toBe(10000);
      expect(math.roundToNearestMagnitude(123456)).toBe(100000);
    });
  });
  describe("min", () => {
    it("should return the minimum of two numbers", () => {
      expect(math.min(1, 2)).toBe(1);
    });
    it("should return the minimum of two bigints", () => {
      expect(math.min(1n, 2n)).toBe(1n);
    });
    it("should return the minimum of a number and a bigint", () => {
      expect(math.min(1, 2n)).toBe(1);
    });
    it("should return the minimum of a bigint and a number", () => {
      expect(math.min(1n, 2)).toBe(1n);
    });
  });
  describe("max", () => {
    it("should return the maximum of two numbers", () => {
      expect(math.max(1, 2)).toBe(2);
    });
    it("should return the maximum of two bigints", () => {
      expect(math.max(1n, 2n)).toBe(2n);
    });
    it("should return the maximum of a number and a bigint", () => {
      expect(math.max(1, 2n)).toBe(2);
    });
    it("should return the maximum of a bigint and a number", () => {
      expect(math.max(1n, 2)).toBe(2n);
    });
  });
  describe("abs", () => {
    it("should return the absolute value of a number", () => {
      expect(math.abs(-1)).toBe(1);
    });
    it("should return the absolute value of a bigint", () => {
      expect(math.abs(-1n)).toBe(1n);
    });
  });
});
