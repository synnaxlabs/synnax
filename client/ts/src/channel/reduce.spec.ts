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
      description: "",
      ...stored,
    },
  },
});

describe("channel reduce", () => {
  describe("createOf", () => {
    it("should mark a created channel with an expression calculated", () => {
      const created = createOf(create({ expression: "return 1" }));
      expect(created != null && isCalculated(created)).toBe(true);
    });

    it("should not mark a virtual channel with no expression calculated", () => {
      const created = createOf(create({}));
      expect(created != null && isCalculated(created)).toBe(false);
    });
  });

  describe("reduceAll", () => {
    const created = createOf(create({}));
    if (created == null) throw new Error("expected a created channel");
    const setExpression = (value: string): Action =>
      set({ path: ["expression"], value });

    it("should mark a virtual channel calculated when its expression arrives", () => {
      const { next, targets } = reduceAll(created, [setExpression("return 1")]);
      expect(isCalculated(next)).toBe(true);
      expect(targets).toContain("calculated");
    });

    it("should keep the calculation fields of a cached channel at its create", () => {
      const calculated = createOf(create({ expression: "return 1" }));
      if (calculated == null) throw new Error("expected a created channel");
      const { next } = reduceAll(calculated, [create({})]);
      expect(next.expression).toBe("return 1");
      expect(isCalculated(next)).toBe(true);
    });

    it("should keep calculated with no expression through operations set", () => {
      const hidden = { ...created, calculated: true };
      // Only a set of the expression recomputes calculated.
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

    it("should unmark a channel whose expression is cleared", () => {
      const calculated = createOf(create({ expression: "return 1" }));
      if (calculated == null) throw new Error("expected a created channel");
      const { next } = reduceAll(calculated, [setExpression("")]);
      expect(isCalculated(next)).toBe(false);
    });
  });
});
