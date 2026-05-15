// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { telem } from "@synnaxlabs/x/telem";
import type { Haul } from "@synnaxlabs/lyra/haul";
import { type ranger } from "@synnaxlabs/client";

import { describe, expect, it } from "vitest";

import {
  canDropHaulItem,
  createHaulItem,
  filterHaulItems,
  HAUL_TYPE,
  isHaulItem,
} from "@/ranger/types";

const PAYLOAD: ranger.Payload = {
  key: "550e8400-e29b-41d4-a716-446655440000",
  name: "Test Range",
  timeRange: new telem.TimeRange(telem.TimeStamp.now(), telem.TimeStamp.now().add(telem.TimeStamp.SECOND)),
  labels: [],
};

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("range haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the range HAUL_TYPE", () => {
      expect(createHaulItem(PAYLOAD).type).toEqual(HAUL_TYPE);
    });

    it("creates an item with the payload key", () => {
      expect(createHaulItem(PAYLOAD).key).toEqual(PAYLOAD.key);
    });

    it("creates an item carrying serializable payload fields in data", () => {
      expect(createHaulItem(PAYLOAD).data).toEqual({
        key: PAYLOAD.key,
        name: PAYLOAD.name,
        timeRange: PAYLOAD.timeRange.numeric,
      });
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an item of the range kind", () => {
      expect(isHaulItem(createHaulItem(PAYLOAD))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps range items and drops items of other kinds", () => {
      const item = createHaulItem(PAYLOAD);
      expect(filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is a range item", () => {
      expect(
        canDropHaulItem({ source: OTHER, items: [createHaulItem(PAYLOAD), OTHER] }),
      ).toBe(true);
    });

    it("returns false when no item is a range item", () => {
      expect(canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
