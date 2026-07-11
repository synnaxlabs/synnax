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
  useLayoutEffect,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";

import { context } from "@/context";
import { Store } from "@/store";

type Value<K extends record.Key = record.Key> = K | K[] | undefined;

interface SelectionState<K extends record.Key = record.Key> {
  value: Value<K>;
  hover?: K;
}

const [BaseContext, useCtx] = context.create<ContextValue>({
  defaultValue: {
    clear: () => {},
    getState: () => ({ value: undefined, hover: undefined }),
    onSelect: () => {},
    setSelected: () => {},
    subscribe: () => () => {},
  },
  displayName: "Selection.Context",
});

const isSelected = <K extends record.Key>(value: Value<K>, key: K): boolean => {
  if (value === undefined) return false;
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

export interface ContextProps<K extends record.Key = record.Key>
  extends
    PropsWithChildren,
    Partial<Pick<ContextValue<K>, "onSelect" | "setSelected" | "clear">>,
    SelectionState<K> {}

const NOOP = () => {};

// Context distributes a controlled selection to keyed item consumers. It diffs
// value changes and notifies only the listeners whose keys entered or left the
// selection, so item re-renders stay surgical via useItemState.
export const Context = <K extends record.Key = record.Key>({
  value,
  onSelect = NOOP,
  clear = NOOP,
  setSelected = NOOP,
  children,
  hover,
}: ContextProps<K>): ReactElement => {
  const valueRef = useRef(value);
  const hoverRef = useRef(hover);

  const { notifyListeners, subscribe } = Store.useKeyedListeners<K>();

  const getState = useCallback(
    () => ({ value: valueRef.current, hover: hoverRef.current }),
    [],
  );
  const ctx = useMemo<ContextValue<K>>(
    () => ({
      onSelect,
      setSelected,
      clear,
      subscribe,
      getState,
    }),
    [getState, onSelect, setSelected, clear, subscribe],
  );
  useLayoutEffect(() => {
    const changed = new Set(array.toArray(valueRef.current)).symmetricDifference(
      new Set(array.toArray(value)),
    );
    valueRef.current = value;
    const prevHover = hoverRef.current;
    if (prevHover !== hover) {
      if (prevHover != null) changed.add(prevHover);
      if (hover != null) changed.add(hover);
      hoverRef.current = hover;
    }
    if (changed.size > 0) notifyListeners(Array.from(changed));
  }, [value, hover, notifyListeners]);

  return (
    <BaseContext value={ctx as unknown as ContextValue<record.Key>}>
      {children}
    </BaseContext>
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

// useItemState subscribes a single keyed item to the enclosing Provider,
// re-rendering only when that key's selected or hovered state flips.
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

// useSelected returns the currently selected keys.
export const useSelected = <K extends record.Key>(): K[] => {
  const { getState, subscribe } = useContext<K>();
  const res = useSyncExternalStore(
    subscribe,
    () => getState().value,
    () => null,
  );
  return useMemo((): K[] => (res == null ? [] : array.toArray(res)), [res]);
};

export const useClear = () => useContext().clear;
