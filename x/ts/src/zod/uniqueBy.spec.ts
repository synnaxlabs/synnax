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

import { zod } from "@/zod";

const itemZ = z.object({ name: z.string(), disabled: z.boolean() });
type Item = z.infer<typeof itemZ>;

describe("uniqueBy", () => {
  const schema = itemZ
    .array()
    .check(zod.uniqueBy<Item, "name">("name", (name) => `Duplicate name "${name}"`));

  it("should accept unique values", () => {
    expect(
      schema.safeParse([
        { name: "a", disabled: false },
        { name: "b", disabled: false },
      ]).success,
    ).toBe(true);
  });

  it("should flag each repeat at its index and key", () => {
    const result = schema.safeParse([
      { name: "a", disabled: false },
      { name: "a", disabled: false },
      { name: "b", disabled: false },
      { name: "a", disabled: false },
    ]);
    expect(
      result.error?.issues.map(({ message, path }) => ({ message, path })),
    ).toEqual([
      { message: 'Duplicate name "a"', path: [1, "name"] },
      { message: 'Duplicate name "a"', path: [3, "name"] },
    ]);
  });

  it("should not check or claim skipped items", () => {
    const skipping = itemZ.array().check(
      zod.uniqueBy<Item, "name">(
        "name",
        (name) => `Duplicate name "${name}"`,
        ({ disabled }) => disabled,
      ),
    );
    const result = skipping.safeParse([
      { name: "a", disabled: true },
      { name: "a", disabled: false },
      { name: "a", disabled: true },
      { name: "a", disabled: false },
    ]);
    expect(result.error?.issues.map(({ path }) => path)).toEqual([[3, "name"]]);
  });

  it("should prefix the path of a nested array", () => {
    const result = z.object({ items: schema }).safeParse({
      items: [
        { name: "a", disabled: false },
        { name: "a", disabled: false },
      ],
    });
    expect(result.error?.issues[0].path).toEqual(["items", 1, "name"]);
  });
});
