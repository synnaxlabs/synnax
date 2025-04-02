import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

import { SYNNAX_DARK, SYNNAX_LIGHT, type Theme, themeZ } from "@/theming/core/theme";
import { toCSSVars } from "@/theming/css";

/*
 * Generate a static css file containing the specified dark and light themes.

 */
const generateStatic = (
  light: Theme,
  dark: Theme,
  defaultTheme: "light" | "dark" = "light",
): string => {
  const lightVars = toCSSVars(light);
  const darkVars = toCSSVars(dark);

  // Convert record to CSS variable declarations
  const formatVars = (vars: Record<string, string | number | undefined>): string =>
    Object.entries(vars)
      .filter(([_, value]) => value != null)
      .map(([key, value]) => `${key}: ${value};`)
      .join("\n");

  // Determine which theme to use as default and which for the media query
  const defaultVars = defaultTheme === "light" ? lightVars : darkVars;
  const mediaQueryVars = defaultTheme === "light" ? darkVars : lightVars;
  const mediaQueryTheme = defaultTheme === "light" ? "dark" : "light";

  return `
:root {
${formatVars(defaultVars)}
}

@media (prefers-color-scheme: ${mediaQueryTheme}) {
    :root {
${formatVars(mediaQueryVars)}
    }
}`;
};

const writeToFile = (content: string) => {
  // get the path of this file
  const __filename = fileURLToPath(import.meta.url);
  const __dirname = path.dirname(__filename);
  fs.writeFileSync(path.join(__dirname, "theme.css"), content, "utf-8");
};

writeToFile(generateStatic(themeZ.parse(SYNNAX_LIGHT), themeZ.parse(SYNNAX_DARK)));
