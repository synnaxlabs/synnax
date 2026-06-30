// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Range } from "@/layered/session/range";

const STATIC: Range.Static = {
  key: "static-1",
  name: "Static 1",
  persisted: true,
  variant: "static",
  timeRange: { start: 0, end: 1000 },
};

const DYNAMIC: Range.Dynamic = {
  key: "dynamic-1",
  name: "Dynamic 1",
  persisted: false,
  variant: "dynamic",
  span: 1000,
};

const storeWith = (ranges: Range.Range[], selected?: string): Range.StoreState => ({
  [Range.SLICE_NAME]: { ...Range.sliceStateZ.parse({ ranges: [] }), ranges, selected },
});

describe("range selectors", () => {
  describe("selectSelectedKey", () => {
    it("should return the selected key", () => {
      const store = storeWith([STATIC], STATIC.key);
      expect(Range.selectSelectedKey(store)).toEqual(STATIC.key);
    });

    it("should return undefined when no range is selected", () => {
      expect(Range.selectSelectedKey(storeWith([STATIC]))).toBeUndefined();
    });
  });

  describe("selectStatic", () => {
    it("should return the static range matching the given key", () => {
      const store = storeWith([STATIC, DYNAMIC]);
      expect(Range.selectStatic(store, STATIC.key)).toEqual(STATIC);
    });

    it("should fall back to the selected range when no key is given", () => {
      const store = storeWith([STATIC, DYNAMIC], STATIC.key);
      expect(Range.selectStatic(store)).toEqual(STATIC);
    });

    it("should return undefined when the matching range is dynamic", () => {
      const store = storeWith([DYNAMIC]);
      expect(Range.selectStatic(store, DYNAMIC.key)).toBeUndefined();
    });

    it("should return undefined when no range matches", () => {
      expect(Range.selectStatic(storeWith([STATIC]), "missing")).toBeUndefined();
    });
  });

  describe("selectMultiple", () => {
    it("should return all ranges when no keys are given", () => {
      const store = storeWith([STATIC, DYNAMIC]);
      expect(Range.selectMultiple(store)).toEqual([STATIC, DYNAMIC]);
    });

    it("should return only the ranges matching the given keys", () => {
      const store = storeWith([STATIC, DYNAMIC]);
      expect(Range.selectMultiple(store, [DYNAMIC.key])).toEqual([DYNAMIC]);
    });
  });

  describe("selectKeys", () => {
    it("should return the keys of every range", () => {
      const store = storeWith([STATIC, DYNAMIC]);
      expect(Range.selectKeys(store)).toEqual([STATIC.key, DYNAMIC.key]);
    });
  });

  describe("selectStaticKeys", () => {
    it("should return only the keys of static ranges", () => {
      const store = storeWith([STATIC, DYNAMIC]);
      expect(Range.selectStaticKeys(store)).toEqual([STATIC.key]);
    });
  });
});
