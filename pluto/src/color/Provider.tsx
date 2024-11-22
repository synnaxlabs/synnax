import React, {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useContext as reactUseContext,
  useMemo,
} from "react";
import { z } from "zod";

import { color } from "@/color/core";
import { useSyncedRef } from "@/hooks";
import { type state } from "@/state";

export const contextStateZ = z.object({
  palettes: z.record(color.paletteZ),
  frequent: z.record(z.number()),
});

export type ContextState = z.infer<typeof contextStateZ>;

export interface ContextValue extends ContextState {
  updateFrequent: (color: color.Color) => void;
}

export const ZERO_CONTEXT_STATE: ContextState = {
  palettes: {
    frequent: {
      key: "frequent",
      name: "Frequent",
      swatches: [],
    },
  },
  frequent: {},
};

const ZERO_CONTEXT_VALUE: ContextValue = {
  ...ZERO_CONTEXT_STATE,
  updateFrequent: () => undefined,
};

const Context = createContext<ContextValue>(ZERO_CONTEXT_VALUE);

export interface ProviderProps extends PropsWithChildren<{}> {
  useState?: state.PureUse<ContextState>;
}

export const useContext = (): ContextValue => reactUseContext(Context);

export const purgeFrequent = (
  limit: number,
  frequent: Record<color.Hex, number>,
): Record<color.Hex, number> => {
  const entries = Object.entries(frequent);
  if (entries.length <= limit) return frequent;
  entries.sort((a, b) => b[1] - a[1]);
  return Object.fromEntries(entries.slice(0, limit));
};

export const Provider = ({
  useState = React.useState,
  children,
}: ProviderProps): ReactElement => {
  const [value, setValue] = useState(ZERO_CONTEXT_STATE);

  const valueRef = useSyncedRef(value);

  const updateFrequent = useCallback(
    (color: color.Color) => {
      const prev = valueRef.current;
      const hex = color.hex;
      const count = prev.frequent[hex] ?? 0;
      const nextFreq = purgeFrequent(10, {
        ...prev.frequent,
        [hex]: count + 1,
      });
      setValue({ ...prev, frequent: nextFreq });
    },
    [setValue],
  );

  const memoValue = useMemo(
    () => ({ ...value, updateFrequent }),
    [value, updateFrequent],
  );
  return <Context.Provider value={memoValue}>{children}</Context.Provider>;
};

export const useFrequent = (): color.Color[] => {
  const { frequent } = useContext();
  return Object.keys(frequent).map((hex) => new color.Color(hex));
};

export const usePalette = (key: string): color.Palette | null => {
  const { palettes } = useContext();
  return palettes[key];
};

export const useRequiredPalette = (key: string): color.Palette => {
  const palette = usePalette(key);
  if (palette == null) throw new Error(`Palette "${key}" not found`);
  return palette;
};

export const useFrequentUpdater = (): ((color: color.Color) => void) => {
  const { updateFrequent } = useContext();
  return updateFrequent;
};
