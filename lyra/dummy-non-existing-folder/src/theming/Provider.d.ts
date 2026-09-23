import "./theme.css";
import { deep } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
import { theme as baseTheme } from "../theme";
export interface ContextValue {
    theme: baseTheme.Theme;
    toggleTheme: () => void;
    setTheme: (key: string) => void;
}
declare const useContext: () => ContextValue;
export { useContext };
export interface UseProviderProps {
    theme?: deep.Partial<baseTheme.ThemeSpec> & {
        key: string;
    };
    setTheme?: (key: string) => void;
    toggleTheme?: () => void;
    themes?: Record<string, baseTheme.ThemeSpec>;
    lightTheme?: string;
    darkTheme?: string;
}
export declare const useProvider: ({ theme, themes, setTheme, toggleTheme, lightTheme, darkTheme, }: UseProviderProps) => ContextValue;
export declare const use: () => baseTheme.Theme;
export interface ProviderProps extends PropsWithChildren<unknown>, UseProviderProps {
    applyCSSVars?: boolean;
    defaultTheme?: string;
    el?: HTMLElement | null;
}
export declare const Provider: ({ children, applyCSSVars, el, ...rest }: ProviderProps) => ReactElement;
//# sourceMappingURL=Provider.d.ts.map