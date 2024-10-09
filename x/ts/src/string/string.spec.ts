// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { naturalLanguageJoin } from "./string";

describe("naturalLanguageJoin", () => {
  it("should return an empty string for an empty array", () =>
    expect(naturalLanguageJoin([])).toBe(""));

  it("should return the zeroLength string for an empty array if provided", () =>
    expect(naturalLanguageJoin([], "No items")).toBe("No items"));

  it("should return the single element for an array with one element", () =>
    expect(naturalLanguageJoin(["apple"])).toBe("apple"));

  it('should join two elements with "and"', () =>
    expect(naturalLanguageJoin(["apple", "banana"])).toBe("apple and banana"));

  it('should join multiple elements with commas and "and"', () =>
    expect(naturalLanguageJoin(["apple", "banana", "cherry"])).toBe(
      "apple, banana, and cherry",
    ));

  it("should handle an array with more than three elements correctly", () =>
    expect(naturalLanguageJoin(["apple", "banana", "cherry", "date"])).toBe(
      "apple, banana, cherry, and date",
    ));
});
