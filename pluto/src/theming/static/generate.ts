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

import { SYNNAX_DARK, SYNNAX_LIGHT, type Theme, themeZ } from "@/theming/base/theme";
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

const copyrightHeader = (): string => {
  const __filename = fileURLToPath(import.meta.url);
  const __dirname = path.dirname(__filename);
  const headerPath = path.resolve(
    __dirname,
    "../../../../licenses/headers/template.txt",
  );
  const headerContent = fs.readFileSync(headerPath, "utf-8").trim();
  const currentYear = new Date().getFullYear();
  const processedHeader = headerContent.replace(
    /\{\{YEAR\}\}/g,
    currentYear.toString(),
  );
  return `/*
 * ${processedHeader.split("\n").join("\n * ")}
 */

`;
};

const generateStatic = (
  light: Theme,
  dark: Theme,
  defaultTheme: "light" | "dark" = "light",
): string => {
  const lightVars = toCSSVars(light);
  const darkVars = toCSSVars(dark);
  const darkPrefixedVars = toCSSVars(dark, "dark-");

  const defaultVars = defaultTheme === "light" ? lightVars : darkVars;
  const mediaQueryVars = defaultTheme === "light" ? darkVars : lightVars;
  const mediaQueryTheme = defaultTheme === "light" ? "dark" : "light";

  return `${copyrightHeader()}:root {
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

const generateSingleTheme = (theme: Theme, dark: Theme): string => {
  const vars = toCSSVars(theme);
  const darkPrefixedVars = toCSSVars(dark, "dark-");
  return `${copyrightHeader()}:root {
${formatVars(vars)}
${formatVars(darkPrefixedVars)}
}
`;
};

const writeToFile = (filename: string, content: string) => {
  const __filename = fileURLToPath(import.meta.url);
  const __dirname = path.dirname(__filename);
  fs.writeFileSync(path.join(__dirname, filename), content, "utf-8");
};

const light = themeZ.parse(SYNNAX_LIGHT);
const dark = themeZ.parse(SYNNAX_DARK);
writeToFile("theme.css", generateStatic(light, dark));
writeToFile("theme-dark.css", generateSingleTheme(dark, dark));
writeToFile("theme-light.css", generateSingleTheme(light, dark));
