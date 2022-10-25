import { synnaxDark, synnaxLight } from "./theme";
export * from "./hooks";
import { ThemeProvider, ThemeSwitch, useThemeContext } from "./ThemeContext";
export type { ThemeProps } from "./theme";

export const Theme = {
  Provider: ThemeProvider,
  Switch: ThemeSwitch,
  useContext: useThemeContext,
  themes: { synnaxDark, synnaxLight },
};
