// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Haul } from "@/haul";
import {
  canDropTabCreateHaulItem,
  canDropTabDropHaulItem,
  createTabCreateHaulItem,
  createTabDropHaulItem,
  filterTabCreateHaulItems,
  filterTabDropHaulItems,
  HAUL_CREATE_TYPE,
  HAUL_DROP_TYPE,
  isTabCreateHaulItem,
  isTabDropHaulItem,
} from "@/mosaic/haul";

const TAB_KEY = "tab-1";
const ELEMENT_ID = "element-1";
const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("mosaic tab-drop haul utilities", () => {
  describe("createTabDropHaulItem", () => {
    it("creates an item with the tab-drop HAUL_TYPE", () => {
      expect(createTabDropHaulItem(TAB_KEY).type).toEqual(HAUL_DROP_TYPE);
    });

    it("creates an item with the provided tab key", () => {
      expect(createTabDropHaulItem(TAB_KEY).key).toEqual(TAB_KEY);
    });

    it("attaches the optional elementID when provided", () => {
      expect(createTabDropHaulItem(TAB_KEY, ELEMENT_ID).elementID).toEqual(ELEMENT_ID);
    });
  });

  describe("isTabDropHaulItem", () => {
    it("returns true for a tab-drop item", () => {
      expect(isTabDropHaulItem(createTabDropHaulItem(TAB_KEY))).toBe(true);
    });

    it("returns false for a tab-create item", () => {
      expect(isTabDropHaulItem(createTabCreateHaulItem(TAB_KEY))).toBe(false);
    });

    it("returns false for an item of another kind", () => {
      expect(isTabDropHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterTabDropHaulItems", () => {
    it("keeps only tab-drop items", () => {
      const drop = createTabDropHaulItem(TAB_KEY);
      const create = createTabCreateHaulItem(TAB_KEY);
      expect(filterTabDropHaulItems([drop, create, OTHER])).toEqual([drop]);
    });
  });

  describe("canDropTabDropHaulItem", () => {
    it("returns true when at least one item is a tab-drop item", () => {
      expect(
        canDropTabDropHaulItem({
          source: OTHER,
          items: [createTabDropHaulItem(TAB_KEY), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when no item is a tab-drop item", () => {
      expect(
        canDropTabDropHaulItem({
          source: OTHER,
          items: [createTabCreateHaulItem(TAB_KEY)],
        }),
      ).toBe(false);
    });
  });
});

describe("mosaic tab-create haul utilities", () => {
  describe("createTabCreateHaulItem", () => {
    it("creates an item with the tab-create HAUL_TYPE", () => {
      expect(createTabCreateHaulItem(TAB_KEY).type).toEqual(HAUL_CREATE_TYPE);
    });

    it("creates an item with the provided tab key", () => {
      expect(createTabCreateHaulItem(TAB_KEY).key).toEqual(TAB_KEY);
    });
  });

  describe("isTabCreateHaulItem", () => {
    it("returns true for a tab-create item", () => {
      expect(isTabCreateHaulItem(createTabCreateHaulItem(TAB_KEY))).toBe(true);
    });

    it("returns false for a tab-drop item", () => {
      expect(isTabCreateHaulItem(createTabDropHaulItem(TAB_KEY))).toBe(false);
    });

    it("returns false for an item of another kind", () => {
      expect(isTabCreateHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterTabCreateHaulItems", () => {
    it("keeps only tab-create items", () => {
      const drop = createTabDropHaulItem(TAB_KEY);
      const create = createTabCreateHaulItem(TAB_KEY);
      expect(filterTabCreateHaulItems([drop, create, OTHER])).toEqual([create]);
    });
  });

  describe("canDropTabCreateHaulItem", () => {
    it("returns true when at least one item is a tab-create item", () => {
      expect(
        canDropTabCreateHaulItem({
          source: OTHER,
          items: [createTabCreateHaulItem(TAB_KEY), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when no item is a tab-create item", () => {
      expect(
        canDropTabCreateHaulItem({
          source: OTHER,
          items: [createTabDropHaulItem(TAB_KEY)],
        }),
      ).toBe(false);
    });
  });
});
