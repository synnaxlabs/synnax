// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color as xcolor } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { enumeration } from "@/enumeration";

const KEY = "5a7d3e1c-2f4b-4c6d-8e9f-0a1b2c3d4e5f";
const NAMED = "6b8e4f2d-3a5c-4d7e-9fa0-1b2c3d4e5f60";

const createValve = (): enumeration.Enumeration =>
  enumeration.enumerationZ.parse({
    key: KEY,
    name: "Valve",
    entries: [
      { key: "closed", value: 0, name: "Closed" },
      { key: "open", value: 1, name: "Open", color: { type: "named", key: NAMED } },
    ],
  });

describe("enumeration.reduceAll", () => {
  describe("set", () => {
    it("should set the name and keep the entries by identity", () => {
      const prev = createValve();
      const { next } = enumeration.reduceAll(prev, [
        enumeration.set({ path: ["name"], value: "Gate" }),
      ]);
      expect(next.name).toBe("Gate");
      expect(next.entries).toBe(prev.entries);
    });

    it("should rename an entry", () => {
      const { next, targets } = enumeration.reduceAll(createValve(), [
        enumeration.set({ path: ["entries", "open", "name"], value: "Opened" }),
      ]);
      expect(next.entries[1].name).toBe("Opened");
      expect(targets).toEqual(["entries.open.name"]);
    });

    it("should set a fixed color on an entry from a hex string", () => {
      const { next } = enumeration.reduceAll(createValve(), [
        enumeration.set({
          path: ["entries", "closed", "color"],
          value: { type: "fixed", value: "#ff0000" },
        }),
      ]);
      expect(next.entries[0].color).toEqual({
        type: "fixed",
        value: xcolor.construct("#ff0000"),
      });
    });

    it("should return an inverse that restores the enumeration", () => {
      const prev = createValve();
      const { next, inverse } = enumeration.reduceAll(prev, [
        enumeration.set({ path: ["entries", "open", "abnormal"], value: true }),
      ]);
      expect(next.entries[1].abnormal).toBe(true);
      expect(enumeration.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should hold an empty name for the Core to refuse", () => {
      const { next } = enumeration.reduceAll(createValve(), [
        enumeration.set({ path: ["name"], value: "" }),
      ]);
      expect(next.name).toBe("");
    });

    it("should throw for a path outside the editable fields", () => {
      expect(() =>
        enumeration.reduceAll(createValve(), [
          enumeration.set({ path: ["key"], value: KEY }),
        ]),
      ).toThrow('no editable field at "key"');
    });
  });

  describe("insert", () => {
    it("should insert an entry after another", () => {
      const { next, inverse } = enumeration.reduceAll(createValve(), [
        enumeration.insert({
          path: ["entries"],
          value: { key: "fault", value: 2, name: "Fault", abnormal: true },
          after: "closed",
        }),
      ]);
      expect(next.entries.map((e) => e.key)).toEqual(["closed", "fault", "open"]);
      expect(enumeration.reduceAll(next, inverse).next).toEqual(createValve());
    });
  });

  describe("move", () => {
    it("should move an entry to the start", () => {
      const { next } = enumeration.reduceAll(createValve(), [
        enumeration.move({ path: ["entries"], key: "open" }),
      ]);
      expect(next.entries.map((e) => e.key)).toEqual(["open", "closed"]);
    });
  });

  describe("remove", () => {
    it("should remove an entry and restore it through the inverse", () => {
      const prev = createValve();
      const { next, inverse } = enumeration.reduceAll(prev, [
        enumeration.remove({ path: ["entries"], key: "open" }),
      ]);
      expect(next.entries.map((e) => e.key)).toEqual(["closed"]);
      expect(enumeration.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should remove the color of an entry", () => {
      const { next } = enumeration.reduceAll(createValve(), [
        enumeration.remove({ path: ["entries", "open"], key: "color" }),
      ]);
      expect(next.entries[1].color).toBeUndefined();
    });
  });

  describe("create", () => {
    it("should replace the enumeration", () => {
      const created = { ...createValve(), name: "Created" };
      const { next, inverse } = enumeration.reduceAll(createValve(), [
        enumeration.create({ enumeration: created }),
      ]);
      expect(next).toEqual(created);
      expect(inverse).toEqual([]);
    });
  });
});

describe("enumeration.encode", () => {
  it("should write a fixed color as a hex string", () => {
    const state = createValve();
    const encoded = enumeration.encode(
      state,
      enumeration.set({
        path: ["entries", "closed", "color"],
        value: { type: "fixed", value: xcolor.construct("#ff0000") },
      }),
    );
    expect(encoded).toEqual(
      enumeration.set({
        path: ["entries", "closed", "color"],
        value: { type: "fixed", value: "#ff0000" },
      }),
    );
  });
});
