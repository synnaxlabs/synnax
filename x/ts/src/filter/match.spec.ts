// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { filter } from "@/filter";
import { TimeStamp } from "@/telem";

describe("filter", () => {
  describe("matchesSet", () => {
    it("should match every value when no member is set", () => {
      expect(filter.matchesSet({}, "a")).toBe(true);
    });

    it("should match a value equal to eq", () => {
      expect(filter.matchesSet({ eq: "a" }, "a")).toBe(true);
      expect(filter.matchesSet({ eq: "a" }, "b")).toBe(false);
    });

    it("should match a value in in", () => {
      expect(filter.matchesSet({ in: ["a", "b"] }, "b")).toBe(true);
      expect(filter.matchesSet({ in: ["a", "b"] }, "c")).toBe(false);
    });

    it("should match no value with an empty in", () => {
      expect(filter.matchesSet({ in: [] }, "a")).toBe(false);
    });

    it("should AND the members that are set", () => {
      expect(filter.matchesSet({ eq: "a", in: ["b"] }, "a")).toBe(false);
    });

    it("should match false against eq", () => {
      expect(filter.matchesSet({ eq: false }, false)).toBe(true);
      expect(filter.matchesSet({ eq: false }, true)).toBe(false);
    });

    it("should compare value extensions by value", () => {
      expect(filter.matchesSet({ eq: new TimeStamp(5) }, new TimeStamp(5))).toBe(true);
    });
  });

  describe("matchesString", () => {
    it("should match eq and in", () => {
      expect(filter.matchesString({ eq: "a", in: ["a"] }, "a")).toBe(true);
    });

    it("should match contains and startsWith", () => {
      expect(filter.matchesString({ contains: "an" }, "Tank")).toBe(true);
      expect(filter.matchesString({ contains: "x" }, "Tank")).toBe(false);
      expect(filter.matchesString({ startsWith: "Ta" }, "Tank")).toBe(true);
      expect(filter.matchesString({ startsWith: "an" }, "Tank")).toBe(false);
    });

    it("should compare case by default", () => {
      expect(filter.matchesString({ eq: "tank" }, "Tank")).toBe(false);
      expect(filter.matchesString({ contains: "TA" }, "Tank")).toBe(false);
    });

    it("should ignore case with caseInsensitive", () => {
      const ci = { caseInsensitive: true };
      expect(filter.matchesString({ eq: "TANK", ...ci }, "tank")).toBe(true);
      expect(filter.matchesString({ in: ["TANK"], ...ci }, "tank")).toBe(true);
      expect(filter.matchesString({ contains: "AN", ...ci }, "tank")).toBe(true);
      expect(filter.matchesString({ startsWith: "TA", ...ci }, "tank")).toBe(true);
      expect(filter.matchesString({ eq: "ÄRGER", ...ci }, "ärger")).toBe(true);
    });

    it.each([
      ["a Greek final sigma", "οδοσ", "ΟΔΟΣ", true],
      ["a final and a medial sigma", "σ", "ς", true],
      ["a long s", "s", "ſ", true],
      ["the Kelvin sign", "k", "\u212a", true],
      ["a Turkish dotted capital I", "i", "İ", false],
    ])(
      "should fold case by Unicode simple case folding: %s",
      (_, eq, value, expected) => {
        expect(filter.matchesString({ eq, caseInsensitive: true }, value)).toBe(
          expected,
        );
      },
    );

    it("should treat regex characters in an operand as plain text", () => {
      const ci = { caseInsensitive: true };
      expect(filter.matchesString({ eq: "a.b", ...ci }, "axb")).toBe(false);
      expect(filter.matchesString({ contains: "(x)", ...ci }, "a(x)b")).toBe(true);
      expect(filter.matchesString({ startsWith: "[a-z]", ...ci }, "[a-z]1")).toBe(true);
      expect(filter.matchesString({ in: ["a\\b"], ...ci }, "a\\b")).toBe(true);
    });

    it("should throw on a regex", () => {
      expect(() => filter.matchesString({ regex: "a" }, "a")).toThrow(
        "cannot match a regex locally",
      );
    });
  });

  describe("matchesOrdered", () => {
    it("should match each bound", () => {
      expect(filter.matchesOrdered({ gt: 1 }, 2)).toBe(true);
      expect(filter.matchesOrdered({ gt: 2 }, 2)).toBe(false);
      expect(filter.matchesOrdered({ gte: 2 }, 2)).toBe(true);
      expect(filter.matchesOrdered({ lt: 2 }, 2)).toBe(false);
      expect(filter.matchesOrdered({ lte: 2 }, 2)).toBe(true);
    });

    it("should AND the bounds with eq and in", () => {
      expect(filter.matchesOrdered({ gt: 1, lt: 3, in: [2, 4] }, 2)).toBe(true);
      expect(filter.matchesOrdered({ gt: 1, lt: 3, in: [2, 4] }, 4)).toBe(false);
    });

    it("should compare timestamps by value", () => {
      const op = { gte: new TimeStamp(5), lt: new TimeStamp(10) };
      expect(filter.matchesOrdered(op, new TimeStamp(5))).toBe(true);
      expect(filter.matchesOrdered(op, new TimeStamp(10))).toBe(false);
    });
  });

  describe("matchesContains", () => {
    it("should match an array that holds an element of any", () => {
      expect(filter.matchesContains({ any: ["a", "b"] }, ["c", "b"])).toBe(true);
      expect(filter.matchesContains({ any: ["a"] }, ["c"])).toBe(false);
    });

    it("should match every array when any is not set", () => {
      expect(filter.matchesContains({}, [])).toBe(true);
    });

    it("should match no array with an empty any", () => {
      expect(filter.matchesContains({ any: [] }, ["a"])).toBe(false);
    });
  });
});
