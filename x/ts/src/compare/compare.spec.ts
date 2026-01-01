// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { compare } from "@/compare";

describe("compare", () => {
  describe("numericStrings", () => {
    interface Spec {
      name: string;
      input: string[];
      expected: string[];
    }
    const SPECS: Spec[] = [
      {
        name: "pure numbers",
        input: ["5", "4", "3", "2", "1"],
        expected: ["1", "2", "3", "4", "5"],
      },
      {
        name: "suffixed numbers with equal prefixes",
        input: ["a5", "a4", "a3", "a2", "a1"],
        expected: ["a1", "a2", "a3", "a4", "a5"],
      },
      {
        name: "suffixed numbers with different prefixes",
        input: ["a1", "b1", "a2", "b2", "a3", "b3"],
        expected: ["a1", "a2", "a3", "b1", "b2", "b3"],
      },
      {
        name: "mixed separators",
        input: ["a2", "a.1", "a-3", "a_4"],
        expected: ["a.1", "a2", "a-3", "a_4"],
      },
    ];
    SPECS.forEach((spec) => {
      it(spec.name, () => {
        expect(spec.input.sort(compare.stringsWithNumbers)).toEqual(spec.expected);
      });
    });
  });
});
