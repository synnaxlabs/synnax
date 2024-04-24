// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type PropsWithChildren,
  type ReactElement,
  createContext,
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

import "@/theming/theme.css";

export interface ContextValue {
  theme: theming.Theme;
  toggleTheme: () => void;
  setTheme: (key: string) => void;
}

const Context = createContext<ContextValue>({
  theme: theming.themeZ.parse(theming.themes.synnaxLight),
  toggleTheme: () => undefined,
  setTheme: () => undefined,
});

export interface UseProviderProps {
  themes: Record<string, theming.ThemeSpec>;
  defaultTheme?: string;
}

export type UseProviderReturn = ContextValue;

export const useProvider = ({
  themes,
  defaultTheme,
}: UseProviderProps): UseProviderReturn => {
  const [selected, setSelected] = useState<string>(
    defaultTheme ?? Object.keys(themes)[0],
  );

  const parsedThemes = useMemo(
    () =>
      Object.entries(themes).reduce<Record<string, theming.Theme>>(
        (acc, [key, value]) => ({ ...acc, [key]: theming.themeZ.parse(value) }),
        {},
      ),
    [themes],
  );

  const toggleTheme = (): void => {
    const keys = Object.keys(themes);
    const index = keys.indexOf(selected);
    const nextIndex = (index + 1) % keys.length;
    setSelected(keys[nextIndex]);
  };

  const parsedTheme = useMemo(() => parsedThemes[selected], [parsedThemes, selected]);

  return {
    theme: parsedTheme,
    toggleTheme,
    setTheme: setSelected,
  };
};

export const useContext = (): ContextValue => reactUseContext(Context);

export const use = (): theming.Theme => useContext().theme;

export interface ProviderProps
  extends PropsWithChildren<unknown>,
    Partial<ContextValue> {
  applyCSSVars?: boolean;
}

export const Provider = Aether.wrap<ProviderProps>(
  theming.Provider.TYPE,
  ({
    children,
    theme,
    setTheme,
    toggleTheme,
    aetherKey,
    applyCSSVars = true,
  }): ReactElement => {
    let ret: UseProviderReturn;
    if (theme == null || toggleTheme == null || setTheme == null) {
      ret = useProvider({
        themes: theming.themes,
        defaultTheme: "synnaxDark",
      });
    } else {
      ret = {
        theme,
        toggleTheme,
        setTheme,
      };
    }
    const [{ path }, , setAetherTheme] = Aether.use({
      aetherKey,
      type: theming.Provider.TYPE,
      schema: theming.Provider.z,
      initialState: { theme: ret.theme },
    });

    useEffect(() => {
      setAetherTheme({ theme: ret.theme });
    }, [ret.theme]);

    useLayoutEffect(() => {
      if (applyCSSVars) CSS.applyVars(document.documentElement, toCSSVars(ret.theme));
      else CSS.removeVars(document.documentElement, "--pluto");
    }, [ret.theme]);
    return (
      <Context.Provider value={ret}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </Context.Provider>
    );
  },
);

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
