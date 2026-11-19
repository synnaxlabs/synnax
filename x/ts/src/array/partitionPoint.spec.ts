// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { array } from "@/array";
import { compare } from "@/compare";

describe("partitionPoint", () => {
  const lowerBound = (arr: number[], v: number): number =>
    array.partitionPoint(arr.length, (i) => arr[i] >= v);
  const upperBound = (arr: number[], v: number): number =>
    array.partitionPoint(arr.length, (i) => arr[i] > v);

  it("should return 0 when n is 0", () => {
    const pred = vi.fn(() => true);
    expect(array.partitionPoint(0, pred)).toBe(0);
    expect(pred).not.toHaveBeenCalled();
  });

  it("should return 0 when pred is true for every index", () => {
    expect(array.partitionPoint(5, () => true)).toBe(0);
  });

  it("should return n when pred is true for no index", () => {
    expect(array.partitionPoint(5, () => false)).toBe(5);
  });

  it("should return the first index where pred is true", () => {
    const arr = [1, 3, 5, 7];
    expect(lowerBound(arr, 0)).toBe(0);
    expect(lowerBound(arr, 1)).toBe(0);
    expect(lowerBound(arr, 4)).toBe(2);
    expect(lowerBound(arr, 7)).toBe(3);
    expect(lowerBound(arr, 8)).toBe(4);
  });

  it("should return the first of duplicates for a lower bound", () => {
    expect(lowerBound([1, 2, 2, 2, 3], 2)).toBe(1);
  });

  it("should return the index after duplicates for an upper bound", () => {
    expect(upperBound([1, 2, 2, 2, 3], 2)).toBe(4);
  });

  it("should handle a single element", () => {
    expect(lowerBound([5], 4)).toBe(0);
    expect(lowerBound([5], 5)).toBe(0);
    expect(lowerBound([5], 6)).toBe(1);
  });

  it("should find the insert position for a comparator", () => {
    const names = ["alpha", "Bravo", "charlie"];
    const at = array.partitionPoint(
      names.length,
      (i) => compare.text(names[i], "beta") >= 0,
    );
    expect(at).toBe(1);
  });

  it("should call pred a logarithmic number of times", () => {
    const pred = vi.fn((i: number) => i >= 700);
    expect(array.partitionPoint(1024, pred)).toBe(700);
    expect(pred.mock.calls.length).toBeLessThanOrEqual(11);
  });
});
