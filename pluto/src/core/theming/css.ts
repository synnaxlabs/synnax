// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS, createHexOpacityVariants, unitProperty } from "@/core/css";
import { Theme } from "@/core/theming/theme";

const OPACITIES: readonly number[] = [80, 60, 40, 20];

export const convertThemeToCSSVars = (
  theme: Theme
): Record<string, number | string | undefined> =>
  Object.entries({
    "theme-name": theme.name,
    "theme-key": theme.key,
    "primary-m1": theme.colors.primary.m1.hex,
    "primary-z": theme.colors.primary.z.hex,
    "primary-p1": theme.colors.primary.p1.hex,
    ...createHexOpacityVariants("primary-z", theme.colors.primary.z, OPACITIES),
    "gray-m3": theme.colors.gray.m3.hex,
    "gray-m2": theme.colors.gray.m2.hex,
    "gray-m1": theme.colors.gray.m1.hex,
    "gray-m0": theme.colors.gray.m0.hex,
    ...createHexOpacityVariants("gray-m0", theme.colors.gray.m0, OPACITIES),
    "gray-p0": theme.colors.gray.p0.hex,
    ...createHexOpacityVariants("gray-p0", theme.colors.gray.p0, OPACITIES),
    "gray-p1": theme.colors.gray.p1.hex,
    "gray-p2": theme.colors.gray.p2.hex,
    "gray-p3": theme.colors.gray.p3.hex,
    "logo-color": theme.colors.logo,
    "error-m1": theme.colors.error.m1.hex,
    "error-z": theme.colors.error.z.hex,
    "error-p1": theme.colors.error.p1.hex,
    white: theme.colors.white.hex,
    "white-rgb": theme.colors.white.rgbString,
    black: theme.colors.black.hex,
    "black-rgb": theme.colors.black.rgbString,
    "background-color": theme.colors.background.hex,
    "text-color": theme.colors.text.hex,
    "text-color-rgb": theme.colors.text.rgbString,
    "border-color": theme.colors.border.hex,
    "base-size": unitProperty(theme.sizes.base, "px"),
    "border-radius": unitProperty(theme.sizes.border.radius, "px"),
    "border-width": unitProperty(theme.sizes.border.width, "px"),
    "font-family": theme.typography.family,
    "h1-size": unitProperty(theme.typography.h1.size, "rem"),
    "h1-weight": theme.typography.h1.weight,
    "h1-line-height": unitProperty(theme.typography.h1.lineHeight, "rem"),
    "h2-size": unitProperty(theme.typography.h2.size, "rem"),
    "h2-weight": theme.typography.h2.weight,
    "h2-line-height": unitProperty(theme.typography.h2.lineHeight, "rem"),
    "h3-size": unitProperty(theme.typography.h3.size, "rem"),
    "h3-weight": theme.typography.h3.weight,
    "h3-line-height": unitProperty(theme.typography.h3.lineHeight, "rem"),
    "h4-size": unitProperty(theme.typography.h4.size, "rem"),
    "h4-weight": theme.typography.h4.weight,
    "h4-line-height": unitProperty(theme.typography.h4.lineHeight, "rem"),
    "h5-size": unitProperty(theme.typography.h5.size, "rem"),
    "h5-weight": theme.typography.h5.weight,
    "h5-line-height": unitProperty(theme.typography.h5.lineHeight, "rem"),
    "h5-text-transform": theme.typography.h5.textTransform,
    "p-size": unitProperty(theme.typography.p.size, "rem"),
    "p-weight": theme.typography.p.weight,
    "p-line-height": unitProperty(theme.typography.p.lineHeight, "rem"),
    "small-size": unitProperty(theme.typography.small.size, "rem"),
    "small-weight": theme.typography.small.weight,
    "small-line-height": unitProperty(theme.typography.small.lineHeight, "rem"),
  }).reduce<Record<string, number | string | undefined>>(
    (acc, [key, value]) => ({
      ...acc,
      [CSS.var(key)]: value,
    }),
    {}
  );
