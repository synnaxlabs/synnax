// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Notation, NOTATIONS, stringifyNumber } from "@/notation/notation";

interface TestCase {
  number: number;
  precision: number;
  expected: Record<Notation, string>;
}

const TEST_CASES: TestCase[] = [
  {
    number: 1234.5678,
    precision: 2,
    expected: { standard: "1234.57", scientific: "1.23e+3", engineering: "1.23e3" },
  },
  {
    number: 0,
    precision: 2,
    expected: { standard: "0.00", scientific: "0.00e+0", engineering: "0.00e0" },
  },
  {
    number: -1234.5678,
    precision: 2,
    expected: { standard: "-1234.57", scientific: "-1.23e+3", engineering: "-1.23e3" },
  },
  {
    number: NaN,
    precision: 2,
    expected: { standard: "NaN", scientific: "NaN", engineering: "NaN" },
  },
  {
    number: Infinity,
    precision: 2,
    expected: { standard: "∞", scientific: "∞", engineering: "∞" },
  },
  {
    number: -Infinity,
    precision: 2,
    expected: { standard: "-∞", scientific: "-∞", engineering: "-∞" },
  },
  {
    number: 0.0001234,
    precision: 2,
    expected: { standard: "0.00", scientific: "1.23e-4", engineering: "123.40e-6" },
  },
];

describe("stringifyNumber", () => {
  TEST_CASES.forEach(({ number, precision, expected }) =>
    describe(`number: ${number}, precision: ${precision}`, () =>
      NOTATIONS.forEach((notation) =>
        it(`should format correctly in ${notation} notation`, () => {
          const result = stringifyNumber(number, precision, notation);
          expect(result).toBe(expected[notation]);
        }),
      )),
  );
});
