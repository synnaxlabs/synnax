// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  PropsWithChildren,
  ReactElement,
  createContext,
  useContext,
  useLayoutEffect,
  useMemo,
  useState,
} from "react";

import { CSS } from "@/core/css";
import { Input } from "@/core/std/Input";
import { InputSwitchProps } from "@/core/std/Input/InputSwitch";
import { convertThemeToCSSVars } from "@/core/theming/css";
import { Theme, themes } from "@/core/theming/theme";

import "@/core/theming/theme.css";

export interface ThemeContextValue {
  theme: Theme;
  toggleTheme: () => void;
  setTheme: (key: string) => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: themes.synnaxLight,
  toggleTheme: () => undefined,
  setTheme: () => undefined,
});

export interface UseThemeProviderProps {
  themes: Record<string, Theme>;
  defaultTheme?: string;
}

export type UseThemeProviderReturn = ThemeContextValue;

export const useThemeProvider = ({
  themes,
  defaultTheme,
}: UseThemeProviderProps): UseThemeProviderReturn => {
  const [selected, setSelected] = useState<string>(
    defaultTheme ?? Object.keys(themes)[0]
  );

  const toggleTheme = (): void => {
    const keys = Object.keys(themes);
    const index = keys.indexOf(selected);
    const nextIndex = (index + 1) % keys.length;
    setSelected(keys[nextIndex]);
  };

  const theme = useMemo(() => themes[selected], [selected]);

  return {
    theme,
    toggleTheme,
    setTheme: setSelected,
  };
};

export const useThemeContext = (): ThemeContextValue => useContext(ThemeContext);

export interface ThemeProviderProps
  extends PropsWithChildren<unknown>,
    Partial<ThemeContextValue> {}

export const ThemeProvider = ({
  children,
  theme,
  setTheme,
  toggleTheme,
}: ThemeProviderProps): ReactElement => {
  let ret: UseThemeProviderReturn;
  if (theme == null || toggleTheme == null || setTheme == null) {
    ret = useThemeProvider({
      themes,
      defaultTheme: "synnaxDark",
    });
  } else {
    ret = {
      theme,
      toggleTheme,
      setTheme,
    };
  }
  useLayoutEffect(
    () => CSS.applyVars(document.documentElement, convertThemeToCSSVars(ret.theme)),
    [ret.theme]
  );
  return <ThemeContext.Provider value={ret}>{children}</ThemeContext.Provider>;
};

export const ThemeSwitch = ({
  ...props
}: Omit<InputSwitchProps, "onChange" | "value">): ReactElement => {
  const { toggleTheme } = useContext(ThemeContext);
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
