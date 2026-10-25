// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { label } from "@/label";

const KEY = "5a7d3e1c-2f4b-4c6d-8e9f-0a1b2c3d4e5f";

const createLabel = (): label.Label =>
  label.labelZ.parse({ key: KEY, name: "Nominal", color: "#00ff00" });

describe("label.reduceAll", () => {
  describe("set", () => {
    it("should set the name and keep the color by identity", () => {
      const prev = createLabel();
      const { next } = label.reduceAll(prev, [
        label.set({ path: ["name"], value: "Anomaly" }),
      ]);
      expect(next.name).toBe("Anomaly");
      expect(next.color).toBe(prev.color);
    });

    it("should set the color from a hex string", () => {
      const { next } = label.reduceAll(createLabel(), [
        label.set({ path: ["color"], value: "#ff0000" }),
      ]);
      expect(next.color).toEqual(color.construct("#ff0000"));
    });

    it("should return an inverse that restores the label", () => {
      const prev = createLabel();
      const { next, inverse, targets } = label.reduceAll(prev, [
        label.set({ path: ["color"], value: "#ff0000" }),
      ]);
      expect(targets).toEqual(["color"]);
      expect(label.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should keep the label unchanged by identity when a guard fails", () => {
      const prev = createLabel();
      const { next } = label.reduceAll(prev, [
        label.set({
          path: ["name"],
          value: "Anomaly",
          guards: [{ path: ["name"], value: "Other" }],
        }),
      ]);
      expect(next).toBe(prev);
    });

    it("should hold an empty name", () => {
      const { next } = label.reduceAll(createLabel(), [
        label.set({ path: ["name"], value: "" }),
      ]);
      expect(next.name).toBe("");
    });

    it("should throw for a path outside the name and color", () => {
      expect(() =>
        label.reduceAll(createLabel(), [label.set({ path: ["key"], value: KEY })]),
      ).toThrow(/key/);
    });

    it("should throw for a value the color cannot hold", () => {
      expect(() =>
        label.reduceAll(createLabel(), [
          label.set({ path: ["color"], value: "not a color" }),
        ]),
      ).toThrow(/color/);
    });
  });

  describe("create", () => {
    it("should replace the label", () => {
      const created = { ...createLabel(), name: "Created" };
      const { next, inverse } = label.reduceAll(createLabel(), [
        label.create({ label: created }),
      ]);
      expect(next).toEqual(created);
      expect(inverse).toEqual([]);
    });
  });
});

describe("label.createOf", () => {
  it("should return the label that a create carries", () => {
    const lbl = createLabel();
    expect(label.createOf(label.create({ label: lbl }))).toEqual(lbl);
  });

  it("should return undefined for a path action", () => {
    expect(label.createOf(label.set({ path: ["name"], value: "x" }))).toBeUndefined();
  });
});
