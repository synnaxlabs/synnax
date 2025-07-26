// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
      if (Number.isNaN(expected)) expect(Number.isNaN(result)).toBeTruthy();
      else expect(result).toBe(expected);
    });
  });
});
