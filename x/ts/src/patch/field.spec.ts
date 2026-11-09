// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { patch } from "@/patch";

const settingsZ = z.object({ rate: z.number(), stream: z.number().default(0) });
const settingsUpdateZ = z.object({
  rate: patch.changeZ(z.number()).optional(),
  stream: patch.changeZ(z.number()).optional(),
});
const updateZ = z.object({
  name: patch.changeZ(z.string()).optional(),
  group: patch.changeZ(z.string()).optional(),
  settings: patch.structZ({ u: settingsUpdateZ, t: settingsZ }).optional(),
  extra: patch.changeZ(patch.structZ({ u: settingsUpdateZ, t: settingsZ })).optional(),
  tags: patch.valueListZ(z.string()).optional(),
  channels: patch
    .listZ({
      e: z.object({ key: z.string(), port: z.number() }),
      k: z.string(),
      u: z.object({ key: z.string(), port: patch.changeZ(z.number()).optional() }),
    })
    .optional(),
});

describe("update fields", () => {
  describe("changeZ", () => {
    it("should accept a value or null", () => {
      expect(updateZ.parse({ name: "a" })).toEqual({ name: "a" });
      expect(updateZ.parse({ group: null })).toEqual({ group: null });
    });
  });

  describe("structZ", () => {
    it("should accept a merge or a replace", () => {
      expect(updateZ.parse({ settings: { rate: 5 } })).toEqual({
        settings: { rate: 5 },
      });
      expect(updateZ.parse({ settings: { set: { rate: 5 } } })).toEqual({
        settings: { set: { rate: 5, stream: 0 } },
      });
    });

    it("should reject null on a required struct", () => {
      expect(() => updateZ.parse({ settings: null })).toThrow();
    });

    it("should reject set beside another member", () => {
      expect(() =>
        updateZ.parse({ settings: { set: { rate: 1 }, rate: 2 } }),
      ).toThrow();
    });

    it("should reject a set that does not parse instead of merging nothing", () => {
      expect(() => updateZ.parse({ settings: { set: { rate: "fast" } } })).toThrow();
    });
  });

  describe("an optional struct", () => {
    it("should make no change when absent", () => {
      expect(updateZ.parse({})).toEqual({});
    });

    it("should merge a partial", () => {
      expect(updateZ.parse({ extra: { stream: 3 } })).toEqual({ extra: { stream: 3 } });
    });

    it("should replace the whole value", () => {
      expect(updateZ.parse({ extra: { set: { rate: 4 } } })).toEqual({
        extra: { set: { rate: 4, stream: 0 } },
      });
    });

    it("should clear on null", () => {
      expect(updateZ.parse({ extra: null })).toEqual({ extra: null });
    });
  });

  describe("listZ and valueListZ", () => {
    it("should accept operators", () => {
      const update = {
        tags: { set: [] },
        channels: {
          remove: ["c1"],
          update: [{ key: "c2", port: 4 }],
          move: [{ key: "c2" }],
        },
      };
      expect(updateZ.parse(update)).toEqual(update);
    });

    it("should reject a plain array", () => {
      expect(() => updateZ.parse({ tags: ["a"] })).toThrow();
      // @ts-expect-error - a list field takes operators, never a plain array.
      const tags: patch.ValueList<z.ZodString> = ["a"];
      expect(tags).toEqual(["a"]);
    });

    it("should reject null for an operator", () => {
      expect(() => updateZ.parse({ tags: { set: null } })).toThrow();
    });

    it("should reject an unknown operator", () => {
      expect(() => updateZ.parse({ tags: { append: ["a"] } })).toThrow();
    });
  });
});
