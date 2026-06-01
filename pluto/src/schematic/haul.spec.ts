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
  canDropHaulItem,
  createHaulItem,
  filterHaulItems,
  HAUL_TYPE,
  type HaulItemData,
  isHaulItem,
} from "@/schematic/haul";

const ITEM_DATA: HaulItemData = { key: "n1", variant: "tank" };

const foreignItem = (key: string, type = "other"): Haul.Item => ({ key, type });

describe("HAUL_TYPE", () => {
  it("is the schematic element type tag", () => {
    expect(HAUL_TYPE).toBe("schematic-element");
  });
});

describe("createHaulItem", () => {
  it("wraps AddNodeProps into a haul item keyed by the node key", () => {
    expect(createHaulItem(ITEM_DATA)).toEqual({
      type: HAUL_TYPE,
      key: ITEM_DATA.key,
      data: ITEM_DATA,
    });
  });

  it("preserves optional position, specKey, and config on the data payload", () => {
    const data: HaulItemData = {
      key: "n2",
      variant: "valve",
      position: { x: 5, y: 7 },
      specKey: "spec-1",
      config: { variant: "valve" },
    };
    expect(createHaulItem(data).data).toEqual(data);
  });
});

describe("isHaulItem", () => {
  it("returns true for items created by createHaulItem", () => {
    expect(isHaulItem(createHaulItem(ITEM_DATA))).toBe(true);
  });

  it("returns false for items of any other type", () => {
    expect(isHaulItem(foreignItem("x"))).toBe(false);
  });
});

describe("filterHaulItems", () => {
  it("keeps only schematic items and preserves their order", () => {
    const a = createHaulItem({ key: "a", variant: "tank" });
    const b = foreignItem("b");
    const c = createHaulItem({ key: "c", variant: "valve" });
    expect(filterHaulItems([a, b, c])).toEqual([a, c]);
  });

  it("returns an empty array when no items match", () => {
    expect(filterHaulItems([foreignItem("a"), foreignItem("b", "another")])).toEqual(
      [],
    );
  });

  it("returns an empty array for empty input", () => {
    expect(filterHaulItems([])).toEqual([]);
  });
});

describe("canDropHaulItem", () => {
  const draggingState = (items: Haul.Item[]): Haul.DraggingState => ({
    source: items[0] ?? { key: "", type: "" },
    items,
  });

  it("accepts a drag containing at least one schematic item", () => {
    expect(
      canDropHaulItem(draggingState([foreignItem("a"), createHaulItem(ITEM_DATA)])),
    ).toBe(true);
  });

  it("rejects a drag containing only foreign item types", () => {
    expect(
      canDropHaulItem(draggingState([foreignItem("a"), foreignItem("b", "another")])),
    ).toBe(false);
  });

  it("rejects an empty drag", () => {
    expect(canDropHaulItem(draggingState([]))).toBe(false);
  });
});
