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

import { type Dialog } from "@/dialog";
import { useRequiredContext } from "@/hooks";
import { List } from "@/list";
import { type SingleTriggerProps } from "@/select/SingleTrigger";
import {
  useMultiple,
  type UseMultipleProps,
  useSingle,
  type UseSingleProps,
} from "@/select/use";

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
  onSelect: (...keys: K[]) => void;
  clear: () => void;
  hover?: K;
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
  extends Omit<ContextValue<K>, "value">,
    PropsWithChildren {
  value: K | K[] | null | undefined;
}

const Provider = <K extends record.Key = record.Key>({
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

export const useContext = <K extends record.Key = record.Key>(): ContextValue<K> =>
  useRequiredContext(Context) as unknown as ContextValue<K>;

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

export interface MultipleProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> extends Omit<MultipleFrameProps<K, E>, "multiple" | "data" | "useListItem">,
    Pick<List.ItemsProps<K>, "emptyContent">,
    Omit<Dialog.FrameProps, "onChange" | "children"> {}

export interface SingleFrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> extends BaseFrameProps<K, E>,
    UseSingleProps<K> {
  multiple?: false;
}

export interface SingleProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> extends Omit<SingleFrameProps<K, E>, "multiple" | "data" | "useListItem">,
    Pick<List.ItemsProps<K>, "emptyContent">,
    Omit<Dialog.FrameProps, "onChange" | "children">,
    Pick<SingleTriggerProps, "disabled" | "placeholder" | "icon"> {}

export type FrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
> = MultipleFrameProps<K, E> | SingleFrameProps<K, E>;

export const Frame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  data,
  useListItem,
  itemHeight,
  value,
  onChange,
  multiple,
  onFetchMore,
  ...rest
}: FrameProps<K, E>): ReactElement => {
  let child: ReactElement;
  if (multiple === true)
    child = <MultipleProvider value={value} onChange={onChange} {...rest} />;
  else child = <SingleProvider value={value} onChange={onChange} {...rest} />;
  return (
    <List.Frame<K, E>
      data={data}
      useListItem={useListItem}
      onFetchMore={onFetchMore}
      itemHeight={itemHeight}
    >
      {child}
    </List.Frame>
  );
};
