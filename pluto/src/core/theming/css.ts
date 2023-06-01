// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Color, Size, Theme } from "./theme";

import { createHexOpacityVariants, unitProperty } from "@/core/css";

const OPACITIES: readonly number[] = [80, 60, 40, 20];

export const convertThemeToCSSVars = (
  theme: Theme
): Record<string, Size | Color | undefined> => ({
  "--pluto-theme-name": theme.name,
  "--pluto-theme-key": theme.key,
  "--pluto-primary-m1": theme.colors.primary.m1,
  "--pluto-primary-z": theme.colors.primary.z,
  "--pluto-primary-p1": theme.colors.primary.p1,
  ...createHexOpacityVariants("--pluto-primary-z", theme.colors.primary.z, OPACITIES),
  "--pluto-gray-m3": theme.colors.gray.m3,
  "--pluto-gray-m2": theme.colors.gray.m2,
  "--pluto-gray-m1": theme.colors.gray.m1,
  "--pluto-gray-m0": theme.colors.gray.m0,
  ...createHexOpacityVariants("--pluto-gray-m0", theme.colors.gray.m0, OPACITIES),
  "--pluto-gray-p0": theme.colors.gray.p0,
  ...createHexOpacityVariants("--pluto-gray-p0", theme.colors.gray.p0, OPACITIES),
  "--pluto-gray-p1": theme.colors.gray.p1,
  "--pluto-gray-p2": theme.colors.gray.p2,
  "--pluto-gray-p3": theme.colors.gray.p3,
  "--pluto-logo-color": theme.colors.logo,
  "--pluto-error-m1": theme.colors.error.m1,
  "--pluto-error-z": theme.colors.error.z,
  "--pluto-error-p1": theme.colors.error.p1,
  "--pluto-white": theme.colors.white,
  "--pluto-black": theme.colors.black,
  "--pluto-background-color": theme.colors.background,
  "--pluto-text-color": theme.colors.text,
  "--pluto-border-color": theme.colors.border,
  "--pluto-base-size": unitProperty(theme.sizes.base, "px"),
  "--pluto-border-radius": unitProperty(theme.sizes.border.radius, "px"),
  "--pluto-border-width": unitProperty(theme.sizes.border.width, "px"),
  "--pluto-font-family": theme.typography.family,
  "--pluto-h1-size": unitProperty(theme.typography.h1.size, "rem"),
  "--pluto-h1-weight": theme.typography.h1.weight,
  "--pluto-h1-line-height": unitProperty(theme.typography.h1.lineHeight, "rem"),
  "--pluto-h2-size": unitProperty(theme.typography.h2.size, "rem"),
  "--pluto-h2-weight": theme.typography.h2.weight,
  "--pluto-h2-line-height": unitProperty(theme.typography.h2.lineHeight, "rem"),
  "--pluto-h3-size": unitProperty(theme.typography.h3.size, "rem"),
  "--pluto-h3-weight": theme.typography.h3.weight,
  "--pluto-h3-line-height": unitProperty(theme.typography.h3.lineHeight, "rem"),
  "--pluto-h4-size": unitProperty(theme.typography.h4.size, "rem"),
  "--pluto-h4-weight": theme.typography.h4.weight,
  "--pluto-h4-line-height": unitProperty(theme.typography.h4.lineHeight, "rem"),
  "--pluto-h5-size": unitProperty(theme.typography.h5.size, "rem"),
  "--pluto-h5-weight": theme.typography.h5.weight,
  "--pluto-h5-line-height": unitProperty(theme.typography.h5.lineHeight, "rem"),
  "--pluto-h5-text-transform": theme.typography.h5.textTransform,
  "--pluto-p-size": unitProperty(theme.typography.p.size, "rem"),
  "--pluto-p-weight": theme.typography.p.weight,
  "--pluto-p-line-height": unitProperty(theme.typography.p.lineHeight, "rem"),
  "--pluto-small-size": unitProperty(theme.typography.small.size, "rem"),
  "--pluto-small-weight": theme.typography.small.weight,
  "--pluto-small-line-height": unitProperty(theme.typography.small.lineHeight, "rem"),
  "--pluto-tiny-size": unitProperty(theme.typography.tiny.size, "rem"),
  "--pluto-tiny-weight": theme.typography.tiny.weight,
  "--pluto-tiny-line-height": unitProperty(theme.typography.tiny.lineHeight, "rem"),
});
