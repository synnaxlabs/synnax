// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { describe, expect, it } from "vitest";

import { size as nodeSize } from "@/schematic/node/common/size";

describe("size.BY_LEVEL", () => {
  describe("levels the picker offers", () => {
    it("should round-trip every rung back to itself", () =>
      Component.SIZES.forEach((size) =>
        expect(nodeSize.BY_LEVEL[nodeSize.LEVELS[size]]).toBe(size),
      ));

    it("should give every rung a distinct level", () =>
      expect(new Set(Component.SIZES.map((size) => nodeSize.LEVELS[size])).size).toBe(
        Component.SIZES.length,
      ));
  });

  describe("levels the picker does not offer", () => {
    it("should fall back to medium for h1", () =>
      expect(nodeSize.BY_LEVEL.h1).toBe("medium"));

    // Legacy string displays store p, and medium is the height they already rendered
    // at.
    it("should fall back to medium for p", () =>
      expect(nodeSize.BY_LEVEL.p).toBe("medium"));
  });
});
