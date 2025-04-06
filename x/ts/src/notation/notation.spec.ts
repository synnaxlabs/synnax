// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { notation } from "@/notation";
import { type bounds } from "@/spatial";

interface TestCase {
  number: number;
  precision: number;
  expected: Record<notation.Notation, string>;
}

const TEST_CASES: TestCase[] = [
  {
    number: 12345.678,
    precision: 1,
    expected: { standard: "12345.7", scientific: "1.2ᴇ4", engineering: "12.3ᴇ3" },
  },
  {
    number: 12345.678,
    precision: 0,
    expected: { standard: "12346", scientific: "1ᴇ4", engineering: "12ᴇ3" },
  },
  {
    number: 0,
    precision: 1,
    expected: { standard: "0.0", scientific: "0.0ᴇ0", engineering: "0.0ᴇ0" },
  },
  {
    number: 0,
    precision: 0,
    expected: { standard: "0", scientific: "0ᴇ0", engineering: "0ᴇ0" },
  },
  {
    number: -1234.5678,
    precision: 1,
    expected: { standard: "-1234.6", scientific: "-1.2ᴇ3", engineering: "-1.2ᴇ3" },
  },
  {
    number: -1234.5678,
    precision: 0,
    expected: { standard: "-1235", scientific: "-1ᴇ3", engineering: "-1ᴇ3" },
  },
  {
    number: NaN,
    precision: 0,
    expected: { standard: "NaN", scientific: "NaN", engineering: "NaN" },
  },
  {
    number: Infinity,
    precision: 0,
    expected: { standard: "∞", scientific: "∞", engineering: "∞" },
  },
  {
    number: -Infinity,
    precision: 0,
    expected: { standard: "-∞", scientific: "-∞", engineering: "-∞" },
  },
  {
    number: 0.0001234,
    precision: 1,
    expected: { standard: "0.0", scientific: "1.2ᴇ-4", engineering: "123.4ᴇ-6" },
  },
  {
    number: 0.0001234,
    precision: 0,
    expected: { standard: "0", scientific: "1ᴇ-4", engineering: "123ᴇ-6" },
  },
];

describe("stringifyNumber", () => {
  TEST_CASES.forEach(({ number, precision, expected }) =>
    describe(`number: ${number}, precision: ${precision}`, () =>
      notation.NOTATIONS.forEach((n) =>
        it(`should format correctly in ${n} notation`, () => {
          const result = notation.stringifyNumber(number, precision, n);
          expect(result).toBe(expected[n]);
        }),
      )),
  );
});

describe("roundSmart", () => {
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

  TEST_CASES.forEach(({ name, value, bounds, expected }) => {
    it(name, () => {
      const result = notation.roundSmart(value, bounds);
      if (Number.isNaN(expected)) expect(Number.isNaN(result)).toBe(true);
      else expect(result).toBe(expected);
    });
  });
});
