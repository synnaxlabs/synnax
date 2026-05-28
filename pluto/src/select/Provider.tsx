// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";

import { context } from "@/context";
import { useSyncedRef } from "@/hooks/ref";
import { Store } from "@/store";

interface SelectionState<K extends record.Key = record.Key> {
  value: K | K[] | null | undefined;
  hover?: K;
}

const [Context, useCtx] = context.create<ContextValue>({
  defaultValue: {
    clear: () => {},
    getState: () => ({ value: undefined, hover: undefined }),
    onSelect: () => {},
    setSelected: () => {},
    subscribe: () => () => {},
  },
  displayName: "Select.Context",
});

const isSelected = <K extends record.Key>(
  value: K | K[] | null | undefined,
  key: K,
): boolean => {
  if (value === null || value === undefined) return false;
  if (Array.isArray(value)) return value.includes(key);
  return value === key;
};

interface ContextValue<K extends record.Key = record.Key> extends Pick<
  Store.UseKeyedListenersReturn<K>,
  "subscribe"
> {
  onSelect: (key: K) => void;
  setSelected: (keys: K[]) => void;
  clear: () => void;
  getState: () => SelectionState<K>;
}

export interface ProviderProps<
  K extends record.Key = record.Key,
> extends PropsWithChildren {
  value: K | K[] | null | undefined;
  onSelect?: (key: K) => void;
  setSelected?: (keys: K[]) => void;
  clear?: () => void;
  hover?: K;
}

const NOOP = () => {};

export const Provider = <K extends record.Key = record.Key>({
  value,
  onSelect = NOOP,
  clear = NOOP,
  setSelected = NOOP,
  children,
  hover,
}: ProviderProps<K>): ReactElement => {
  const valueRef = useRef(array.toArray(value));
  const hoverRef = useSyncedRef(hover);

  const { notifyListeners, subscribe } = Store.useKeyedListeners<K>();

  const getState = useCallback(
    () => ({ value: valueRef.current, hover: hoverRef.current }),
    [],
  );
  const ctx = useMemo(
    () => ({
      onSelect,
      setSelected,
      clear,
      hover,
      subscribe,
      getState,
    }),
    [getState, onSelect, setSelected, clear, hover, subscribe],
  );
  useEffect(() => {
    const prev = valueRef.current;
    const next = array.toArray(value);
    const prevSet = new Set(prev);
    const nextSet = new Set(next);
    const changed: K[] = [];
    for (const k of prev) if (!nextSet.has(k)) changed.push(k);
    for (const k of next) if (!prevSet.has(k)) changed.push(k);
    valueRef.current = next;
    if (changed.length > 0) notifyListeners(changed);
  }, [value, notifyListeners]);

  return (
    <Context value={ctx as unknown as ContextValue<record.Key>}>{children}</Context>
  );
};

export interface UseItemStateReturn {
  selected: boolean;
  hovered: boolean;
  onSelect: () => void;
}

export const useContext = <K extends record.Key = record.Key>(): ContextValue<K> =>
  useCtx() as unknown as ContextValue<K>;

type ItemState = "none" | "selected" | "hovered" | "selected-hovered";

export const useItemState = <K extends record.Key>(key: K): UseItemStateReturn => {
  const { getState, onSelect, subscribe } = useContext();
  const handleSelect = useCallback(() => onSelect(key), [key, onSelect]);
  const itemState = useSyncExternalStore(
    useCallback((onStoreChange) => subscribe(onStoreChange, key), [key, subscribe]),
    useCallback((): ItemState => {
      const state = getState();
      const selected = isSelected(state.value, key);
      const hovered = state.hover === key;
      if (selected && hovered) return "selected-hovered";
      if (selected) return "selected";
      if (hovered) return "hovered";
      return "none";
    }, [key, getState]),
    useCallback((): ItemState => "none", []),
  );
  return useMemo(
    () => ({
      selected: itemState === "selected" || itemState === "selected-hovered",
      hovered: itemState === "hovered" || itemState === "selected-hovered",
      onSelect: handleSelect,
    }),
    [itemState, handleSelect],
  );
};

export const useSelection = <K extends record.Key>(): K[] => {
  const { getState, subscribe } = useContext<K>();
  const res = useSyncExternalStore(
    subscribe,
    () => getState().value,
    () => null,
  );
  return useMemo((): K[] => {
    if (res == null) return [];
    return array.toArray(res);
  }, [res]);
};

export const useClear = () => useContext().clear;
