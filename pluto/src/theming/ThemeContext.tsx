import {
  PropsWithChildren,
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { Theme, synnaxLight } from "./theme";
import { applyThemeAsCssVars } from "./css";
import "./theme.css";
import { Switch, SwitchProps } from "@/atoms";

export interface ThemeProviderProps extends PropsWithChildren<unknown> {
  theme: Theme;
  toggleTheme: () => void;
  setTheme: (key: string) => void;
}

const ThemeContext = createContext<ThemeProviderProps>({
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
}: UseThemeProviderProps) => {
  const [selected, setSelected] = useState<string>(
    defaultTheme || Object.keys(themes)[0]
  );

  const toggleTheme = () => {
    const keys = Object.keys(themes);
    const index = keys.indexOf(selected);
    const nextIndex = (index + 1) % keys.length;
    setSelected(keys[nextIndex]);
  };

  const setTheme = (key: string) => {
    setSelected(key);
  };

  const theme = useMemo(() => themes[selected], [selected]);

  return {
    theme,
    toggleTheme,
    setTheme,
  };
};

export const useThemeContext = () => useContext(ThemeContext);

export const ThemeProvider = ({
  theme,
  children,
  ...props
}: ThemeProviderProps) => {
  useEffect(() => {
    applyThemeAsCssVars(document.documentElement, theme);
  }, [theme]);
  return (
    <ThemeContext.Provider value={{ theme, ...props }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const ThemeSwitch = ({ onChange, ...props }: SwitchProps) => {
  const { toggleTheme } = useContext(ThemeContext);
  return (
    <Switch
      onChange={(e) => {
        toggleTheme();
        if (onChange) onChange(e);
      }}
      {...props}
    />
  );
};
