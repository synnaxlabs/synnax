// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
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
  value: K | K[] | null | undefined;
  onSelect: (key: K) => void;
}

export interface ProviderProps<K extends record.Key = record.Key>
  extends ContextValue<K>,
    PropsWithChildren {
  onSelect: (key: K) => void;
}

export const Provider = <K extends record.Key = record.Key>({
  value,
  onSelect,
}: ProviderProps<K>): ReactElement => {
  const ctx = useMemo(() => ({ value, onSelect }), [value, onSelect]);
  return <Context.Provider value={ctx}></Context.Provider>;
};

export const useItem = <K extends record.Key>(key: K): [boolean, () => void] => {
  const { value, onSelect } = useRequiredContext(Context);
  const handleSelect = useCallback(() => onSelect(key), [key, onSelect]);
  return [isSelected(value, key), handleSelect];
};
