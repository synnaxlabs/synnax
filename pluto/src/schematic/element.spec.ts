// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { colorFallback } from "@/schematic/element";

const THEME: theme.Theme = theme.themeZ.parse(theme.SYNNAX_DARK);

describe("colorFallback", () => {
  it("should return the fallback the variant's spec declares", () => {
    expect(colorFallback("strokeColor", "gauge", THEME)).toEqual(
      THEME.colors.visualization.palettes.default[0],
    );
    expect(colorFallback("textColor", "tank", THEME)).toEqual(THEME.colors.gray.l10);
    expect(colorFallback("fillColor", "button", THEME)).toEqual(THEME.colors.primary.z);
  });

  it("should return no fill for a variant that declares no fill fallback", () => {
    expect(colorFallback("fillColor", "tank", THEME)).toEqual(color.ZERO);
  });

  it("should return the default for a stroke or text the variant does not override", () => {
    expect(colorFallback("strokeColor", "valve", THEME)).toEqual(THEME.colors.gray.l11);
    expect(colorFallback("textColor", "value", THEME)).toEqual(THEME.colors.gray.l11);
  });

  it("should resolve the fallbacks of edge variants", () => {
    expect(colorFallback("strokeColor", "pipe", THEME)).toEqual(THEME.colors.gray.l11);
  });
});
