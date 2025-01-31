// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/theming/theme.css";

import geistMono from "@fontsource/geist-mono/files/geist-mono-latin-400-normal.woff2";
import interWoff from "@fontsource-variable/inter/files/inter-latin-standard-normal.woff2";
import { caseconv, deep } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useContext as reactUseContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
} from "react";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { Input } from "@/input";
import { type SwitchProps } from "@/input/Switch";
import { theming } from "@/theming/aether";
import { toCSSVars } from "@/theming/css";

export interface ContextValue {
  theme: theming.Theme;
  toggleTheme: () => void;
  setTheme: (key: string) => void;
}

const Context = createContext<ContextValue>({
  theme: theming.themeZ.parse(theming.SYNNAX_THEMES.synnaxLight),
  toggleTheme: () => undefined,
  setTheme: () => undefined,
});

export interface UseProviderProps {
  theme?: deep.Partial<theming.ThemeSpec> & { key: string };
  setTheme?: (key: string) => void;
  toggleTheme?: () => void;
  themes?: Record<string, theming.ThemeSpec>;
  lightTheme?: string;
  darkTheme?: string;
}

export type UseProviderReturn = ContextValue;

const prefersDark = (): MediaQueryList | null => {
  if (typeof window?.matchMedia === "undefined") return null;
  return window.matchMedia("(prefers-color-scheme: dark)");
};

const isDarkMode = (): boolean => prefersDark()?.matches ?? true;

export const useProvider = ({
  theme,
  themes = theming.SYNNAX_THEMES,
  setTheme,
  toggleTheme,
  lightTheme = "synnaxLight",
  darkTheme = "synnaxDark",
}: UseProviderProps): UseProviderReturn => {
  const [selected, setSelected] = useState<string>(
    isDarkMode() ? darkTheme : lightTheme,
  );

  const parsedThemes = useMemo(() => {
    if (theme != null) {
      const synnaxLight = theming.themeZ.parse(
        deep.override(deep.copy(theming.SYNNAX_LIGHT), theme),
      );
      const synnaxDark = theming.themeZ.parse(
        deep.override(deep.copy(theming.SYNNAX_DARK), theme),
      );
      if (theme.key != null && theme.key.length > 0) setSelected(theme.key);
      return { synnaxLight, synnaxDark };
    }
    return Object.entries(themes).reduce<Record<string, theming.Theme>>(
      (acc, [key, value]) => ({ ...acc, [key]: theming.themeZ.parse(value) }),
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

  useEffect(() => {
    const listener = (): void => setSelected(isDarkMode() ? darkTheme : lightTheme);
    prefersDark()?.addEventListener("change", listener);
    return () => prefersDark()?.removeEventListener("change", listener);
  }, []);

  return {
    theme: parsedTheme,
    toggleTheme: toggleTheme ?? handleToggle,
    setTheme: setTheme ?? setSelected,
  };
};

export const useContext = (): ContextValue => reactUseContext(Context);

export const use = (): theming.Theme => useContext().theme;

export interface ProviderProps extends PropsWithChildren<unknown>, UseProviderProps {
  applyCSSVars?: boolean;
  defaultTheme?: string;
}

const CLASS_PREFIX = "pluto-theme-";

const setThemeClass = (el: HTMLElement, theme: theming.Theme): void => {
  const existing = Array.from(el.classList).find((c) => c.startsWith(CLASS_PREFIX));
  if (existing != null) el.classList.remove(existing);
  el.classList.add(`${CLASS_PREFIX}${caseconv.toKebab(theme.key)}`);
};

export const Provider = ({
  children,
  applyCSSVars = true,
  ...props
}: ProviderProps): ReactElement => {
  const ret = useProvider(props);
  const [{ path }, , setAetherTheme] = Aether.use({
    type: theming.Provider.TYPE,
    schema: theming.Provider.z,
    initialState: {
      theme: ret.theme,
      fontURLs: [
        { name: "Inter Variable", url: interWoff },
        { name: "Geist Mono", url: geistMono },
      ],
    },
  });

  useEffect(() => setAetherTheme((p) => ({ ...p, theme: ret.theme })), [ret.theme.key]);

  useLayoutEffect(() => {
    const el = document.documentElement;
    setThemeClass(el, ret.theme);
    if (applyCSSVars) CSS.applyVars(el, toCSSVars(ret.theme));
    else CSS.removeVars(el, "--pluto");
  }, [ret.theme]);
  return (
    <Context.Provider value={ret}>
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </Context.Provider>
  );
};

export const Switch = ({
  ...props
}: Omit<SwitchProps, "onChange" | "value">): ReactElement => {
  const { toggleTheme } = useContext();
  const [checked, setChecked] = useState(false);
  return (
    <Input.Switch
      value={checked}
      onChange={(v) => {
        toggleTheme();
        setChecked(v);
      }}
      {...props}
    />
  );
};
