// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, TimeStamp } from "@synnaxlabs/x";
import React, {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
} from "react";
import { z } from "zod";

import { context } from "@/context";
import { useSyncedRef } from "@/hooks";
import { type state } from "@/state";

export const relevancyZ = z.object({
  lastUsed: z.number(),
  count: z.number(),
  relevance: z.number(),
});

export const frequentZ = z.record(z.string(), relevancyZ);

export interface Frequent extends z.infer<typeof frequentZ> {}

export const contextStateZ = z.object({
  palettes: z.record(z.string(), color.paletteZ),
  frequent: z.record(z.string(), relevancyZ),
});

export interface ContextState extends z.infer<typeof contextStateZ> {}

export interface ContextValue extends ContextState {
  updateFrequent: (color: color.Color) => void;
}

export const ZERO_CONTEXT_STATE: ContextState = {
  palettes: { frequent: { key: "frequent", name: "Frequent", swatches: [] } },
  frequent: {},
};

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: { ...ZERO_CONTEXT_STATE, updateFrequent: () => undefined },
  displayName: "Color.Context",
});
export { useContext };

const RECENCY_WEIGHT = 0.6;
const FREQUENCY_WEIGHT = 0.4;

export const recalculate = (limit: number, frequent: Frequent): Frequent => {
  const now = Number(TimeStamp.now().valueOf());
  const maxFrequency = Object.values(frequent).reduce(
    (acc, { count }) => Math.max(acc, count),
    0,
  );
  const maxRecency = Object.values(frequent).reduce(
    (acc, { lastUsed }) => Math.max(acc, lastUsed),
    0,
  );
  Object.entries(frequent).forEach(([hex, { lastUsed, count }]) => {
    const normalizedRecency = 1 - (now - lastUsed) / maxRecency;
    const normalizedFrequency = count / maxFrequency;
    const relevance =
      normalizedRecency * RECENCY_WEIGHT + normalizedFrequency * FREQUENCY_WEIGHT;
    frequent[hex] = { lastUsed, count, relevance };
  });
  const entries = Object.entries(frequent);
  if (entries.length <= limit) return frequent;
  entries.sort((a, b) => b[1].relevance - a[1].relevance);
  return Object.fromEntries(entries.slice(0, limit));
};

export interface ProviderProps extends PropsWithChildren<{}> {
  useState?: state.PureUse<ContextState>;
}

export const Provider = ({
  useState = React.useState,
  children,
}: ProviderProps): ReactElement => {
  const [value, setValue] = useState(ZERO_CONTEXT_STATE);

  const valueRef = useSyncedRef(value);

  const updateFrequent = useCallback(
    (colorVal: color.Color) => {
      const prev = valueRef.current;
      const hex = color.hex(colorVal);
      const count = prev.frequent[hex]?.count ?? 0;
      const next: Frequent = {
        ...prev.frequent,
        [hex]: {
          lastUsed: Number(TimeStamp.now().valueOf()),
          count: count + 1,
          relevance: 0,
        },
      };
      const nextFreq = recalculate(10, next);
      setValue({ ...prev, frequent: nextFreq });
    },
    [setValue],
  );

  const memoValue = useMemo(
    () => ({ ...value, updateFrequent }),
    [value, updateFrequent],
  );
  return <Context value={memoValue}>{children}</Context>;
};

export const useFrequent = (): color.Color[] => {
  const { frequent } = useContext();
  return Object.keys(frequent).map((hex) => color.construct(hex));
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
