// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  CUSTOM_VARIANTS,
  isCustomVariant,
  REGISTRY,
  STATIC_SPECS,
} from "@/schematic/node/registry";

describe("Schematic.Node.STATIC_SPECS", () => {
  it("should not include custom variants", () => {
    const keys = STATIC_SPECS.map((s) => s.key);
    for (const variant of CUSTOM_VARIANTS) expect(keys).not.toContain(variant);
  });

  it("should include all non-custom registry entries", () => {
    const allKeys = Object.keys(REGISTRY);
    const expectedKeys = allKeys.filter((k) => !isCustomVariant(k));
    const actualKeys = STATIC_SPECS.map((s) => s.key);
    expect(actualKeys).toEqual(expect.arrayContaining(expectedKeys));
    expect(actualKeys).toHaveLength(expectedKeys.length);
  });

  it("custom variants should exist in the registry", () => {
    for (const variant of CUSTOM_VARIANTS) expect(REGISTRY).toHaveProperty(variant);
  });
});
