// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { SIZES } from "@/component/size";
import { levelSize, SIZE_LEVELS } from "@/schematic/node/common/size";

describe("levelSize", () => {
  describe("levels the picker offers", () => {
    it("should round-trip every rung back to itself", () =>
      SIZES.forEach((size) => expect(levelSize(SIZE_LEVELS[size])).toBe(size)));

    it("should give every rung a distinct level", () =>
      expect(new Set(SIZES.map((size) => SIZE_LEVELS[size])).size).toBe(SIZES.length));
  });

  describe("levels the picker does not offer", () => {
    it("should fall back to medium for h1", () =>
      expect(levelSize("h1")).toBe("medium"));

    // Legacy string displays store p, and medium is the height they already rendered at.
    it("should fall back to medium for p", () => expect(levelSize("p")).toBe("medium"));
  });
});
