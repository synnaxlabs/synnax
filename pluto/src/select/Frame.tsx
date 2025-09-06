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
  useContext as reactUseContext,
  useEffect,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";

import { useSyncedRef } from "@/hooks/ref";
import { List } from "@/list";
import {
  useMultiple,
  type UseMultipleProps,
  useSingle,
  type UseSingleProps,
} from "@/select/use";
import { Store } from "@/store";

interface SelectionState<K extends record.Key = record.Key> {
  value: K | K[] | null | undefined;
  hover?: K;
}

const Context = createContext<ContextValue<any>>({
  getState: () => ({ value: undefined, hover: undefined }),
  onSelect: () => {},
  setSelected: () => {},
  clear: () => {},
  subscribe: () => () => {},
});

const isSelected = <K extends record.Key>(
  value: K | K[] | null | undefined,
  key: K,
): boolean => {
  if (value === null || value === undefined) return false;
  if (Array.isArray(value)) return value.includes(key);
  return value === key;
};

interface ContextValue<K extends record.Key = record.Key>
  extends Pick<Store.UseKeyedListenersReturn<K>, "subscribe"> {
  onSelect: (key: K) => void;
  setSelected: (keys: K[]) => void;
  clear: () => void;
  getState: () => SelectionState<K>;
}

const MultipleProvider = <K extends record.Key = record.Key>({
  children,
  ...rest
}: UseMultipleProps<K> & PropsWithChildren): ReactElement => {
  const { value } = rest;
  const res = useMultiple(rest);
  return (
    <Provider value={value} {...res}>
      {children}
    </Provider>
  );
};

const SingleProvider = <K extends record.Key = record.Key>({
  children,
  ...rest
}: UseSingleProps<K> & PropsWithChildren): ReactElement => {
  const { value } = rest;
  const res = useSingle(rest);
  return (
    <Provider<K> value={value} {...res}>
      {children}
    </Provider>
  );
};

interface ProviderProps<K extends record.Key = record.Key>
  extends Omit<ContextValue<K>, "getState" | "subscribe">,
    PropsWithChildren {
  value: K | K[] | null | undefined;
  hover?: K;
}

const Provider = <K extends record.Key = record.Key>({
  value,
  onSelect,
  clear,
  setSelected,
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
    const prevValue = valueRef.current;
    const nextValue = array.toArray(value);
    const notify = [...prevValue, ...nextValue];
    valueRef.current = nextValue;
    notifyListeners(notify);
  }, [value, notifyListeners]);

  return <Context.Provider value={ctx}>{children}</Context.Provider>;
};

export interface UseItemStateReturn {
  selected: boolean;
  hovered: boolean;
  onSelect: () => void;
}

export const useContext = <K extends record.Key = record.Key>(): ContextValue<K> =>
  reactUseContext(Context) as unknown as ContextValue<K>;

enum ItemState {
  NONE = 0,
  SELECTED = 1,
  HOVERED = 2,
  SELECTED_HOVERED = 3,
}

export const useItemState = <K extends record.Key>(key: K): UseItemStateReturn => {
  const { getState, onSelect, subscribe } = useContext();
  const handleSelect = useCallback(() => onSelect(key), [key, onSelect]);
  const selected = useSyncExternalStore(
    useCallback((onStoreChange) => subscribe(onStoreChange, key), [key, subscribe]),
    useCallback(() => {
      const state = getState();
      const selected = isSelected(state.value, key);
      const hovered = state.hover === key;
      if (selected && hovered) return ItemState.SELECTED_HOVERED;
      if (selected) return ItemState.SELECTED;
      if (hovered) return ItemState.HOVERED;
      return ItemState.NONE;
    }, [key, getState]),
  );
  return {
    selected:
      selected === ItemState.SELECTED || selected === ItemState.SELECTED_HOVERED,
    hovered: selected === ItemState.HOVERED || selected === ItemState.SELECTED_HOVERED,
    onSelect: handleSelect,
  };
};

export const useSelection = <K extends record.Key>(): K[] => {
  const { getState, subscribe } = useContext();
  const res = useSyncExternalStore(subscribe, () => getState().value);
  if (res == null) return [];
  return array.toArray(res) as K[];
};

export const useClear = () => useContext().clear;

export interface TriggerProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> {
  value: K | null;
  useItem: (key: K) => E;
  onClick: () => void;
}

interface BaseFrameProps<K extends record.Key, E extends record.Keyed<K> | undefined>
  extends Omit<List.FrameProps<K, E>, "onChange"> {}

export interface MultipleFrameProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> extends BaseFrameProps<K, E>,
    UseMultipleProps<K> {
  multiple: true;
}

export interface SingleFrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> extends BaseFrameProps<K, E>,
    UseSingleProps<K> {
  multiple?: false;
}

export type FrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
> = MultipleFrameProps<K, E> | SingleFrameProps<K, E>;

export const Frame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  data,
  getItem,
  subscribe,
  itemHeight,
  value,
  onChange,
  multiple,
  onFetchMore,
  virtual = false,
  ...rest
}: FrameProps<K, E>): ReactElement => {
  let child: ReactElement;
  if (multiple === true)
    child = <MultipleProvider value={value} onChange={onChange} {...rest} />;
  else child = <SingleProvider value={value} onChange={onChange} {...rest} />;
  return (
    <List.Frame<K, E>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={onFetchMore}
      itemHeight={itemHeight}
      virtual={virtual}
    >
      {child}
    </List.Frame>
  );
};
