// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Haul } from "@synnaxlabs/lyra/haul";
import { describe, expect, it } from "vitest";

import { Mosaic } from "@/mosaic";

const TAB_KEY = "tab-1";
const ELEMENT_ID = "element-1";
const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("mosaic tab-drop haul utilities", () => {
  describe("createTabDropHaulItem", () => {
    it("creates an item with the tab-drop HAUL_TYPE", () => {
      expect(Mosaic.createTabDropHaulItem(TAB_KEY).type).toEqual(Mosaic.HAUL_DROP_TYPE);
    });

    it("creates an item with the provided tab key", () => {
      expect(Mosaic.createTabDropHaulItem(TAB_KEY).key).toEqual(TAB_KEY);
    });

    it("attaches the optional elementID when provided", () => {
      expect(Mosaic.createTabDropHaulItem(TAB_KEY, ELEMENT_ID).elementID).toEqual(
        ELEMENT_ID,
      );
    });
  });

  describe("isTabDropHaulItem", () => {
    it("returns true for a tab-drop item", () => {
      expect(Mosaic.isTabDropHaulItem(Mosaic.createTabDropHaulItem(TAB_KEY))).toBe(
        true,
      );
    });

    it("returns false for a tab-create item", () => {
      expect(Mosaic.isTabDropHaulItem(Mosaic.createTabCreateHaulItem(TAB_KEY))).toBe(
        false,
      );
    });

    it("returns false for an item of another kind", () => {
      expect(Mosaic.isTabDropHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterTabDropHaulItems", () => {
    it("keeps only tab-drop items", () => {
      const drop = Mosaic.createTabDropHaulItem(TAB_KEY);
      const create = Mosaic.createTabCreateHaulItem(TAB_KEY);
      expect(Mosaic.filterTabDropHaulItems([drop, create, OTHER])).toEqual([drop]);
    });
  });

  describe("canDropTabDropHaulItem", () => {
    it("returns true when at least one item is a tab-drop item", () => {
      expect(
        Mosaic.canDropTabDropHaulItem({
          source: OTHER,
          items: [Mosaic.createTabDropHaulItem(TAB_KEY), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when no item is a tab-drop item", () => {
      expect(
        Mosaic.canDropTabDropHaulItem({
          source: OTHER,
          items: [Mosaic.createTabCreateHaulItem(TAB_KEY)],
        }),
      ).toBe(false);
    });
  });
});

describe("mosaic tab-create haul utilities", () => {
  describe("createTabCreateHaulItem", () => {
    it("creates an item with the tab-create HAUL_TYPE", () => {
      expect(Mosaic.createTabCreateHaulItem(TAB_KEY).type).toEqual(
        Mosaic.HAUL_CREATE_TYPE,
      );
    });

    it("creates an item with the provided tab key", () => {
      expect(Mosaic.createTabCreateHaulItem(TAB_KEY).key).toEqual(TAB_KEY);
    });
  });

  describe("isTabCreateHaulItem", () => {
    it("returns true for a tab-create item", () => {
      expect(Mosaic.isTabCreateHaulItem(Mosaic.createTabCreateHaulItem(TAB_KEY))).toBe(
        true,
      );
    });

    it("returns false for a tab-drop item", () => {
      expect(Mosaic.isTabCreateHaulItem(Mosaic.createTabDropHaulItem(TAB_KEY))).toBe(
        false,
      );
    });

    it("returns false for an item of another kind", () => {
      expect(Mosaic.isTabCreateHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterTabCreateHaulItems", () => {
    it("keeps only tab-create items", () => {
      const drop = Mosaic.createTabDropHaulItem(TAB_KEY);
      const create = Mosaic.createTabCreateHaulItem(TAB_KEY);
      expect(Mosaic.filterTabCreateHaulItems([drop, create, OTHER])).toEqual([create]);
    });
  });

  describe("canDropTabCreateHaulItem", () => {
    it("returns true when at least one item is a tab-create item", () => {
      expect(
        Mosaic.canDropTabCreateHaulItem({
          source: OTHER,
          items: [Mosaic.createTabCreateHaulItem(TAB_KEY), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when no item is a tab-create item", () => {
      expect(
        Mosaic.canDropTabCreateHaulItem({
          source: OTHER,
          items: [Mosaic.createTabDropHaulItem(TAB_KEY)],
        }),
      ).toBe(false);
    });
  });
});
