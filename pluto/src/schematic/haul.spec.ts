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

import { Schematic } from "@/schematic";

const ITEM_DATA: Schematic.HaulItemData = { key: "n1", config: { variant: "tank" } };

const foreignItem = (key: string, type = "other"): Haul.Item => ({ key, type });

describe("HAUL_TYPE", () => {
  it("is the schematic element type tag", () => {
    expect(Schematic.HAUL_TYPE).toBe("schematic-element");
  });
});

describe("createHaulItem", () => {
  it("wraps AddNodeProps into a haul item keyed by the node key", () => {
    expect(Schematic.createHaulItem(ITEM_DATA)).toEqual({
      type: Schematic.HAUL_TYPE,
      key: ITEM_DATA.key,
      data: ITEM_DATA,
    });
  });

  it("preserves the position and config fields on the data payload", () => {
    const data: Schematic.HaulItemData<"valve"> = {
      key: "n2",
      position: { x: 5, y: 7 },
      config: { variant: "valve", label: { label: "V1" } },
    };
    expect(Schematic.createHaulItem(data).data).toEqual(data);
  });
});

describe("isHaulItem", () => {
  it("returns true for items created by createHaulItem", () => {
    expect(Schematic.isHaulItem(Schematic.createHaulItem(ITEM_DATA))).toBe(true);
  });

  it("returns false for items of any other type", () => {
    expect(Schematic.isHaulItem(foreignItem("x"))).toBe(false);
  });
});

describe("filterHaulItems", () => {
  it("keeps only schematic items and preserves their order", () => {
    const a = Schematic.createHaulItem({ key: "a", config: { variant: "tank" } });
    const b = foreignItem("b");
    const c = Schematic.createHaulItem({ key: "c", config: { variant: "valve" } });
    expect(Schematic.filterHaulItems([a, b, c])).toEqual([a, c]);
  });

  it("returns an empty array when no items match", () => {
    expect(
      Schematic.filterHaulItems([foreignItem("a"), foreignItem("b", "another")]),
    ).toEqual([]);
  });

  it("returns an empty array for empty input", () => {
    expect(Schematic.filterHaulItems([])).toEqual([]);
  });
});

describe("canDropHaulItem", () => {
  const draggingState = (items: Haul.Item[]): Haul.DraggingState => ({
    source: items[0] ?? { key: "", type: "" },
    items,
  });

  it("accepts a drag containing at least one schematic item", () => {
    expect(
      Schematic.canDropHaulItem(
        draggingState([foreignItem("a"), Schematic.createHaulItem(ITEM_DATA)]),
      ),
    ).toBe(true);
  });

  it("rejects a drag containing only foreign item types", () => {
    expect(
      Schematic.canDropHaulItem(
        draggingState([foreignItem("a"), foreignItem("b", "another")]),
      ),
    ).toBe(false);
  });

  it("rejects an empty drag", () => {
    expect(Schematic.canDropHaulItem(draggingState([]))).toBe(false);
  });
});
