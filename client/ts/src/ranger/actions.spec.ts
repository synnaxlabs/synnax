// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ranger } from "@/ranger";

const KEY = "5a7d3e1c-2f4b-4c6d-8e9f-0a1b2c3d4e5f";
const PARENT = "6b8e4f2d-3a5c-4d7e-9fa0-1b2c3d4e5f6a";
const LABEL = "7c9f5a3e-4b6d-4e8f-a0b1-2c3d4e5f6a7b";

const createRange = (): ranger.Payload =>
  ranger.payloadZ.parse({
    key: KEY,
    name: "Hot fire",
    timeRange: new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(5)),
  });

describe("ranger.reduceAll", () => {
  describe("set", () => {
    it("should set the name", () => {
      const { next } = ranger.reduceAll(createRange(), [
        ranger.set({ path: ["name"], value: "Cold flow" }),
      ]);
      expect(next.name).toBe("Cold flow");
    });

    it("should set the end of the time range from its wire form", () => {
      const { next } = ranger.reduceAll(createRange(), [
        ranger.set({ path: ["time_range", "end"], value: "9000000000" }),
      ]);
      expect(next.timeRange).toEqual(
        new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(9)),
      );
      expect(next.timeRange).toBeInstanceOf(TimeRange);
    });

    it("should set a color the range does not have", () => {
      const { next } = ranger.reduceAll(createRange(), [
        ranger.set({ path: ["color"], value: "#ff0000" }),
      ]);
      expect(next.color).toEqual(color.construct("#ff0000"));
    });

    it("should remove the color when a set holds null", () => {
      const prev = { ...createRange(), color: color.construct("#00ff00") };
      const { next } = ranger.reduceAll(prev, [
        ranger.set({ path: ["color"], value: null }),
      ]);
      expect(next).not.toHaveProperty("color");
    });

    it("should keep the range unchanged by identity when nothing changes", () => {
      const prev = createRange();
      const { next } = ranger.reduceAll(prev, [
        ranger.set({
          path: ["name"],
          value: "Cold flow",
          guards: [{ path: ["name"], value: "Other" }],
        }),
      ]);
      expect(next).toBe(prev);
    });

    it("should keep the identity of the time range when the name changes", () => {
      const prev = createRange();
      const { next } = ranger.reduceAll(prev, [
        ranger.set({ path: ["name"], value: "Cold flow" }),
      ]);
      expect(next.timeRange).toBe(prev.timeRange);
    });

    it("should return a wire-form inverse that restores the range", () => {
      const prev = createRange();
      const { next, inverse, targets } = ranger.reduceAll(prev, [
        ranger.set({ path: ["time_range", "end"], value: "9000000000" }),
      ]);
      expect(targets).toEqual(["time_range.end"]);
      expect(ranger.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should hold a time range that ends before it starts", () => {
      const { next } = ranger.reduceAll(createRange(), [
        ranger.set({ path: ["time_range", "end"], value: "0" }),
      ]);
      expect(next.timeRange.end).toEqual(new TimeStamp(0));
    });

    it("should set the labels and return an inverse that restores them", () => {
      const prev = createRange();
      const { next, inverse } = ranger.reduceAll(prev, [
        ranger.set({ path: ["labels"], value: [LABEL] }),
      ]);
      expect(next.labels).toEqual([LABEL]);
      expect(ranger.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should set a parent the range does not have", () => {
      const prev = createRange();
      const { next, inverse } = ranger.reduceAll(prev, [
        ranger.set({ path: ["parent"], value: PARENT }),
      ]);
      expect(next.parent).toBe(PARENT);
      expect(ranger.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should throw for a path outside the fields that path actions edit", () => {
      expect(() =>
        ranger.reduceAll(createRange(), [ranger.set({ path: ["key"], value: "k" })]),
      ).toThrow(/key/);
    });
  });

  describe("remove", () => {
    it("should remove the color", () => {
      const prev = { ...createRange(), color: color.construct("#00ff00") };
      const { next, inverse } = ranger.reduceAll(prev, [
        ranger.remove({ path: [], key: "color" }),
      ]);
      expect(next).not.toHaveProperty("color");
      expect(ranger.reduceAll(next, inverse).next).toEqual(prev);
    });

    it("should remove the parent", () => {
      const prev = { ...createRange(), parent: PARENT };
      const { next, inverse } = ranger.reduceAll(prev, [
        ranger.remove({ path: [], key: "parent" }),
      ]);
      expect(next).not.toHaveProperty("parent");
      expect(ranger.reduceAll(next, inverse).next).toEqual(prev);
    });
  });

  describe("create", () => {
    it("should replace the range, dropping fields the created range lacks", () => {
      const prev = { ...createRange(), color: color.construct("#00ff00") };
      const created = { ...createRange(), name: "Created" };
      const { next } = ranger.reduceAll(prev, [ranger.create({ range: created })]);
      expect(next).toEqual(created);
    });

    it("should keep the labels so a late create echo cannot undo a label write", () => {
      const prev = { ...createRange(), labels: [LABEL] };
      const created = { ...createRange(), name: "Created" };
      const { next } = ranger.reduceAll(prev, [ranger.create({ range: created })]);
      expect(next).toEqual({ ...created, labels: [LABEL] });
    });

    it("should keep hasChildren, which a create carries as false", () => {
      const prev = { ...createRange(), hasChildren: true };
      const { next } = ranger.reduceAll(prev, [
        ranger.create({ range: createRange() }),
      ]);
      expect(next.hasChildren).toBe(true);
    });
  });
});

describe("ranger.encode", () => {
  const end = (action: ranger.Action): unknown =>
    action.type === "set" ? action.set.value : undefined;

  it("should send a time range that never ends with its exact end", () => {
    const action = ranger.set({
      path: ["timeRange"],
      value: { start: 1, end: TimeStamp.MAX.nanoseconds },
    });
    expect(end(ranger.encode(createRange(), action))).toEqual({
      start: "1",
      end: TimeStamp.MAX.valueOf().toString(),
    });
  });

  it("should send one bound of a time range with its exact value", () => {
    const action = ranger.set({
      path: ["timeRange", "end"],
      value: TimeStamp.MAX.nanoseconds,
    });
    expect(end(ranger.encode(createRange(), action))).toBe(
      TimeStamp.MAX.valueOf().toString(),
    );
  });

  it("should keep other fields as they are", () => {
    const action = ranger.set({ path: ["name"], value: "Cold flow" });
    expect(end(ranger.encode(createRange(), action))).toBe("Cold flow");
  });
});

describe("ranger.createOf", () => {
  it("should return the range that a create carries", () => {
    const range = createRange();
    expect(ranger.createOf(ranger.create({ range }))).toEqual(range);
  });

  it("should return undefined for a path action", () => {
    expect(ranger.createOf(ranger.set({ path: ["name"], value: "x" }))).toBeUndefined();
  });
});

describe("ranger.parentOf", () => {
  it("should return the parent of a created range", () => {
    const range = { ...createRange(), parent: PARENT };
    expect(ranger.parentOf(ranger.create({ range }))).toBe(PARENT);
    expect(ranger.parentOf(ranger.create({ range: createRange() }))).toBeUndefined();
  });

  it("should return the parent that a set of the parent holds", () => {
    expect(ranger.parentOf(ranger.set({ path: ["parent"], value: PARENT }))).toBe(
      PARENT,
    );
  });

  it("should return undefined for other actions", () => {
    expect(ranger.parentOf(ranger.set({ path: ["name"], value: "x" }))).toBeUndefined();
    expect(ranger.parentOf(ranger.remove({ key: "parent" }))).toBeUndefined();
    expect(
      ranger.parentOf(ranger.set({ path: ["parent"], value: null })),
    ).toBeUndefined();
  });
});
