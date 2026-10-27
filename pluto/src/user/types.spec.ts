// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type user } from "@synnaxlabs/client";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { describe, expect, it } from "vitest";

import { User } from "@/user";

const USER: user.User = {
  key: "550e8400-e29b-41d4-a716-446655440000",
  username: "synnax",
  firstName: "Test",
  lastName: "User",
  rootUser: false,
};

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("user haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the user HAUL_TYPE", () => {
      expect(User.createHaulItem(USER).type).toEqual(User.HAUL_TYPE);
    });

    it("creates an item with the user payload key", () => {
      expect(User.createHaulItem(USER).key).toEqual(USER.key);
    });

    it("creates an item carrying the full user payload in data", () => {
      expect(User.createHaulItem(USER).data).toEqual(USER);
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an item of the user kind", () => {
      expect(User.isHaulItem(User.createHaulItem(USER))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(User.isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps user items and drops items of other kinds", () => {
      const item = User.createHaulItem(USER);
      expect(User.filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is a user item", () => {
      expect(
        User.canDropHaulItem({
          source: OTHER,
          items: [User.createHaulItem(USER), OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when no item is a user item", () => {
      expect(User.canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});
