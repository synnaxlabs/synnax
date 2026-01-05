// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* eslint-disable no-loss-of-precision */

import { describe, expect, it } from "vitest";

import { math } from "@/math";
import { type bounds } from "@/spatial";

interface TestCase {
  name: string;
  value: number;
  bounds: bounds.Bounds<number>;
  expected: number;
}

const TEST_CASES: TestCase[] = [
  {
    name: "should handle large spans (>= 1000)",
    value: 1234.5678,
    bounds: { lower: 0, upper: 2000 },
    expected: 1234.57,
  },
  {
    name: "should handle medium spans (>= 1)",
    value: 1.23456,
    bounds: { lower: 0, upper: 2 },
    expected: 1.235,
  },
  {
    name: "should handle small spans (< 1)",
    value: 0.123456,
    bounds: { lower: 0, upper: 0.2 },
    expected: 0.123,
  },
  {
    name: "should handle very small spans",
    value: 0.0001234,
    bounds: { lower: 0, upper: 0.0002 },
    expected: 0.000123,
  },
  {
    name: "should handle negative values",
    value: -1.23456,
    bounds: { lower: -2, upper: 0 },
    expected: -1.235,
  },
  {
    name: "should handle NaN",
    value: NaN,
    bounds: { lower: 0, upper: 1 },
    expected: NaN,
  },
  {
    name: "should handle Infinity",
    value: Infinity,
    bounds: { lower: 0, upper: 1 },
    expected: Infinity,
  },
  {
    name: "should handle zero span bounds",
    value: 1.23456,
    bounds: { lower: 1, upper: 1 },
    expected: 1.23456,
  },
];

describe("roundBySpan", () => {
  TEST_CASES.forEach(({ name, value, bounds, expected }) => {
    it(name, () => {
      const result = math.roundBySpan(value, bounds);
      if (Number.isNaN(expected)) expect(Number.isNaN(result)).toBe(true);
      else expect(result).toBe(expected);
    });
  });
});

interface SmartRoundTestCase {
  value: number;
  bounds?: bounds.Bounds<number>;
  expected: number;
}

describe("smartRound", () => {
  describe("span-based rounding (significant span)", () => {
    const cases: SmartRoundTestCase[] = [
      { value: 1234.5678, bounds: { lower: 0, upper: 2000 }, expected: 1234.57 },
      { value: -5432.987654, bounds: { lower: -6000, upper: 0 }, expected: -5432.99 },
      { value: 1.23456, bounds: { lower: 0, upper: 2 }, expected: 1.235 },
      { value: -15.6789, bounds: { lower: -20, upper: -10 }, expected: -15.679 },
      { value: 0.123456, bounds: { lower: 0, upper: 0.2 }, expected: 0.123 },
      { value: 0.0001234567, bounds: { lower: 0, upper: 0.001 }, expected: 0.00012 },
      {
        value: 0.00000123456789,
        bounds: { lower: 0, upper: 0.00001 },
        expected: 0.0000012,
      },
      { value: 999.9999, bounds: { lower: 0, upper: 1500 }, expected: 1000.0 },
      { value: 0.99999, bounds: { lower: 0, upper: 1.5 }, expected: 1.0 },
    ];
    cases.forEach(({ value, bounds, expected }) => {
      it(`${value} with span ${(bounds!.upper - bounds!.lower).toFixed(1)}`, () => {
        expect(math.smartRound(value, bounds)).toBe(expected);
      });
    });
  });

  describe("zero or negligible span", () => {
    const cases: SmartRoundTestCase[] = [
      {
        value: 71.60000610351562,
        bounds: { lower: 71.6, upper: 71.6 },
        expected: 71.6,
      },
      {
        value: 1234567.89123456,
        bounds: { lower: 1234567, upper: 1234567 },
        expected: 1234567.89,
      },
      {
        value: -9876543.21987654,
        bounds: { lower: -9876543, upper: -9876543 },
        expected: -9876543.22,
      },
      { value: 123.456789123, bounds: { lower: 123, upper: 123 }, expected: 123.457 },
      {
        value: -456.789123456,
        bounds: { lower: -456, upper: -456 },
        expected: -456.789,
      },
      {
        value: 0.0001234567,
        bounds: { lower: 0.00012, upper: 0.00012 },
        expected: 0.00012346,
      },
      {
        value: 0.00000987654321,
        bounds: { lower: 0.0000099, upper: 0.0000099 },
        expected: 0.0000098765,
      },
      { value: 1.00000123456, bounds: { lower: 1, upper: 1 }, expected: 1.0 },
      {
        value: 0.999999123456,
        bounds: { lower: 0.999999, upper: 0.999999 },
        expected: 1.0,
      },
      {
        value: 1000.123456789,
        bounds: { lower: 1000, upper: 1000 },
        expected: 1000.12,
      },
      {
        value: 1000000.5,
        bounds: { lower: 1000000.499999, upper: 1000000.500001 },
        expected: 1000000.5,
      },
    ];
    cases.forEach(({ value, bounds, expected }) => {
      it(`${value}`, () => expect(math.smartRound(value, bounds)).toBe(expected));
    });
  });

  describe("no bounds", () => {
    const cases: SmartRoundTestCase[] = [
      { value: 123456.789123, expected: 123456.79 },
      { value: 71.60000610351562, expected: 71.6 },
      { value: 0.00012345678, expected: 0.00012346 },
      { value: 9876543210.123456, expected: 9876543210.12 },
      { value: 0.00000000123456789, expected: 0.0000000012346 },
      { value: -987654.321987, expected: -987654.32 },
      { value: -42.123456789, expected: -42.123 },
      { value: -0.000456789123, expected: -0.00045679 },
      { value: 999.9999, expected: 1000.0 },
      { value: 0.9999999, expected: 1.0 },
    ];
    cases.forEach(({ value, expected }) => {
      it(`${value}`, () => expect(math.smartRound(value)).toBe(expected));
    });
  });

  describe("edge cases", () => {
    it("NaN", () => expect(Number.isNaN(math.smartRound(NaN))).toBe(true));
    it("NaN with bounds", () =>
      expect(Number.isNaN(math.smartRound(NaN, { lower: 0, upper: 10 }))).toBe(true));
    it("Infinity", () => expect(math.smartRound(Infinity)).toBe(Infinity));
    it("-Infinity", () => expect(math.smartRound(-Infinity)).toBe(-Infinity));
    it("Infinity with bounds", () =>
      expect(math.smartRound(Infinity, { lower: 0, upper: 100 })).toBe(Infinity));
    it("zero", () => expect(math.smartRound(0)).toBe(0));
    it("zero with bounds", () =>
      expect(math.smartRound(0, { lower: -10, upper: 10 })).toBe(0));
    it("very small value", () => expect(math.smartRound(1e-15)).toBeCloseTo(1e-15, 17));
    it("very large value", () => expect(math.smartRound(1e15)).toBe(1e15));
    it("negative zero", () => expect(math.smartRound(-0)).toBe(0));
  });
  describe("boundary conditions", () => {
    it("zero span", () =>
      expect(math.smartRound(123.456789, { lower: 100, upper: 100 })).toBe(123.457));
    it("span above threshold", () =>
      expect(math.smartRound(1000, { lower: 999, upper: 1000 })).toBe(1000.0));
    it("span below threshold", () =>
      expect(math.smartRound(1000000, { lower: 1000000, upper: 1000000 + 1e-12 })).toBe(
        1000000.0,
      ));
    it("1000 boundary", () => {
      expect(math.smartRound(999.999)).toBe(999.999);
      expect(math.smartRound(1000.001)).toBe(1000.0);
    });
    it("1 boundary", () => {
      expect(math.smartRound(0.999999)).toBe(1.0);
      expect(math.smartRound(1.000001)).toBe(1.0);
    });
  });
  describe("precision preservation", () => {
    it("no artificial precision", () =>
      expect(math.smartRound(1.5, { lower: 1, upper: 2 })).toBe(1.5));
    it("0.1 + 0.2", () => expect(math.smartRound(0.1 + 0.2)).toBe(0.3));
    it("0.1 * 3", () => expect(math.smartRound(0.1 * 3)).toBe(0.3));
    it("integer preservation", () =>
      expect(math.smartRound(42, { lower: 0, upper: 100 })).toBe(42.0));
    it("rounds to integer", () =>
      expect(math.smartRound(42.0001, { lower: 42, upper: 42 })).toBe(42.0));
  });
  describe("stress tests", () => {
    it("random values", () => {
      for (let i = 0; i < 100; i++) {
        const value = Math.random() * 1000000 - 500000;
        expect(Number.isFinite(math.smartRound(value))).toBe(true);
      }
    });
    it("random with bounds", () => {
      for (let i = 0; i < 100; i++) {
        const lower = Math.random() * 1000 - 500;
        const upper = lower + Math.random() * 1000;
        const value = lower + (upper - lower) * Math.random();
        const result = math.smartRound(value, { lower, upper });
        expect(Number.isFinite(result)).toBe(true);
        expect(result).toBeGreaterThanOrEqual(lower - 1);
        expect(result).toBeLessThanOrEqual(upper + 1);
      }
    });
  });
});
