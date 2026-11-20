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

describe("lessThan", () => {
  const schema = z
    .object({ min: z.number(), max: z.number() })
    .check(zod.lessThan("min", "max", "min must be less than max"));

  it("should pass a value less than the other", () => {
    expect(schema.safeParse({ min: 0, max: 1 }).success).toBe(true);
  });

  it.each([
    ["equal to", 1],
    ["greater than", 2],
  ])("should report a value %s the other at its key", (_, min) => {
    const issues = schema.safeParse({ min, max: 1 }).error?.issues;
    expect(issues).toEqual([
      expect.objectContaining({ path: ["min"], message: "min must be less than max" }),
    ]);
  });
});
