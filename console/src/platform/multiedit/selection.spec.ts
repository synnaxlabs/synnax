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
import { z } from "zod";

import { MultiEdit } from "@/platform/multiedit";

const SCHEMAS = {
  gauge: z.object({
    variant: z.literal("gauge"),
    fillColor: color.colorZ.optional(),
    precision: z.number(),
  }),
  label: z.object({ variant: z.literal("label"), text: z.string() }),
};

type Config = z.infer<(typeof SCHEMAS)[keyof typeof SCHEMAS]>;

const FIELDS = MultiEdit.fieldsByVariant(SCHEMAS);

const RED = color.construct("#ff0000");
const BLUE = color.construct("#0000ff");

const create = (configs: Array<[string, Config]>) => {
  const updates: Array<Array<[string, Config]>> = [];
  const selection = MultiEdit.selection<Config>({
    configs: new Map(configs),
    fields: FIELDS,
    onChange: (u) => updates.push(u),
  });
  return { selection, updates };
};

describe("selection", () => {
  const { selection, updates } = create([
    ["a", { variant: "gauge", fillColor: RED, precision: 2 }],
    ["b", { variant: "label", text: "hi" }],
    ["c", { variant: "gauge", precision: 4 }],
  ]);

  it("should report the fields some element declares", () => {
    expect(selection.has("precision")).toBe(true);
    expect(selection.has("text")).toBe(true);
  });

  it("should read the first element that declares a field", () => {
    expect(selection.first("precision")).toBe(2);
    expect(selection.first("text")).toBe("hi");
  });

  it("should read the color of each element that declares a field", () => {
    expect(selection.colors("fillColor")).toEqual([RED, undefined]);
  });

  it("should set a field only on elements that declare it", () => {
    updates.length = 0;
    selection.set("precision", 3);
    expect(updates).toEqual([
      [
        ["a", { variant: "gauge", fillColor: RED, precision: 3 }],
        ["c", { variant: "gauge", precision: 3 }],
      ],
    ]);
  });

  it("should remove a field set to undefined", () => {
    updates.length = 0;
    selection.set("fillColor", undefined);
    expect(updates[0][0]).toEqual(["a", { variant: "gauge", precision: 2 }]);
  });

  it("should update a field from each element's own value", () => {
    updates.length = 0;
    selection.update("precision", (p) => (p ?? 0) + 1);
    expect(updates[0].map(([, c]) => c)).toEqual([
      { variant: "gauge", fillColor: RED, precision: 3 },
      { variant: "gauge", precision: 5 },
    ]);
  });

  it("should set every referenced color", () => {
    updates.length = 0;
    selection.setColors([{ key: "a", path: "fillColor", value: RED }], BLUE);
    expect(updates).toEqual([
      [["a", { variant: "gauge", fillColor: BLUE, precision: 2 }]],
    ]);
  });

  it("should not call onChange when no element declares the field", () => {
    const { selection: labels, updates: none } = create([
      ["b", { variant: "label", text: "hi" }],
    ]);
    labels.set("precision", 3);
    expect(none).toEqual([]);
  });

  it("should throw when a reference names an element outside the selection", () => {
    expect(() =>
      selection.setColors([{ key: "z", path: "fillColor", value: RED }], BLUE),
    ).toThrow("[multiedit] - no config for z");
  });

  it("should throw when a variant has no schema", () => {
    const { selection: unknown } = create([
      ["x", { variant: "dial", precision: 1 } as unknown as Config],
    ]);
    expect(() => unknown.has("precision")).toThrow("[multiedit] - no schema for dial");
  });
});
