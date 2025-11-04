// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { insert, search } from "@/compare/binary";

describe("compare", () => {
  describe("search", () => {
    describe("number arrays", () => {
      interface Spec {
        name: string;
        array: number[];
        value: number;
        expected: number;
      }
      const SPECS: Spec[] = [
        {
          name: "finds element at start",
          array: [1, 2, 3, 4, 5],
          value: 1,
          expected: 0,
        },
        {
          name: "finds element in middle",
          array: [1, 2, 3, 4, 5],
          value: 3,
          expected: 2,
        },
        {
          name: "finds element at end",
          array: [1, 2, 3, 4, 5],
          value: 5,
          expected: 4,
        },
        {
          name: "returns insertion point for missing element (start)",
          array: [2, 4, 6, 8],
          value: 1,
          expected: 0,
        },
        {
          name: "returns insertion point for missing element (middle)",
          array: [1, 3, 5, 7],
          value: 4,
          expected: 2,
        },
        {
          name: "returns insertion point for missing element (end)",
          array: [1, 2, 3, 4],
          value: 5,
          expected: 4,
        },
        {
          name: "handles empty array",
          array: [],
          value: 1,
          expected: 0,
        },
        {
          name: "handles single element array (found)",
          array: [5],
          value: 5,
          expected: 0,
        },
        {
          name: "handles single element array (not found, before)",
          array: [5],
          value: 3,
          expected: 0,
        },
        {
          name: "handles single element array (not found, after)",
          array: [5],
          value: 7,
          expected: 1,
        },
      ];
      SPECS.forEach((spec) => {
        it(spec.name, () => {
          expect(search(spec.array, spec.value)).toBe(spec.expected);
        });
      });
    });

    describe("string arrays", () => {
      interface Spec {
        name: string;
        array: string[];
        value: string;
        expected: number;
      }
      const SPECS: Spec[] = [
        {
          name: "finds string element",
          array: ["apple", "banana", "cherry", "date"],
          value: "cherry",
          expected: 2,
        },
        {
          name: "returns insertion point for missing string",
          array: ["apple", "banana", "date"],
          value: "cherry",
          expected: 2,
        },
        {
          name: "handles not found string",
          array: ["apple", "banana", "date"],
          value: "cherry",
          expected: 2,
        },
      ];
      SPECS.forEach((spec) => {
        it(spec.name, () => {
          expect(search(spec.array, spec.value)).toBe(spec.expected);
        });
      });
    });

    describe("custom comparator", () => {
      interface Item {
        id: number;
        name: string;
      }
      interface Spec {
        name: string;
        array: Item[];
        value: Item;
        expected: number;
      }
      const SPECS: Spec[] = [
        {
          name: "finds element with custom comparator",
          array: [
            { id: 1, name: "a" },
            { id: 2, name: "b" },
            { id: 3, name: "c" },
          ],
          value: { id: 2, name: "b" },
          expected: 1,
        },
        {
          name: "returns insertion point with custom comparator",
          array: [
            { id: 1, name: "a" },
            { id: 3, name: "c" },
            { id: 5, name: "e" },
          ],
          value: { id: 4, name: "d" },
          expected: 2,
        },
      ];
      SPECS.forEach((spec) => {
        it(spec.name, () => {
          const comparator = (a: Item, b: Item) => a.id - b.id;
          expect(search(spec.array, spec.value, comparator)).toBe(spec.expected);
        });
      });
    });
  });

  describe("insert", () => {
    describe("number arrays", () => {
      interface Spec {
        name: string;
        array: number[];
        value: number;
        expected: number[];
      }
      const SPECS: Spec[] = [
        {
          name: "inserts into empty array",
          array: [],
          value: 5,
          expected: [5],
        },
        {
          name: "inserts at beginning",
          array: [2, 3, 4],
          value: 1,
          expected: [1, 2, 3, 4],
        },
        {
          name: "inserts in middle",
          array: [1, 2, 4, 5],
          value: 3,
          expected: [1, 2, 3, 4, 5],
        },
        {
          name: "inserts at end",
          array: [1, 2, 3],
          value: 4,
          expected: [1, 2, 3, 4],
        },
        {
          name: "inserts duplicate value",
          array: [1, 2, 3, 4],
          value: 2,
          expected: [1, 2, 2, 3, 4],
        },
      ];
      SPECS.forEach((spec) => {
        it(spec.name, () => {
          const arr = [...spec.array];
          insert(arr, spec.value);
          expect(arr).toEqual(spec.expected);
        });
      });
    });

    describe("string arrays", () => {
      interface Spec {
        name: string;
        array: string[];
        value: string;
        expected: string[];
      }
      const SPECS: Spec[] = [
        {
          name: "inserts string in correct position",
          array: ["apple", "banana", "date"],
          value: "cherry",
          expected: ["apple", "banana", "cherry", "date"],
        },
        {
          name: "inserts at beginning of string array",
          array: ["banana", "cherry"],
          value: "apple",
          expected: ["apple", "banana", "cherry"],
        },
        {
          name: "handles not found string",
          array: ["apple", "banana", "date"],
          value: "cherry",
          expected: ["apple", "banana", "cherry", "date"],
        },
        {
          name: "handles empty array",
          array: [],
          value: "cherry",
          expected: ["cherry"],
        },
      ];
      SPECS.forEach((spec) => {
        it(spec.name, () => {
          const arr = [...spec.array];
          insert(arr, spec.value);
          expect(arr).toEqual(spec.expected);
        });
      });
    });

    describe("custom comparator", () => {
      interface Item {
        id: number;
        name: string;
      }
      interface Spec {
        name: string;
        array: Item[];
        value: Item;
        expected: Item[];
      }
      const SPECS: Spec[] = [
        {
          name: "inserts with custom comparator",
          array: [
            { id: 1, name: "a" },
            { id: 3, name: "c" },
          ],
          value: { id: 2, name: "b" },
          expected: [
            { id: 1, name: "a" },
            { id: 2, name: "b" },
            { id: 3, name: "c" },
          ],
        },
        {
          name: "maintains sort order with custom comparator",
          array: [
            { id: 5, name: "e" },
            { id: 10, name: "j" },
          ],
          value: { id: 7, name: "g" },
          expected: [
            { id: 5, name: "e" },
            { id: 7, name: "g" },
            { id: 10, name: "j" },
          ],
        },
      ];
      SPECS.forEach((spec) => {
        it(spec.name, () => {
          const arr = [...spec.array];
          const comparator = (a: Item, b: Item) => a.id - b.id;
          insert(arr, spec.value, comparator);
          expect(arr).toEqual(spec.expected);
        });
      });
    });
  });
});
