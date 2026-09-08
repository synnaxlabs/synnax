// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { notation } from "@/notation";

interface TestCase {
  number: number | bigint;
  precision: number;
  expected: Record<notation.Notation, string>;
}

const TEST_CASES: TestCase[] = [
  // === Basic positive numbers ===
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
  // === Negative zero — should render identically to positive zero ===
  {
    number: -0,
    precision: 0,
    expected: { standard: "0", scientific: "0ᴇ0", engineering: "0ᴇ0" },
  },
  {
    number: -0,
    precision: 2,
    expected: { standard: "0.00", scientific: "0.00ᴇ0", engineering: "0.00ᴇ0" },
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
  // === Special values (precision must be ignored) ===
  {
    number: NaN,
    precision: 0,
    expected: { standard: "NaN", scientific: "NaN", engineering: "NaN" },
  },
  {
    number: NaN,
    precision: 4,
    expected: { standard: "NaN", scientific: "NaN", engineering: "NaN" },
  },
  {
    number: Infinity,
    precision: 0,
    expected: { standard: "∞", scientific: "∞", engineering: "∞" },
  },
  {
    number: Infinity,
    precision: 4,
    expected: { standard: "∞", scientific: "∞", engineering: "∞" },
  },
  {
    number: -Infinity,
    precision: 0,
    expected: { standard: "-∞", scientific: "-∞", engineering: "-∞" },
  },
  {
    number: -Infinity,
    precision: 4,
    expected: { standard: "-∞", scientific: "-∞", engineering: "-∞" },
  },
  // === Small fractional numbers ===
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
  // === Negative small fractional numbers ===
  {
    number: -0.0001234,
    precision: 1,
    expected: { standard: "-0.0", scientific: "-1.2ᴇ-4", engineering: "-123.4ᴇ-6" },
  },
  {
    number: -0.0001234,
    precision: 0,
    expected: { standard: "-0", scientific: "-1ᴇ-4", engineering: "-123ᴇ-6" },
  },
  {
    number: 1,
    precision: 0,
    expected: { standard: "1", scientific: "1ᴇ0", engineering: "1ᴇ0" },
  },
  {
    number: 1,
    precision: 2,
    expected: { standard: "1.00", scientific: "1.00ᴇ0", engineering: "1.00ᴇ0" },
  },
  // === Power-of-10 boundaries where engineering and scientific differ ===
  {
    number: 10,
    precision: 0,
    expected: { standard: "10", scientific: "1ᴇ1", engineering: "10ᴇ0" },
  },
  {
    number: 100,
    precision: 0,
    expected: { standard: "100", scientific: "1ᴇ2", engineering: "100ᴇ0" },
  },
  {
    number: 1000,
    precision: 0,
    expected: { standard: "1000", scientific: "1ᴇ3", engineering: "1ᴇ3" },
  },
  {
    number: 10000,
    precision: 0,
    expected: { standard: "10000", scientific: "1ᴇ4", engineering: "10ᴇ3" },
  },
  {
    number: 100000,
    precision: 0,
    expected: { standard: "100000", scientific: "1ᴇ5", engineering: "100ᴇ3" },
  },
  {
    number: 1000000,
    precision: 0,
    expected: { standard: "1000000", scientific: "1ᴇ6", engineering: "1ᴇ6" },
  },
  // === Negative power-of-10 boundaries (fractions) ===
  {
    number: 0.1,
    precision: 1,
    expected: { standard: "0.1", scientific: "1.0ᴇ-1", engineering: "100.0ᴇ-3" },
  },
  {
    number: 0.01,
    precision: 1,
    expected: { standard: "0.0", scientific: "1.0ᴇ-2", engineering: "10.0ᴇ-3" },
  },
  {
    number: 0.001,
    precision: 1,
    expected: { standard: "0.0", scientific: "1.0ᴇ-3", engineering: "1.0ᴇ-3" },
  },
  // === Negative power-of-10 boundaries ===
  {
    number: -10,
    precision: 0,
    expected: { standard: "-10", scientific: "-1ᴇ1", engineering: "-10ᴇ0" },
  },
  {
    number: -100,
    precision: 0,
    expected: { standard: "-100", scientific: "-1ᴇ2", engineering: "-100ᴇ0" },
  },
  // === Power of 10 with non-zero precision ===
  {
    number: 100,
    precision: 2,
    expected: { standard: "100.00", scientific: "1.00ᴇ2", engineering: "100.00ᴇ0" },
  },
  {
    number: 1000,
    precision: 3,
    expected: { standard: "1000.000", scientific: "1.000ᴇ3", engineering: "1.000ᴇ3" },
  },
  // === Rounding overflow — toFixed pushes the mantissa past the canonical upper
  //     bound (>= 10 for scientific, >= 1000 for engineering). The exponent must be
  //     bumped so the mantissa stays in the canonical range. ===
  {
    number: 9.999,
    precision: 1,
    expected: { standard: "10.0", scientific: "1.0ᴇ1", engineering: "10.0ᴇ0" },
  },
  {
    number: 9.999,
    precision: 0,
    expected: { standard: "10", scientific: "1ᴇ1", engineering: "10ᴇ0" },
  },
  {
    number: 99.99,
    precision: 0,
    expected: { standard: "100", scientific: "1ᴇ2", engineering: "100ᴇ0" },
  },
  {
    number: 999.99,
    precision: 0,
    expected: { standard: "1000", scientific: "1ᴇ3", engineering: "1ᴇ3" },
  },
  {
    number: 9999.99,
    precision: 0,
    expected: { standard: "10000", scientific: "1ᴇ4", engineering: "10ᴇ3" },
  },
  {
    number: -9.999,
    precision: 1,
    expected: { standard: "-10.0", scientific: "-1.0ᴇ1", engineering: "-10.0ᴇ0" },
  },
  {
    number: -999.99,
    precision: 0,
    expected: { standard: "-1000", scientific: "-1ᴇ3", engineering: "-1ᴇ3" },
  },
  {
    number: 1n,
    precision: 0,
    expected: { standard: "1", scientific: "1ᴇ0", engineering: "1ᴇ0" },
  },
  {
    number: 1n,
    precision: 2,
    expected: { standard: "1.00", scientific: "1.00ᴇ0", engineering: "1.00ᴇ0" },
  },
  {
    number: 1778020940471336960n,
    precision: 0,
    expected: {
      standard: "1778020940471336960",
      scientific: "2ᴇ18",
      engineering: "2ᴇ18",
    },
  },
  {
    number: 1778020940471336960n,
    precision: 11,
    expected: {
      standard: "1778020940471336960.00000000000",
      scientific: "1.77802094047ᴇ18",
      engineering: "1.77802094047ᴇ18",
    },
  },
  {
    number: -1n,
    precision: 0,
    expected: { standard: "-1", scientific: "-1ᴇ0", engineering: "-1ᴇ0" },
  },
  {
    number: -1778020940471336960n,
    precision: 0,
    expected: {
      standard: "-1778020940471336960",
      scientific: "-2ᴇ18",
      engineering: "-2ᴇ18",
    },
  },
  {
    number: -1778020940471336960n,
    precision: 11,
    expected: {
      standard: "-1778020940471336960.00000000000",
      scientific: "-1.77802094047ᴇ18",
      engineering: "-1.77802094047ᴇ18",
    },
  },
  {
    number: 0n,
    precision: 0,
    expected: { standard: "0", scientific: "0ᴇ0", engineering: "0ᴇ0" },
  },
  {
    number: 0n,
    precision: 11,
    expected: {
      standard: "0.00000000000",
      scientific: "0.00000000000ᴇ0",
      engineering: "0.00000000000ᴇ0",
    },
  },
  // === Large-magnitude floats expressed with e-notation ===
  // For values >= 10^21, JS's toFixed delegates to ToString, so standard output
  // here is itself in e-notation. The scientific and engineering branches keep
  // their canonical forms.
  {
    number: 9e124,
    precision: 0,
    expected: { standard: "9e+124", scientific: "9ᴇ124", engineering: "90ᴇ123" },
  },
  {
    number: 9.99e124,
    precision: 0,
    expected: { standard: "9.99e+124", scientific: "1ᴇ125", engineering: "100ᴇ123" },
  },
  {
    number: -3e10,
    precision: 0,
    expected: { standard: "-30000000000", scientific: "-3ᴇ10", engineering: "-30ᴇ9" },
  },
  // === Tiny-magnitude floats expressed with e-notation ===
  {
    number: 1e-200,
    precision: 0,
    expected: { standard: "0", scientific: "1ᴇ-200", engineering: "10ᴇ-201" },
  },
  {
    number: 5e-50,
    precision: 0,
    expected: { standard: "0", scientific: "5ᴇ-50", engineering: "50ᴇ-51" },
  },
];

describe("stringifyNumber", () => {
  TEST_CASES.forEach(({ number, precision, expected }) =>
    describe(`${typeof number === "bigint" ? "bigint " : ""}number: ${number}, precision: ${precision}`, () =>
      notation.NOTATIONS.forEach((n) =>
        it(`should format correctly in ${n} notation`, () => {
          const result = notation.stringifyNumber(number, precision, n);
          expect(result).toBe(expected[n]);
        }),
      )),
  );
});
