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
  isHaulItem,
} from "@/hardware/opc/device/Browser";
import { type ScannedNode } from "@/hardware/opc/task/types";

const NODE: ScannedNode = {
  key: "ns=2;s=Demo.Static.Scalar.Float",
  nodeId: "ns=2;s=Demo.Static.Scalar.Float",
  name: "Demo Float",
  nodeClass: "Variable",
  dataType: "float32",
  isArray: false,
};

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("opc device haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the opc HAUL_TYPE", () => {
      expect(createHaulItem(NODE).type).toEqual(HAUL_TYPE);
    });

    it("creates an item with the node nodeId as key", () => {
      expect(createHaulItem(NODE).key).toEqual(NODE.nodeId);
    });

    it("creates an item carrying the full scanned node in data", () => {
      expect(createHaulItem(NODE).data).toEqual(NODE);
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an opc item", () => {
      expect(isHaulItem(createHaulItem(NODE))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps opc items and drops items of other kinds", () => {
      const item = createHaulItem(NODE);
      expect(filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is an opc item", () => {
      expect(
        canDropHaulItem({ source: OTHER, items: [createHaulItem(NODE), OTHER] }),
      ).toBe(true);
    });

    it("returns false when no item is an opc item", () => {
      expect(canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
