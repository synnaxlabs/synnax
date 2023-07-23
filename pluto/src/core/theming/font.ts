// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TypographyLevel, Typography, TypographySpec } from "@/core/std";
import { ThemeSpec } from "@/core/theming/theme";
import { useThemeContext } from "@/core/theming/ThemeContext";
import { ComponentSize, isComponentSize } from "@/util/component";

const { ComponentSizeLevels } = Typography;

export const fontString = (
  theme: ThemeSpec,
  level: TypographyLevel | ComponentSize
): string => {
  const {
    typography,
    sizes: { base },
  } = theme;
  const size =
    typography[isComponentSize(level) ? ComponentSizeLevels[level] : level].size;
  return ` ${base * size}px ${typography.family}`;
};

export interface UseTypographyReturn extends TypographySpec {
  toString: () => string;
  baseSize: number;
  lineHeightPx: number;
  sizePx: number;
}

export const useTypography = (level: TypographyLevel): UseTypographyReturn => {
  const { theme } = useThemeContext();
  const t = theme.typography[level];
  return {
    ...t,
    toString: () => fontString(theme, level),
    baseSize: theme.sizes.base,
    lineHeightPx: t.lineHeight * theme.sizes.base,
    sizePx: t.size * theme.sizes.base,
  };
};
