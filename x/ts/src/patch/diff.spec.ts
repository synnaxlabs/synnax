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
import { DataType } from "@/telem";

const schema = z.discriminatedUnion("variant", [
  z.object({
    variant: z.literal("valve"),
    color: z.string().optional(),
    size: z.number(),
    label: z.object({ text: z.string(), color: z.string() }).optional(),
    segments: z.array(z.object({ length: z.number() })).optional(),
  }),
  z.object({ variant: z.literal("tank"), size: z.number() }),
]);

const GUARDS = [{ path: ["variant"], value: "valve" }];

describe("diff", () => {
  it("should list no action when nothing differs", () => {
    const value = { variant: "valve", size: 1, label: { text: "a", color: "b" } };
    expect(patch.diff(schema, value, structuredClone(value))).toEqual([]);
  });

  it("should list no action for a structured clone of a data type", () => {
    const typed = z.object({ dataType: DataType.z });
    const base = { dataType: DataType.FLOAT64 };
    expect(patch.diff(typed, base, structuredClone(base))).toEqual([]);
  });

  it("should set only the fields that differ", () => {
    const base = { variant: "valve", size: 1, color: "red" };
    expect(patch.diff(schema, base, { ...base, size: 2 })).toEqual([
      { type: "set", set: { path: ["size"], value: 2, guards: GUARDS } },
    ]);
  });

  it("should reach into a nested object", () => {
    const base = { variant: "valve", size: 1, label: { text: "a", color: "b" } };
    const next = { ...base, label: { ...base.label, text: "c" } };
    expect(patch.diff(schema, base, next)).toEqual([
      { type: "set", set: { path: ["label", "text"], value: "c", guards: GUARDS } },
    ]);
  });

  it("should set a field that base does not hold", () => {
    const base = { variant: "valve", size: 1 };
    expect(patch.diff(schema, base, { ...base, color: "red" })).toEqual([
      { type: "set", set: { path: ["color"], value: "red", guards: GUARDS } },
    ]);
  });

  it("should remove a field that next does not hold", () => {
    const base = { variant: "valve", size: 1, color: "red" };
    const expected = [
      { type: "remove", remove: { path: [], key: "color", guards: GUARDS } },
    ];
    expect(patch.diff(schema, base, { variant: "valve", size: 1 })).toEqual(expected);
    expect(patch.diff(schema, base, { ...base, color: undefined })).toEqual(expected);
  });

  it("should set a list whole", () => {
    const base = { variant: "valve", size: 1, segments: [{ length: 1 }] };
    const segments = [{ length: 1 }, { length: 2 }];
    expect(patch.diff(schema, base, { ...base, segments })).toEqual([
      { type: "set", set: { path: ["segments"], value: segments, guards: GUARDS } },
    ]);
  });

  it("should set an object whole when its union changes member", () => {
    const base = { variant: "valve", size: 1, color: "red" };
    const next = { variant: "tank", size: 2 };
    expect(patch.diff(schema, base, next)).toEqual([
      { type: "set", set: { path: [], value: next, guards: [] } },
    ]);
  });

  it("should throw when a path that differs is outside the schema", () => {
    const base = { variant: "valve", size: 1 };
    expect(() => patch.diff(schema, base, { ...base, weight: 2 })).toThrow(
      "Schema does not contain the path weight",
    );
  });
});
