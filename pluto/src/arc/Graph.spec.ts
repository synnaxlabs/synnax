// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Haul } from "@synnaxlabs/pluto";
import { describe, expect, it } from "vitest";

import {
  canDropHaulItem,
  createHaulItem,
  filterHaulItems,
  HAUL_TYPE,
} from "@/arc/Graph";

const KEY = "stage-key";
const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("arc element haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the arc element HAUL_TYPE", () => {
      expect(createHaulItem(KEY).type).toEqual(HAUL_TYPE);
    });

    it("creates an item with the provided key", () => {
      expect(createHaulItem(KEY).key).toEqual(KEY);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps arc element items and drops items of other kinds", () => {
      const item = createHaulItem(KEY);
      expect(filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is an arc element item", () => {
      expect(
        canDropHaulItem({ source: OTHER, items: [createHaulItem(KEY), OTHER] }),
      ).toBe(true);
    });

    it("returns false when no item is an arc element item", () => {
      expect(canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
