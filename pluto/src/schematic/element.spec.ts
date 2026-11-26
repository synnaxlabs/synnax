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

import { Schematic } from "@/schematic";

const THEME: theme.Theme = theme.themeZ.parse(theme.SYNNAX_DARK);
const SERIES = [color.construct("#dc136c"), color.construct("#20a4f3")];

describe("colorFallback", () => {
  it("should return the fallback the variant's spec declares", () => {
    expect(Schematic.colorFallback("strokeColor", "gauge", THEME, SERIES)).toEqual(
      SERIES[0],
    );
    expect(Schematic.colorFallback("textColor", "tank", THEME, SERIES)).toEqual(
      THEME.colors.gray.l10,
    );
    expect(Schematic.colorFallback("fillColor", "button", THEME, SERIES)).toEqual(
      THEME.colors.primary.z,
    );
  });

  it("should return no fill for a variant that declares no fill fallback", () => {
    expect(Schematic.colorFallback("fillColor", "tank", THEME, SERIES)).toEqual(
      color.TRANSPARENT,
    );
  });

  it("should return the default for a stroke or text the variant does not override", () => {
    expect(Schematic.colorFallback("strokeColor", "polygon", THEME, SERIES)).toEqual(
      THEME.colors.gray.l11,
    );
    expect(Schematic.colorFallback("textColor", "value", THEME, SERIES)).toEqual(
      THEME.colors.gray.l11,
    );
  });

  it("should paint a gauge transparent while the series colors load", () => {
    expect(Schematic.colorFallback("strokeColor", "gauge", THEME, [])).toEqual(
      color.TRANSPARENT,
    );
  });

  it("should resolve the fallbacks of edge variants", () => {
    expect(Schematic.colorFallback("strokeColor", "pipe", THEME, SERIES)).toEqual(
      THEME.colors.gray.l11,
    );
  });
});
