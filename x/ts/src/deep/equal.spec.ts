// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, it, expect } from "vitest";

import { deep } from "@/deep";

interface TestRecord {
  a: number;
  b: {
    c?: number;
    d?: number;
  };
}

describe("deepEqual", () => {
  describe("deepPartialEqual", () => {
    it("should return true for overlapping objects", () => {
      const base: TestRecord = { a: 1, b: { c: 2, d: 3 } };
      const partial: TestRecord = { a: 1, b: { c: 2 } };
      expect(deep.partialEqual(base, partial)).toBe(true);
    });
    it("should return false for non-overlapping objects", () => {
      const base = { a: 1, b: { c: 2, d: 3 } };
      const partial = { a: 1, b: { c: 2, d: 4 } };
      expect(deep.partialEqual(base, partial)).toBe(false);
    });
  });
  describe("deepEqual", () => {
    it("should return true for equal objects", () => {
      const a = { a: 1, b: { c: 2, d: 3 } };
      const b = { a: 1, b: { c: 2, d: 3 } };
      expect(deep.equal(a, b)).toBe(true);
    });
    it("should return false for non-equal objects", () => {
      const a = { a: 1, b: { c: 2, d: 3 } };
      const b = { a: 1, b: { c: 2, d: 4 } };
      expect(deep.equal(a, b)).toBe(false);
    });
    it("should return false for partial objects", () => {
      const a = { a: 1, b: { c: 2, d: 3 } };
      const b = { a: 1, b: { c: 2 } };
      expect(deep.equal(a, b)).toBe(false);
    });
  });
});
