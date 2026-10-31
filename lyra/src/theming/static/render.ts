// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@/theme";
import { toCSSVars } from "@/theming/css";

const INDENTATION = "    ";

const formatVars = (
  vars: Record<string, string | number | undefined>,
  indentationLevel: number = 1,
): string =>
  `${INDENTATION.repeat(indentationLevel)}${Object.entries(vars)
    .filter(([_, value]) => value != null)
    .map(([key, value]) => `${key}: ${value};`)
    .join(`\n${INDENTATION.repeat(indentationLevel)}`)}`;

const generateStatic = (
  light: theme.Theme,
  dark: theme.Theme,
  defaultTheme: "light" | "dark" = "light",
): string => {
  const lightVars = toCSSVars(light);
  const darkVars = toCSSVars(dark);
  const darkPrefixedVars = toCSSVars(dark, "dark-");

  const defaultVars = defaultTheme === "light" ? lightVars : darkVars;
  const mediaQueryVars = defaultTheme === "light" ? darkVars : lightVars;
  const mediaQueryTheme = defaultTheme === "light" ? "dark" : "light";

  return `:root {
${formatVars(defaultVars)}
${formatVars(darkPrefixedVars)}
}

@media (prefers-color-scheme: ${mediaQueryTheme}) {
    :root {
${formatVars(mediaQueryVars, 2)}
${formatVars(darkPrefixedVars, 2)}
    }
}
`;
};

const generateSingleTheme = (theme: theme.Theme, dark: theme.Theme): string => {
  const vars = toCSSVars(theme);
  const darkPrefixedVars = toCSSVars(dark, "dark-");
  return `:root {
${formatVars(vars)}
${formatVars(darkPrefixedVars)}
}
`;
};

/** @returns the body of each static theme file by name, without its license header. */
export const render = (): Record<string, string> => {
  const light = theme.themeZ.parse(theme.SYNNAX_LIGHT);
  const dark = theme.themeZ.parse(theme.SYNNAX_DARK);
  return {
    "theme.css": generateStatic(light, dark),
    "theme-dark.css": generateSingleTheme(dark, dark),
    "theme-light.css": generateSingleTheme(light, dark),
  };
};
