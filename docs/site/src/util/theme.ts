// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Theming } from "@synnaxlabs/pluto/theming";

export const applyCSSVars = (
  element: HTMLElement,
  vars: Record<string, string | number | undefined>,
): void =>
  Object.entries(vars).forEach(
    ([key, value]) => value != null && element.style.setProperty(key, `${value}`),
  );

const modifyTheme = (theme: Theming.ThemeSpec): Theming.ThemeSpec => {
  const m = { ...theme };
  m.sizes.base = 7;
  m.typography.small.lineHeight = m.typography.small.size * 1.3;
  m.typography.p.lineHeight = 4;
  m.typography.small.weight = 350;
  m.typography.p.weight = 350;
  m.typography.h1 = {
    ...m.typography.h1,
    lineHeight: 8,
    size: 7,
    weight: 450,
  };
  m.typography.h2 = {
    ...m.typography.h2,
    weight: 550,
  };
  m.typography.h3 = {
    ...m.typography.h3,
    lineHeight: m.typography.h3.size * 1.5,
    weight: 500,
  };
  m.typography.h4 = {
    ...m.typography.h4,
    lineHeight: m.typography.h4.size * 1.5,
    weight: 500,
  };

  m.typography.h5.textTransform = "none";
  return m;
};

export const DARK = Theming.themeZ.parse(modifyTheme(Theming.themes.synnaxDark));
export const LIGHT = Theming.themeZ.parse(modifyTheme(Theming.themes.synnaxLight));

export const DEFAULT_THEME = modifyTheme(Theming.themes.synnaxDark);

export const startThemeDriver = (): void => {
  applyTheme(getPreferredTheme());
  listenForThemeChanges();
};

export const toggleTheme = (): void => {
  const theme = THEME_ALTERNATES[localStorage.getItem("theme") ?? "synnaxDark"];
  applyTheme(theme);
};

export const getCurrentTheme = (): Theming.ThemeSpec => {
  const theme = localStorage.getItem("theme") ?? "synnaxDark";
  return Theming.themes[theme as keyof typeof Theming.themes];
};

const THEME_ALTERNATES: Record<string, Theming.ThemeSpec> = {
  [DARK.key]: LIGHT,
  [LIGHT.key]: DARK,
};

const applyTheme = (theme: Theming.ThemeSpec): void => {
  applyCSSVars(
    window.document.documentElement,
    Theming.toCSSVars(Theming.themeZ.parse(theme)),
  );
  localStorage.setItem("theme", theme.key);
};

const prefersDarkTheme = (): boolean =>
  window.matchMedia("(prefers-color-scheme: dark)").matches;

const getPreferredTheme = (): Theming.ThemeSpec =>
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
