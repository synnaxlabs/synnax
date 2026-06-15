// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type user } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { type Haul } from "@/haul";
import {
  canDropHaulItem,
  createHaulItem,
  filterHaulItems,
  HAUL_TYPE,
  isHaulItem,
} from "@/user/types";

const USER: user.User = {
  key: "550e8400-e29b-41d4-a716-446655440000",
  username: "synnax",
  firstName: "Test",
  lastName: "User",
  avatar: "",
  rootUser: false,
};

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("user haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the user HAUL_TYPE", () => {
      expect(createHaulItem(USER).type).toEqual(HAUL_TYPE);
    });

    it("creates an item with the user payload key", () => {
      expect(createHaulItem(USER).key).toEqual(USER.key);
    });

    it("creates an item carrying the full user payload in data", () => {
      expect(createHaulItem(USER).data).toEqual(USER);
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an item of the user kind", () => {
      expect(isHaulItem(createHaulItem(USER))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps user items and drops items of other kinds", () => {
      const item = createHaulItem(USER);
      expect(filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is a user item", () => {
      expect(
        canDropHaulItem({ source: OTHER, items: [createHaulItem(USER), OTHER] }),
      ).toBe(true);
    });

    it("returns false when no item is a user item", () => {
      expect(canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
