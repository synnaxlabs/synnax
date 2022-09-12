import { PropsWithChildren } from "react";
import { Theme } from "./theme";
export declare const useThemeContext: () => {
    theme: Theme;
    toggleTheme: () => void;
};
export declare const ThemeProvider: ({ children, themes, }: PropsWithChildren<{
    themes: Theme[];
}>) => JSX.Element;
export declare const ThemeSwitch: () => JSX.Element;
