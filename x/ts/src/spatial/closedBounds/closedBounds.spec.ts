// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { closedBounds } from "@/spatial/closedBounds";

describe("ClosedBounds", () => {
  describe("schema", () => {
    it("should parse number bounds", () => {
      expect(closedBounds.closedBoundsZ().parse({ lower: 0, upper: 10 })).toEqual({
        lower: 0,
        upper: 10,
      });
    });
  });

  describe("contains", () => {
    const b: closedBounds.ClosedBounds = { lower: 0, upper: 10 };
    it.each([
      ["a value inside", 5, true],
      ["the lower bound", 0, true],
      ["the upper bound", 10, true],
      ["a value below", -0.1, false],
      ["a value above", 10.1, false],
    ])("should test %s", (_, value, expected) => {
      expect(closedBounds.contains(b, value)).toEqual(expected);
    });

    it("should include the upper bound of bigint bounds", () => {
      expect(closedBounds.contains({ lower: 0n, upper: 10n }, 10n)).toEqual(true);
    });

    it.each([
      ["equal bounds", { lower: 0, upper: 10 }, true],
      ["bounds inside", { lower: 2, upper: 8 }, true],
      ["bounds past the upper", { lower: 2, upper: 11 }, false],
      ["bounds past the lower", { lower: -1, upper: 8 }, false],
    ])("should test %s", (_, target, expected) => {
      expect(closedBounds.contains(b, target)).toEqual(expected);
    });

    it("should contain nothing when the lower is above the upper", () => {
      expect(closedBounds.contains({ lower: 10, upper: 0 }, 5)).toEqual(false);
    });
  });

  describe("span", () => {
    it("should return upper minus lower", () => {
      expect(closedBounds.span({ lower: -10, upper: 10 })).toEqual(20);
    });
    it("should return a bigint span for bigint bounds", () => {
      expect(closedBounds.span({ lower: 1n, upper: 4n })).toEqual(3n);
    });
    it("should return a negative span when the lower is above the upper", () => {
      expect(closedBounds.span({ lower: 2, upper: 1 })).toEqual(-1);
    });
  });
});
