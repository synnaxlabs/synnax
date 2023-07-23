// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { convertThemeToCSSVars } from "@/core/theming/css";
import { fontString, useTypography } from "@/core/theming/font";
import { ThemeSpec, themeZ, themes } from "@/core/theming/theme";
import {
  ThemeProvider,
  ThemeSwitch,
  useThemeContext,
  useThemeProvider,
} from "@/core/theming/ThemeContext";
export type { ThemeSpec as Theme } from "@/core/theming/theme";
export type { ThemeProviderProps } from "@/core/theming/ThemeContext";

export interface ThemingType {
  Provider: typeof ThemeProvider;
  Switch: typeof ThemeSwitch;
  useContext: typeof useThemeContext;
  useProvider: typeof useThemeProvider;
  use: () => ThemeSpec;
  themes: typeof themes;
  toCSSVars: typeof convertThemeToCSSVars;
  fontString: typeof fontString;
  useTypography: typeof useTypography;
  schema: typeof themeZ;
}

export const Theming: ThemingType = {
  Provider: ThemeProvider,
  Switch: ThemeSwitch,
  useContext: useThemeContext,
  useProvider: useThemeProvider,
  use: (): ThemeSpec => useThemeContext().theme,
  themes,
  toCSSVars: convertThemeToCSSVars,
  fontString,
  useTypography,
  schema: themeZ,
};
