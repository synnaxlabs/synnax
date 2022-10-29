import {
  createContext,
  useContext,
  useState,
  PropsWithChildren,
  useEffect,
  useMemo,
} from "react";
import { Theme, synnaxLight } from "./theme";
import { applyThemeAsCssVars } from "./css";
import "./theme.css";
import { Switch } from "@/atoms";

export interface ThemeProviderProps extends PropsWithChildren<any> {
  theme: Theme;
  toggleTheme: () => void;
  setTheme: (key: string) => void;
}

const ThemeContext = createContext<ThemeProviderProps>({
  theme: synnaxLight,
  toggleTheme: () => {},
  setTheme: (key: string) => {},
});

export const useThemeProvider = (
  themes: Record<string, Theme>
): ThemeProviderProps => {
  const [selected, setSelected] = useState<string>(
    Object.values(themes)[0]?.key
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

export const ThemeSwitch = () => {
  const { theme, toggleTheme } = useContext(ThemeContext);
  return (
    <Switch
      onChange={() => {
        toggleTheme();
      }}
    ></Switch>
  );
};
