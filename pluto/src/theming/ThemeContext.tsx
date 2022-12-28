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
import { Input } from "@/atoms";
import { InputSwitchProps } from "@/atoms/Input/InputSwitch";

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

export const ThemeSwitch = ({
  ...props
}: Omit<InputSwitchProps, "onChange" | "value">): JSX.Element => {
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
