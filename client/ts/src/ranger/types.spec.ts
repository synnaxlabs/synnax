// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ranger } from "@/ranger";

const KEY = "4b6b2fd6-3c93-4b7f-8f43-7d1f8a0b9c2e";

describe("refZ", () => {
  it("should parse a key ref", () => {
    expect(ranger.refZ.parse({ variant: "key", key: KEY })).toEqual({
      variant: "key",
      key: KEY,
    });
  });

  it("should parse a rolling window ref", () => {
    const ref = {
      variant: "window",
      window: { variant: "dynamic", timeSpan: 60e9 },
    };
    expect(ranger.refZ.parse(ref)).toEqual(ref);
  });

  it("should keep a fixed window ref's endpoints as decimal strings", () => {
    expect(
      ranger.refZ.parse({
        variant: "window",
        window: { variant: "static", timeRange: { start: 0, end: 1000n } },
      }),
    ).toEqual({
      variant: "window",
      window: { variant: "static", timeRange: { start: "0", end: "1000" } },
    });
  });

  it.each([
    ["an unknown variant", { variant: "bogus", key: KEY }],
    ["a key ref without a key", { variant: "key" }],
    ["a key ref with a non-UUID key", { variant: "key", key: "rolling1m" }],
    ["a window ref without a window", { variant: "window" }],
  ])("should reject %s", (_, value) => {
    expect(ranger.refZ.safeParse(value).success).toBe(false);
  });
});
