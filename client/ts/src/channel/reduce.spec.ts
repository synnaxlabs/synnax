// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type Action, set, type Stored } from "@/channel/actions";
import { isCalculated } from "@/channel/payload";
import { createOf, reduceAll } from "@/channel/reduce";

const create = (stored: Partial<Stored>): Action => ({
  type: "create",
  create: {
    channel: {
      name: "created",
      leaseholder: 1,
      dataType: "float32",
      isIndex: false,
      localKey: 1,
      localIndex: 0,
      virtual: true,
      concurrency: control.Concurrency.exclusive,
      expression: "",
      ...stored,
    },
  },
});

describe("channel reduce", () => {
  describe("createOf", () => {
    it("should not mark a created channel calculated by its expression", () => {
      const created = createOf(create({ expression: "return 1" }));
      expect(created != null && isCalculated(created)).toBe(false);
    });
  });

  describe("reduceAll", () => {
    const created = createOf(create({}));
    if (created == null) throw new Error("expected a created channel");
    const setExpression = (value: string): Action =>
      set({ path: ["expression"], value });
    const setCalculated = (value: boolean): Action =>
      set({ path: ["calculated"], value });

    it("should mark a channel calculated when the Core sets calculated", () => {
      const { next, targets } = reduceAll(created, [setCalculated(true)]);
      expect(isCalculated(next)).toBe(true);
      expect(targets).toContain("calculated");
    });

    it("should not mark a channel calculated when only its expression arrives", () => {
      const { next } = reduceAll(created, [setExpression("return 1")]);
      expect(next.expression).toBe("return 1");
      expect(isCalculated(next)).toBe(false);
    });

    it("should keep the calculation fields of a cached channel at its create", () => {
      const cached = { ...created, expression: "return 1", calculated: true };
      const { next } = reduceAll(cached, [create({})]);
      expect(next.expression).toBe("return 1");
      expect(isCalculated(next)).toBe(true);
    });

    it("should keep calculated with no expression through operations set", () => {
      const hidden = { ...created, calculated: true };
      const operations = set({
        path: ["operations"],
        value: [{ type: "max", reset_channel: 0, duration: "1000000000" }],
      });
      const { next } = reduceAll(hidden, [operations]);
      expect(next.calculated).toBe(true);
    });

    it("should keep calculated with no expression through sets and a create", () => {
      const hidden = { ...created, calculated: true };
      const group = "3f9a3a5e-0a4b-4c64-9d8e-1f2a3b4c5d6e";
      const { next: edited } = reduceAll(hidden, [
        set({ path: ["group"], value: group }),
        set({ path: ["data_type"], value: "float64" }),
      ]);
      expect(String(edited.dataType)).toBe("float64");
      expect(edited.group).toBe(group);
      expect(isCalculated(edited)).toBe(true);
      const { next } = reduceAll(edited, [create({})]);
      expect(next.expression).toBe("");
      expect(isCalculated(next)).toBe(true);
    });

    it("should keep calculated when only the expression is cleared", () => {
      const cached = { ...created, expression: "return 1", calculated: true };
      const { next } = reduceAll(cached, [setExpression("")]);
      expect(isCalculated(next)).toBe(true);
    });

    it("should unmark a channel when the Core clears calculated", () => {
      const cached = { ...created, expression: "return 1", calculated: true };
      const { next } = reduceAll(cached, [setCalculated(false), setExpression("")]);
      expect(isCalculated(next)).toBe(false);
    });
  });
});
