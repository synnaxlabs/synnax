// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  PropsWithChildren,
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import { applyThemeAsCssVars } from "./css";
import { Theme, synnaxLight } from "./theme";

import "./theme.css";
import { Input, InputProps } from "@/atoms";

export interface ThemeContextValue {
  theme: Theme;
  toggleTheme: () => void;
  setTheme: (key: string) => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: synnaxLight,
  toggleTheme: () => undefined,
  setTheme: () => undefined,
});

export interface UseThemeProviderProps {
  themes: Record<string, Theme>;
  defaultTheme?: string;
}

export const useThemeProvider = ({
  themes,
  defaultTheme,
}: UseThemeProviderProps): ThemeProviderProps => {
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
    ThemeContextValue {}

export const ThemeProvider = ({
  theme,
  children,
  ...props
}: ThemeProviderProps): JSX.Element => {
  useEffect(() => {
    applyThemeAsCssVars(document.documentElement, theme);
  }, [theme]);
  return (
    <ThemeContext.Provider value={{ theme, ...props }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const ThemeSwitch = ({ onChange, ...props }: InputProps): JSX.Element => {
  const { toggleTheme } = useContext(ThemeContext);
  return (
    <Input.Switch
      onChange={(e) => {
        toggleTheme();
        if (onChange != null) onChange(e);
      }}
      {...props}
    />
  );
};
