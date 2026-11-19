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

import { theme } from "@/theme";

const LIGHT = theme.themeZ.parse(theme.SYNNAX_LIGHT);
const DARK = theme.themeZ.parse(theme.SYNNAX_DARK);

const lightness = (c: color.Color): number => color.oklch(c)[0];

describe("theme", () => {
  describe("dark", () => {
    it("should be set only on the dark theme", () => {
      expect(LIGHT.dark).toBe(false);
      expect(DARK.dark).toBe(true);
    });
  });

  describe("legible", () => {
    it("should lift a dark color to the band floor on the dark theme", () => {
      expect(lightness(theme.legible("#202060", DARK))).toBeCloseTo(DARK.symbol.lMin);
    });

    it("should cap a pale color at the band ceiling on the light theme", () => {
      expect(lightness(theme.legible("#F7AEF8", LIGHT))).toBeCloseTo(LIGHT.symbol.lMax);
    });

    it("should snap a gray to the extreme of the theme", () => {
      expect(lightness(theme.legible("#333333", DARK))).toBeCloseTo(1);
      expect(lightness(theme.legible("#CCCCCC", LIGHT))).toBeCloseTo(0);
    });

    it("should keep the alpha", () => {
      expect(color.opacity(theme.legible("#20206080", DARK))).toBeCloseTo(0.5, 2);
    });
  });

  describe("resolve", () => {
    const pair: theme.Pair = { light: "#DC136C", dark: "#20A4F3" };

    it("should return the light value on the light theme", () => {
      expect(theme.resolve(pair, LIGHT)).toEqual(color.construct("#DC136C"));
    });

    it("should return the dark value on the dark theme", () => {
      expect(theme.resolve(pair, DARK)).toEqual(color.construct("#20A4F3"));
    });

    it("should derive an absent dark value from the band", () => {
      expect(theme.resolve({ light: "#202060" }, DARK)).toEqual(
        theme.legible("#202060", DARK),
      );
    });

    it("should ignore the band on the light theme", () => {
      expect(theme.resolve({ light: "#F7AEF8" }, LIGHT)).toEqual(
        color.construct("#F7AEF8"),
      );
    });
  });
});
