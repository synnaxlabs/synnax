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

import { resolveColor } from "@/schematic/node/common/color";
import { Theming } from "@/theming";

describe("resolveColor", () => {
  const light = Theming.themeZ.parse(Theming.SYNNAX_LIGHT);
  const dark = Theming.themeZ.parse(Theming.SYNNAX_DARK);

  it("should resolve the ZERO follow-theme sentinel to the theme's default color", () => {
    expect(resolveColor(color.ZERO, light)).toEqual(light.colors.gray.l11);
    expect(resolveColor(color.ZERO, dark)).toEqual(dark.colors.gray.l11);
  });

  it("should resolve an unset color to the theme's default color", () => {
    expect(resolveColor(undefined, light)).toEqual(light.colors.gray.l11);
    expect(resolveColor(undefined, dark)).toEqual(dark.colors.gray.l11);
  });

  it("should re-color a default symbol when the theme changes", () => {
    expect(resolveColor(color.ZERO, light)).not.toEqual(resolveColor(color.ZERO, dark));
  });

  it("should honor an explicit color regardless of theme", () => {
    expect(resolveColor("#ff0000", light)).toEqual("#ff0000");
    expect(resolveColor("#ff0000", dark)).toEqual("#ff0000");
  });
});
