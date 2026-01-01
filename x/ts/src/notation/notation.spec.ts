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
