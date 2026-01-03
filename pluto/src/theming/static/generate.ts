// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

import { SYNNAX_DARK, SYNNAX_LIGHT, type Theme, themeZ } from "@/theming/core/theme";
import { toCSSVars } from "@/theming/css";

const INDENTATION = "    ";

/**
 * Generates a static css file containing the specified dark and light themes.
 * @param light - The light theme.
 * @param dark - The dark theme.
 * @param defaultTheme - The default theme to use.
 * @returns A string containing the generated css.
 */
const generateStatic = (
  light: Theme,
  dark: Theme,
  defaultTheme: "light" | "dark" = "light",
): string => {
  const lightVars = toCSSVars(light);
  const darkVars = toCSSVars(dark);
  const darkPrefixedVars = toCSSVars(dark, "dark-");

  // Convert record to CSS variable declarations
  const formatVars = (
    vars: Record<string, string | number | undefined>,
    indentationLevel: number = 1,
  ): string =>
    `${INDENTATION.repeat(indentationLevel)}${Object.entries(vars)
      .filter(([_, value]) => value != null)
      .map(([key, value]) => `${key}: ${value};`)
      .join(`\n${INDENTATION.repeat(indentationLevel)}`)}`;

  // Determine which theme to use as default and which for the media query
  const defaultVars = defaultTheme === "light" ? lightVars : darkVars;
  const mediaQueryVars = defaultTheme === "light" ? darkVars : lightVars;
  const mediaQueryTheme = defaultTheme === "light" ? "dark" : "light";

  // Read the copyright header from the licenses directory
  const __filename = fileURLToPath(import.meta.url);
  const __dirname = path.dirname(__filename);
  const headerPath = path.resolve(__dirname, "../../../licenses/headers/template.txt");
  const headerContent = fs.readFileSync(headerPath, "utf-8").trim();

  // Replace template arguments
  const currentYear = new Date().getFullYear();
  const processedHeader = headerContent.replace(
    /\{\{YEAR\}\}/g,
    currentYear.toString(),
  );

  // Format as CSS comment
  const copyrightHeader = `/*
 * ${processedHeader.split("\n").join("\n * ")}
 */

`;

  return `${copyrightHeader}:root {
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

const writeToFile = (content: string) => {
  // get the path of this file
  const __filename = fileURLToPath(import.meta.url);
  const __dirname = path.dirname(__filename);
  fs.writeFileSync(path.join(__dirname, "theme.css"), content, "utf-8");
};

writeToFile(generateStatic(themeZ.parse(SYNNAX_LIGHT), themeZ.parse(SYNNAX_DARK)));
