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

import { color } from "@/color";

const KEY = "5a7d3e1c-2f4b-4c6d-8e9f-0a1b2c3d4e5f";
const GROUP = "6b8e4f2d-3a5c-4d7e-9fa0-1b2c3d4e5f60";

const createNamed = (): color.Named =>
  color.namedZ.parse({ key: KEY, name: "Nominal", light: "#00ff00" });

describe("color.reduceAll", () => {
  describe("set", () => {
    it("should set the name and keep the light value by identity", () => {
      const prev = createNamed();
      const { next } = color.reduceAll(prev, [
        color.set({ path: ["name"], value: "Anomaly" }),
      ]);
      expect(next.name).toBe("Anomaly");
      expect(next.light).toBe(prev.light);
    });

    it("should set the light value from a hex string", () => {
      const { next } = color.reduceAll(createNamed(), [
        color.set({ path: ["light"], value: "#ff0000" }),
      ]);
      expect(next.light).toEqual(xcolor.construct("#ff0000"));
    });

    it("should set a dark value on a named color that has none", () => {
      const { next } = color.reduceAll(createNamed(), [
        color.set({ path: ["dark"], value: "#0000ff" }),
      ]);
      expect(next.dark).toEqual(xcolor.construct("#0000ff"));
    });

    it("should set and clear the group", () => {
      const { next } = color.reduceAll(createNamed(), [
        color.set({ path: ["group"], value: GROUP }),
      ]);
      expect(next.group).toBe(GROUP);
      const cleared = color.reduceAll(next, [
        color.set({ path: ["group"], value: null }),
      ]);
      expect(cleared.next.group).toBeUndefined();
    });

    it("should set the position", () => {
      const { next } = color.reduceAll(createNamed(), [
        color.set({ path: ["position"], value: "a0V" }),
      ]);
      expect(next.position).toBe("a0V");
    });

    it("should return an inverse that restores the named color", () => {
      const prev = createNamed();
      const { next, inverse, targets } = color.reduceAll(prev, [
        color.set({ path: ["dark"], value: "#ff0000" }),
      ]);
      expect(targets).toEqual(["dark"]);
      expect(color.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should keep the named color unchanged by identity when a guard fails", () => {
      const prev = createNamed();
      const { next } = color.reduceAll(prev, [
        color.set({
          path: ["name"],
          value: "Anomaly",
          guards: [{ path: ["name"], value: "Other" }],
        }),
      ]);
      expect(next).toBe(prev);
    });

    it("should hold an empty name", () => {
      const { next } = color.reduceAll(createNamed(), [
        color.set({ path: ["name"], value: "" }),
      ]);
      expect(next.name).toBe("");
    });

    it("should throw for a path outside the editable fields", () => {
      expect(() =>
        color.reduceAll(createNamed(), [color.set({ path: ["key"], value: KEY })]),
      ).toThrow(/key/);
    });

    it("should throw for a value the light value cannot hold", () => {
      expect(() =>
        color.reduceAll(createNamed(), [
          color.set({ path: ["light"], value: "not a color" }),
        ]),
      ).toThrow(/light/);
    });
  });

  describe("remove", () => {
    it("should remove the position", () => {
      const prev = { ...createNamed(), position: "a0" };
      const { next } = color.reduceAll(prev, [
        color.remove({ path: [], key: "position" }),
      ]);
      expect(next.position).toBeUndefined();
    });

    it("should remove the dark value", () => {
      const prev = color.namedZ.parse({ ...createNamed(), dark: "#0000ff" });
      const { next, inverse } = color.reduceAll(prev, [
        color.remove({ path: [], key: "dark" }),
      ]);
      expect(next.dark).toBeUndefined();
      expect(color.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should throw for a removal of the light value", () => {
      expect(() =>
        color.reduceAll(createNamed(), [color.remove({ path: [], key: "light" })]),
      ).toThrow(/light/);
    });
  });

  describe("create", () => {
    it("should replace the named color", () => {
      const created = { ...createNamed(), name: "Created" };
      const { next, inverse } = color.reduceAll(createNamed(), [
        color.create({ named: created }),
      ]);
      expect(next).toEqual(created);
      expect(inverse).toEqual([]);
    });
  });
});

describe("color.createOf", () => {
  it("should return the named color that a create carries", () => {
    const named = createNamed();
    expect(color.createOf(color.create({ named }))).toEqual(named);
  });

  it("should return undefined for a path action", () => {
    expect(color.createOf(color.set({ path: ["name"], value: "x" }))).toBeUndefined();
  });
});
