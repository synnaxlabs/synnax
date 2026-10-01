// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, deep, zod } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
} from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { theme as baseTheme } from "@/theme";
import { toCSSVars } from "@/theming/css";

export interface ContextValue {
  theme: baseTheme.Theme;
  toggleTheme: () => void;
  setTheme: (key: string) => void;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: {
    theme: zod.parse(baseTheme.themeZ, baseTheme.SYNNAX_THEMES.synnaxLight, {
      label: "theme",
    }),
    toggleTheme: () => {},
    setTheme: () => {},
  },
  displayName: "Theming.Context",
});
export { useContext };

export interface UseProviderProps {
  theme?: deep.Partial<baseTheme.ThemeSpec> & { key: string };
  setTheme?: (key: string) => void;
  toggleTheme?: () => void;
  themes?: Record<string, baseTheme.ThemeSpec>;
  lightTheme?: string;
  darkTheme?: string;
}

const prefersDark = () =>
  typeof window?.matchMedia === "undefined"
    ? null
    : window.matchMedia("(prefers-color-scheme: dark)");

const isDarkMode = (): boolean => prefersDark()?.matches ?? true;

export const useProvider = ({
  theme,
  themes = baseTheme.SYNNAX_THEMES,
  setTheme,
  toggleTheme,
  lightTheme = "synnaxLight",
  darkTheme = "synnaxDark",
}: UseProviderProps): ContextValue => {
  const [selected, setSelected] = useState<string>(
    isDarkMode() ? darkTheme : lightTheme,
  );

  const parsedThemes = useMemo(() => {
    if (theme != null) {
      // The override applies to both variants and each keeps its own key, since
      // consumers tell light from dark by key. The caller's key pins a variant when it
      // names one and is otherwise ignored.
      const { key, ...override } = theme;
      const synnaxLight = zod.parse(
        baseTheme.themeZ,
        deep.override(deep.copy(baseTheme.SYNNAX_LIGHT), override),
        { label: "theme" },
      );
      const synnaxDark = zod.parse(
        baseTheme.themeZ,
        deep.override(deep.copy(baseTheme.SYNNAX_DARK), override),
        { label: "theme" },
      );
      const overridden: Record<string, baseTheme.Theme> = { synnaxLight, synnaxDark };
      if (Object.hasOwn(overridden, key)) setSelected(key);
      return overridden;
    }
    return Object.entries(themes).reduce<Record<string, baseTheme.Theme>>(
      (acc, [key, value]) => ({
        ...acc,
        [key]: zod.parse(baseTheme.themeZ, value, { label: "theme" }),
      }),
      {},
    );
  }, [theme, themes]);

  const handleToggle = useCallback((): void => {
    const keys = Object.keys(themes);
    const index = keys.indexOf(selected);
    const nextIndex = (index + 1) % keys.length;
    setSelected(keys[nextIndex]);
  }, [toggleTheme, selected, themes]);

  const parsedTheme = useMemo(() => parsedThemes[selected], [parsedThemes, selected]);

  // When a caller supplies theme, they own OS-sync (or the decision to opt out of
  // it) themselves; listening here too would flip selected out from under a
  // caller that has deliberately pinned the theme.
  useEffect(() => {
    if (theme != null) return;
    const listener = (): void => setSelected(isDarkMode() ? darkTheme : lightTheme);
    prefersDark()?.addEventListener("change", listener);
    return () => prefersDark()?.removeEventListener("change", listener);
  }, [theme]);

  return {
    theme: parsedTheme,
    toggleTheme: toggleTheme ?? handleToggle,
    setTheme: setTheme ?? setSelected,
  };
};

export const use = (): baseTheme.Theme => useContext().theme;

export interface ProviderProps extends PropsWithChildren<unknown>, UseProviderProps {
  applyCSSVars?: boolean;
  defaultTheme?: string;
  el?: HTMLElement | null;
}

const CLASS_PREFIX = "pluto-theme-";

const setThemeClass = (el: HTMLElement, theme: baseTheme.Theme): void => {
  const existing = Array.from(el.classList).find((c) => c.startsWith(CLASS_PREFIX));
  if (existing != null) el.classList.remove(existing);
  el.classList.add(`${CLASS_PREFIX}${caseconv.toKebab(theme.key)}`);
};

export const Provider = ({
  children,
  applyCSSVars = true,
  el,
  ...rest
}: ProviderProps): ReactElement => {
  const ret = useProvider(rest);
  // The ret.theme.key dep will not trigger a re-render if the theme properties change
  // but not the key. This reduces re-renders, but should be corrected once the user
  // has the ability to edit the theme themselves.
  useLayoutEffect(() => {
    if (el === undefined) el = document.documentElement;
    if (el == null) return;
    setThemeClass(el, ret.theme);
    if (applyCSSVars) CSS.applyVars(el, toCSSVars(ret.theme));
    else CSS.removeVars(el, "--pluto");
  }, [ret.theme.key, el]);

  return <Context value={ret}>{children}</Context>;
};
