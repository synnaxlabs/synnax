import { Theme, Theming } from "@synnaxlabs/pluto";
import { applyCSSVars } from "@synnaxlabs/x";

const themeAlternates: Record<string, Theme> = {
  [Theming.themes.synnaxDark.key]: Theming.themes.synnaxLight,
  [Theming.themes.synnaxLight.key]: Theming.themes.synnaxDark,
};

const applyTheme = (theme: Theme): void => {
  applyCSSVars(window.document.documentElement, Theming.toCSSVars(theme));
  localStorage.setItem("theme", theme.key);
};

const prefersDarkTheme = (): boolean => {
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
};

const getDefaultTheme = (): Theme =>
  prefersDarkTheme() ? Theming.themes.synnaxDark : Theming.themes.synnaxLight;

const listenForThemeChanges = (): void => {
  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", (event) => {
      const theme = event.matches
        ? Theming.themes.synnaxDark
        : Theming.themes.synnaxLight;
      applyTheme(theme);
    });
};

export const startThemeDriver = (): void => {
  applyTheme(getDefaultTheme());
  listenForThemeChanges();
};

export const toggleTheme = (): void => {
  const theme = themeAlternates[localStorage.getItem("theme") ?? "synnaxLight"];
  applyTheme(theme);
};

export const getCurrentTheme = (): Theme => {
  const theme = localStorage.getItem("theme") ?? "synnaxLight";
  return Theming.themes[theme as keyof typeof Theming.themes];
};

export const getBaseThemeCSSVars = (): Record<string, string | number | undefined> =>
  Theming.toCSSVars(Theming.themes.synnaxLight);
