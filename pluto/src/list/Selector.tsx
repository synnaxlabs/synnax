// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  type PropsWithChildren,
  useContext,
  useMemo,
  type ReactElement,
} from "react";

import { type Key, type Keyed, nullToArr } from "@synnaxlabs/x";

import { useSyncedRef } from "@/hooks";
import { useGetTransformedData } from "@/list/Data";
import { useSelect, type UseSelectProps } from "@/list/useSelect";

interface SelectContextValue<K extends Key = Key> {
  selected: K[];
}

interface SelectUtilContextValue<K extends Key = Key> {
  onSelect: (key: K) => void;
  clear: () => void;
  getSelected: () => K[];
}

export type SelectorProps<
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
> = PropsWithChildren<Omit<UseSelectProps<K, E>, "data">>;

const SelectionContext = createContext<SelectContextValue>({
  selected: [],
});

const SelectionUtilContext = createContext<SelectUtilContextValue>({
  onSelect: () => {},
  clear: () => {},
  getSelected: () => [],
});

export const useSelectionContext = <K extends Key = Key>(): SelectContextValue<K> =>
  useContext(SelectionContext) as unknown as SelectContextValue<K>;

export const useSelection = <K extends Key = Key>(): K[] =>
  useSelectionContext<K>().selected;

export const useSelectionUtils = <K extends Key = Key>(): SelectUtilContextValue<K> =>
  useContext(SelectionUtilContext) as unknown as SelectUtilContextValue<K>;

/**
 * Implements selection behavior for a list.
 *
 * @param props - The props for the List.Selector component. These props are identical
 * to the props for {@link useSelect} hook.
 */
export const Selector = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  value,
  children,
  ...props
}: SelectorProps<K, E>): ReactElement => {
  const getData = useGetTransformedData<K, E>();
  const { onSelect, clear } = useSelect<K, E>({
    data: getData,
    value,
    ...props,
  } as const as UseSelectProps<K, E>);
  const selectedRef = useSyncedRef(value);
  const ctxValue: SelectContextValue<K> = useMemo(
    () => ({ selected: nullToArr(value) }),
    [value],
  );
  const utilCtxValue: SelectUtilContextValue<K> = useMemo(
    () => ({
      onSelect,
      clear,
      getSelected: () => nullToArr(selectedRef.current),
    }),
    [onSelect, clear],
  );
  return (
    <SelectionUtilContext.Provider
      value={utilCtxValue as unknown as SelectUtilContextValue}
    >
      <SelectionContext.Provider value={ctxValue as unknown as SelectContextValue}>
        {children}
      </SelectionContext.Provider>
    </SelectionUtilContext.Provider>
  );
};
