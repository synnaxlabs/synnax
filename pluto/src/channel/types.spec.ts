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

import { Channel } from "@/channel";

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("channel haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the channel HAUL_TYPE", () => {
      expect(Channel.createHaulItem(42).type).toEqual(Channel.HAUL_TYPE);
    });

    it("creates an item with the provided key", () => {
      expect(Channel.createHaulItem(42).key).toEqual(42);
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an item of the channel kind", () => {
      expect(Channel.isHaulItem(Channel.createHaulItem(1))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(Channel.isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps channel items and drops items of other kinds", () => {
      const item = Channel.createHaulItem(1);
      expect(Channel.filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is a channel item", () => {
      const item = Channel.createHaulItem(1);
      expect(Channel.canDropHaulItem({ source: OTHER, items: [item, OTHER] })).toBe(
        true,
      );
    });

    it("returns false when no item is a channel item", () => {
      expect(Channel.canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
