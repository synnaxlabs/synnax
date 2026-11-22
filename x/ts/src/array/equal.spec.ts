// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { array } from "@/array";

describe("equal", () => {
  it("should be true for the same reference", () => {
    const a = [1, 2, 3];
    expect(array.equal(a, a)).toBe(true);
  });

  it("should be true for arrays with the same elements in the same order", () => {
    expect(array.equal([1, 2, 3], [1, 2, 3])).toBe(true);
    expect(array.equal(["a", "a"], ["a", "a"])).toBe(true);
  });

  it("should be true for two empty arrays", () => {
    expect(array.equal([], [])).toBe(true);
  });

  it("should be false when the lengths differ", () => {
    expect(array.equal([1, 2], [1, 2, 3])).toBe(false);
    expect(array.equal([], [1])).toBe(false);
  });

  it("should be false when an element differs", () => {
    expect(array.equal([1, 2, 3], [1, 9, 3])).toBe(false);
  });

  it("should be false for the same elements in a different order", () => {
    expect(array.equal([1, 2, 3], [3, 2, 1])).toBe(false);
  });

  it("should compare objects by reference", () => {
    const a = { v: 1 };
    expect(array.equal([a], [a])).toBe(true);
    expect(array.equal([a], [{ v: 1 }])).toBe(false);
  });

  it("should be false for NaN, since NaN !== NaN", () => {
    expect(array.equal([NaN], [NaN])).toBe(false);
  });

  it("should accept readonly arrays", () => {
    const a: readonly number[] = [1, 2];
    const b: readonly number[] = [1, 2];
    expect(array.equal(a, b)).toBe(true);
  });
});

describe("equalUnordered", () => {
  it("should be true for the same elements in any order", () => {
    expect(array.equalUnordered([1, 2, 3], [3, 1, 2])).toBe(true);
  });

  it("should be true for two empty arrays", () => {
    expect(array.equalUnordered([], [])).toBe(true);
  });

  it("should be true when duplicates have the same counts", () => {
    expect(array.equalUnordered(["a", "b", "a"], ["a", "a", "b"])).toBe(true);
  });

  it("should be false when duplicates have different counts", () => {
    expect(array.equalUnordered(["a", "a", "b"], ["a", "b", "b"])).toBe(false);
  });

  it("should be false when the lengths differ", () => {
    expect(array.equalUnordered([1, 2], [1, 2, 2])).toBe(false);
  });

  it("should be false when an element is missing", () => {
    expect(array.equalUnordered([1, 2, 3], [1, 2, 4])).toBe(false);
  });

  it("should match NaN with NaN", () => {
    expect(array.equalUnordered([NaN, 1], [1, NaN])).toBe(true);
  });

  it("should compare objects by reference", () => {
    const a = { v: 1 };
    expect(array.equalUnordered([a], [a])).toBe(true);
    expect(array.equalUnordered([a], [{ v: 1 }])).toBe(false);
  });
});

describe("equalAsSets", () => {
  it("should be true for the same distinct elements in any order", () => {
    expect(array.equalAsSets([1, 2, 3], [3, 2, 1])).toBe(true);
  });

  it("should be true for two empty arrays", () => {
    expect(array.equalAsSets([], [])).toBe(true);
  });

  it("should ignore duplicates", () => {
    expect(array.equalAsSets(["a", "a", "b"], ["b", "a"])).toBe(true);
    expect(array.equalAsSets(["a", "a", "b"], ["a", "b", "b"])).toBe(true);
  });

  it("should be false when an element is missing", () => {
    expect(array.equalAsSets([1, 2], [1, 2, 3])).toBe(false);
    expect(array.equalAsSets([1, 2], [1, 3])).toBe(false);
  });

  it("should be false for an empty and a non-empty array", () => {
    expect(array.equalAsSets([], [1])).toBe(false);
  });

  it("should match NaN with NaN", () => {
    expect(array.equalAsSets([NaN, NaN], [NaN])).toBe(true);
  });
});
