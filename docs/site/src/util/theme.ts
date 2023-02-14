import { Theme, Theming } from "@synnaxlabs/pluto";
import { applyCSSVars } from "@synnaxlabs/x";

const modifyTheme = (theme: Theme): Theme => {
  const m = { ...theme };
  m.sizes.base = 6.5;
  m.typography.p.lineHeight = 3.5;
  m.typography.h1 = {
    ...m.typography.h1,
    lineHeight: 8,
    size: 7,
  };
  return m;
};

const DARK = modifyTheme(Theming.themes.synnaxDark);
const LIGHT = modifyTheme(Theming.themes.synnaxLight);

export const DEFAULT_THEME = Theming.themes.synnaxLight;

export const startThemeDriver = (): void => {
  applyTheme(getPreferredTheme());
  listenForThemeChanges();
};

export const toggleTheme = (): void => {
  const theme = THEME_ALTERNATES[localStorage.getItem("theme") ?? "synnaxLight"];
  applyTheme(theme);
};

export const getCurrentTheme = (): Theme => {
  const theme = localStorage.getItem("theme") ?? "synnaxLight";
  return Theming.themes[theme as keyof typeof Theming.themes];
};

const THEME_ALTERNATES: Record<string, Theme> = {
  [DARK.key]: LIGHT,
  [LIGHT.key]: DARK,
};

const applyTheme = (theme: Theme): void => {
  applyCSSVars(window.document.documentElement, Theming.toCSSVars(theme));
  localStorage.setItem("theme", theme.key);
};

const prefersDarkTheme = (): boolean =>
  window.matchMedia("(prefers-color-scheme: dark)").matches;

const getPreferredTheme = (): Theme =>
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
