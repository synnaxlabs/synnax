// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { narrow } from "@/narrow";

describe("narrow", () => {
  describe("isObject", () => {
    it("should return true for plain objects", () => {
      expect(narrow.isObject({})).toBe(true);
      expect(narrow.isObject({ a: 1 })).toBe(true);
      expect(narrow.isObject({ nested: { obj: true } })).toBe(true);
    });

    it("should return false for null", () => {
      expect(narrow.isObject(null)).toBe(false);
    });

    it("should return false for undefined", () => {
      expect(narrow.isObject(undefined)).toBe(false);
      expect(narrow.isObject()).toBe(false);
    });

    it("should return false for arrays", () => {
      expect(narrow.isObject([])).toBe(false);
      expect(narrow.isObject([1, 2, 3])).toBe(false);
      expect(narrow.isObject([{ a: 1 }])).toBe(false);
    });

    it("should return false for primitives", () => {
      expect(narrow.isObject(0)).toBe(false);
      expect(narrow.isObject(1)).toBe(false);
      expect(narrow.isObject("")).toBe(false);
      expect(narrow.isObject("string")).toBe(false);
      expect(narrow.isObject(true)).toBe(false);
      expect(narrow.isObject(false)).toBe(false);
      expect(narrow.isObject(Symbol("test"))).toBe(false);
      expect(narrow.isObject(42n)).toBe(false);
    });

    it("should return true for class instances", () => {
      class TestClass {}
      expect(narrow.isObject(new TestClass())).toBe(true);
      expect(narrow.isObject(new Date())).toBe(true);
      expect(narrow.isObject(new Map())).toBe(true);
      expect(narrow.isObject(new Set())).toBe(true);
    });

    it("should return true for Object.create(null)", () => {
      expect(narrow.isObject(Object.create(null))).toBe(true);
    });

    it("should return false for functions", () => {
      expect(narrow.isObject(() => {})).toBe(false);
      expect(narrow.isObject(function () {})).toBe(false);
      expect(narrow.isObject(async () => {})).toBe(false);
    });

    it("should return true for regex", () => {
      expect(narrow.isObject(/test/)).toBe(true);
      expect(narrow.isObject(new RegExp("test"))).toBe(true);
    });
  });
});
