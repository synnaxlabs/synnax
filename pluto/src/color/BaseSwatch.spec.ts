// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Haul } from "@synnaxlabs/charon/haul";
import { describe, expect, it } from "vitest";

import {
  canDropHaulItem,
  createHaulItem,
  filterHaulItems,
  HAUL_TYPE,
  isHaulItem,
} from "@/color/BaseSwatch";

const HEX = "#ff0000";
const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("color haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the color HAUL_TYPE", () => {
      expect(createHaulItem(HEX).type).toEqual(HAUL_TYPE);
    });

    it("creates an item with the provided hex key", () => {
      expect(createHaulItem(HEX).key).toEqual(HEX);
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an item of the color kind", () => {
      expect(isHaulItem(createHaulItem(HEX))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps color items and drops items of other kinds", () => {
      const item = createHaulItem(HEX);
      expect(filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is a color item", () => {
      expect(
        canDropHaulItem({ source: OTHER, items: [createHaulItem(HEX), OTHER] }),
      ).toBe(true);
    });

    it("returns false when no item is a color item", () => {
      expect(canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
