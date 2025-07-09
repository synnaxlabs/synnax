// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type record } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
} from "react";

import { useRequiredContext } from "@/hooks";

const Context = createContext<ContextValue<any> | null>(null);

const isSelected = <K extends record.Key>(
  value: K | K[] | null | undefined,
  key: K,
): boolean => {
  if (value === null || value === undefined) return false;
  if (Array.isArray(value)) return value.includes(key);
  return value === key;
};

interface ContextValue<K extends record.Key = record.Key> {
  value: K[];
  onSelect: (key: K) => void;
  clear: () => void;
  hover?: K;
}

export interface ProviderProps<K extends record.Key = record.Key>
  extends Omit<ContextValue<K>, "value">,
    PropsWithChildren {
  value: K | K[] | null | undefined;
  hover?: K;
}

export const Provider = <K extends record.Key = record.Key>({
  value,
  onSelect,
  clear,
  children,
  hover,
}: ProviderProps<K>): ReactElement => {
  const ctx = useMemo(
    () => ({ value: array.toArray(value), onSelect, clear, hover }),
    [value, onSelect, clear, hover],
  );
  return <Context.Provider value={ctx}>{children}</Context.Provider>;
};

export interface UseItemStateReturn {
  selected: boolean;
  hovered: boolean;
  onSelect: () => void;
}

export const useItemState = <K extends record.Key>(key: K): UseItemStateReturn => {
  const { value, onSelect, hover } = useRequiredContext(Context);
  const handleSelect = useCallback(() => onSelect(key), [key, onSelect]);
  return {
    selected: isSelected(value, key),
    hovered: hover === key,
    onSelect: handleSelect,
  };
};

export const useSelection = <K extends record.Key>(): K[] => {
  const { value } = useRequiredContext(Context);
  return value;
};

export const useClear = () => useRequiredContext(Context).clear;
