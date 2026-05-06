// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Haul } from "@synnaxlabs/charon";
import { describe, expect, it } from "vitest";

import {
  canDropHaulItem,
  createHaulItem,
  filterHaulItems,
  HAUL_TYPE,
  isHaulItem,
} from "@/tree/Tree";

const KEY = "node-1";
const DEPTH = 3;
const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("tree haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the tree HAUL_TYPE", () => {
      expect(createHaulItem(KEY, DEPTH).type).toEqual(HAUL_TYPE);
    });

    it("creates an item with the provided key", () => {
      expect(createHaulItem(KEY, DEPTH).key).toEqual(KEY);
    });

    it("creates an item carrying the provided depth in data", () => {
      expect(createHaulItem(KEY, DEPTH).data).toEqual({ depth: DEPTH });
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an item of the tree kind", () => {
      expect(isHaulItem(createHaulItem(KEY, DEPTH))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps tree items and drops items of other kinds", () => {
      const item = createHaulItem(KEY, DEPTH);
      expect(filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is a tree item", () => {
      expect(
        canDropHaulItem({ source: OTHER, items: [createHaulItem(KEY, DEPTH), OTHER] }),
      ).toBe(true);
    });

    it("returns false when no item is a tree item", () => {
      expect(canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
