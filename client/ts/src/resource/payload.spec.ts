// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { resource } from "@/resource";

describe("resource", () => {
  describe("idZ", () => {
    it("should parse an ID object", () => {
      expect(resource.idZ.parse({ type: "group", key: "one" })).toEqual({
        type: "group",
        key: "one",
      });
    });

    it("should parse a string ID", () => {
      expect(resource.idZ.parse("group:one")).toEqual({ type: "group", key: "one" });
    });

    it("should parse a string ID with no key as an empty key", () => {
      expect(resource.idZ.parse("group")).toEqual({ type: "group", key: "" });
    });

    it("should reject an invalid resource type without throwing", () => {
      expect(resource.idZ.validate("nonsense:key")).toBe(false);
    });
  });

  describe("parseID", () => {
    it("should parse an ID object", () => {
      expect(resource.parseID({ type: "channel", key: "1" })).toEqual({
        type: "channel",
        key: "1",
      });
    });

    it("should parse a string ID", () => {
      expect(resource.parseID("channel:1")).toEqual({ type: "channel", key: "1" });
    });

    it("should throw a labeled error for an invalid resource type", () => {
      expect(() => resource.parseID("dog:1")).toThrow(
        "Failed to parse resource ID (1 issue)",
      );
    });
  });

  describe("createIDFactory", () => {
    const resourceID = resource.createIDFactory<number>("channel");

    it("should stamp the resource type onto a key", () => {
      expect(resourceID(12)).toEqual({ type: "channel", key: "12" });
    });

    it("should stamp the resource type onto an array of keys", () => {
      expect(resourceID([1, 2])).toEqual([
        { type: "channel", key: "1" },
        { type: "channel", key: "2" },
      ]);
    });

    it("should map a zero key to an empty key", () => {
      expect(resourceID(0)).toEqual({ type: "channel", key: "" });
    });
  });

  describe("idsEqual", () => {
    it("should return true for IDs with the same type and key", () => {
      expect(
        resource.idsEqual({ type: "group", key: "one" }, { type: "group", key: "one" }),
      ).toBe(true);
    });

    it("should return false for IDs with different keys", () => {
      expect(
        resource.idsEqual({ type: "group", key: "one" }, { type: "group", key: "two" }),
      ).toBe(false);
    });

    it("should return false for IDs with different types", () => {
      expect(
        resource.idsEqual(
          { type: "group", key: "one" },
          { type: "channel", key: "one" },
        ),
      ).toBe(false);
    });
  });

  describe("idToString", () => {
    it("should convert an ID to a string", () => {
      expect(resource.idToString({ type: "group", key: "one" })).toEqual("group:one");
    });

    it("should convert an array of IDs to strings", () => {
      expect(
        resource.idToString([
          { type: "group", key: "one" },
          { type: "channel", key: "two" },
        ]),
      ).toEqual(["group:one", "channel:two"]);
    });

    it("should pass through string IDs", () => {
      expect(resource.idToString("group:one")).toEqual("group:one");
    });

    it("should validate string IDs that get passed", () => {
      expect(() => resource.idToString("dog")).toThrow(
        "Failed to parse resource ID (1 issue)",
      );
    });

    it("should pass through an array of string IDs", () => {
      expect(resource.idToString(["group:one", "channel:two"])).toEqual([
        "group:one",
        "channel:two",
      ]);
    });

    it("should validate an array of string IDs that get passed", () => {
      expect(() => resource.idToString(["group:one", "channel:two", "dog"])).toThrow(
        "Failed to parse resource ID (1 issue)",
      );
    });
  });
});
