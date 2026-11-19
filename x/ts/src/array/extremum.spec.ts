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

describe("extremum", () => {
  const items = [
    { name: "a", size: 2 },
    { name: "b", size: 5 },
    { name: "c", size: -1 },
    { name: "d", size: 5 },
    { name: "e", size: -1 },
  ];

  describe("maxBy", () => {
    it("should return the first value with the largest key", () => {
      expect(array.maxBy(items, ({ size }) => size)?.name).toEqual("b");
    });
    it("should return undefined for an empty array", () => {
      expect(array.maxBy([], (v: number) => v)).toBeUndefined();
    });
    it("should handle arrays too long to spread into Math.max", () => {
      const values = Array.from({ length: 500_000 }, (_, i) => i);
      expect(array.maxBy(values, (v) => v)).toEqual(499_999);
    });
  });

  describe("minBy", () => {
    it("should return the first value with the smallest key", () => {
      expect(array.minBy(items, ({ size }) => size)?.name).toEqual("c");
    });
    it("should return undefined for an empty array", () => {
      expect(array.minBy([], (v: number) => v)).toBeUndefined();
    });
  });
});
