// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Schematic } from "@synnaxlabs/pluto";
import { describe, expect, it } from "vitest";

import { ALL_STATIC_SYMBOLS, CUSTOM_VARIANTS } from "@/schematic/toolbar/Symbols";

describe("Symbols toolbar", () => {
  describe("ALL_STATIC_SYMBOLS", () => {
    it("should not include custom variants", () => {
      const keys = ALL_STATIC_SYMBOLS.map((s) => s.key);
      for (const variant of CUSTOM_VARIANTS) expect(keys).not.toContain(variant);
    });

    it("should include all non-custom registry entries", () => {
      const allKeys = Object.keys(Schematic.Symbol.REGISTRY);
      const expectedKeys = allKeys.filter((k) => !CUSTOM_VARIANTS.has(k));
      const actualKeys = ALL_STATIC_SYMBOLS.map((s) => s.key);
      expect(actualKeys).toEqual(expect.arrayContaining(expectedKeys));
      expect(actualKeys).toHaveLength(expectedKeys.length);
    });

    it("custom variants should exist in the registry", () => {
      for (const variant of CUSTOM_VARIANTS)
        expect(Schematic.Symbol.REGISTRY).toHaveProperty(variant);
    });
  });
});
