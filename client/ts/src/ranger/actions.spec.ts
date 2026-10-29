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

    it("should throw for a path outside the name, time range, and color", () => {
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
  });

  describe("create", () => {
    it("should replace the range, dropping fields the created range lacks", () => {
      const prev = { ...createRange(), color: color.construct("#00ff00") };
      const created = { ...createRange(), name: "Created" };
      const { next } = ranger.reduceAll(prev, [ranger.create({ range: created })]);
      expect(next).toEqual(created);
    });
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
