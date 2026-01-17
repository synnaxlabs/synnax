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

describe("toArray", () => {
  it("should return the same array if input is already an array", () => {
    const input = [1, 2, 3];
    expect(array.toArray(input)).toBe(input);
  });

  it("should wrap a single value in an array", () => {
    expect(array.toArray(1)).toEqual([1]);
    expect(array.toArray("test")).toEqual(["test"]);
    expect(array.toArray({ key: "value" })).toEqual([{ key: "value" }]);
  });

  it("should handle empty arrays", () => {
    const input: number[] = [];
    expect(array.toArray(input)).toBe(input);
  });
});

describe("nullToArray", () => {
  it("should return the same array if input is already an array", () => {
    const input = [1, 2, 3];
    expect(array.toArray(input)).toBe(input);
  });

  it("should wrap a single value in an array", () => {
    expect(array.toArray(1)).toEqual([1]);
    expect(array.toArray("test")).toEqual(["test"]);
    expect(array.toArray({ key: "value" })).toEqual([{ key: "value" }]);
  });

  it("should return an empty array for null input", () => {
    expect(array.toArray(null)).toEqual([]);
  });

  it("should return an empty array for undefined input", () => {
    expect(array.toArray(undefined)).toEqual([]);
  });

  it("should handle empty arrays", () => {
    const input: number[] = [];
    expect(array.toArray(input)).toBe(input);
  });
});
