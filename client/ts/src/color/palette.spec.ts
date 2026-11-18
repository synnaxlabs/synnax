// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color as xcolor } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { color } from "@/color";
import { createTestClient } from "@/testutil";

const named = (key: string, position?: string): color.Named => ({
  key,
  name: key,
  light: xcolor.construct("#000000"),
  position,
});

describe("palette", () => {
  describe("compare", () => {
    it("should order named colors by position, then by key, with no position last", () => {
      const sorted = [
        named("00000000-0000-4000-8000-000000000004"),
        named("00000000-0000-4000-8000-000000000003", "a1"),
        named("00000000-0000-4000-8000-000000000002", "a0"),
        named("00000000-0000-4000-8000-000000000001", "a1"),
      ].sort(color.compare);
      expect(sorted.map((c) => c.key.at(-1))).toEqual(["2", "1", "3", "4"]);
    });
  });

  describe("SERIES_PALETTE", () => {
    it("should name the palette that holds the built-in named colors", async () => {
      const builtins = await createTestClient().colors.retrieveBuiltins();
      expect(builtins.length).toBeGreaterThan(0);
      for (const c of builtins) expect(c.group).toBe(color.SERIES_PALETTE);
    });
  });
});
