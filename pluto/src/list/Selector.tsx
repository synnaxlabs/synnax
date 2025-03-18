// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, type Keyed, nullToArr } from "@synnaxlabs/x";
import {
  createContext,
  memo,
  type PropsWithChildren,
  type ReactElement,
  useContext,
  useMemo,
} from "react";

import { useSyncedRef } from "@/hooks";
import { useGetTransformedData } from "@/list/Data";
import { useSelect, type UseSelectProps } from "@/list/useSelect";

interface SelectContextValue<K extends Key = Key> {
  selected: K[];
}

interface SelectUtilsContextValue<K extends Key = Key> {
  onSelect: (key: K) => void;
  clear: () => void;
  getSelected: () => K[];
}

export type SelectorProps<
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
> = PropsWithChildren<UseSelectProps<K, E>>;

const Context = createContext<SelectContextValue>({ selected: [] });

const UtilsContext = createContext<SelectUtilsContextValue>({
  onSelect: () => {},
  clear: () => {},
  getSelected: () => [],
});

export const useSelectionContext = <K extends Key = Key>() =>
  useContext(Context) as SelectContextValue<K>;

export const useSelection = <K extends Key = Key>() =>
  useSelectionContext<K>().selected;

export const useSelectionUtils = <K extends Key = Key>() =>
  useContext(UtilsContext) as unknown as SelectUtilsContextValue<K>;

/**
 * Implements selection behavior for a list.
 *
 * @param props - The props for the List.Selector component. These props are identical
 * to the props for {@link useSelect} hook.
 */
const Base = memo(
  <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
    value,
    children,
    ...rest
  }: SelectorProps<K, E>): ReactElement => {
    const getData = useGetTransformedData<K, E>();
    const { onSelect, clear } = useSelect<K, E>({
      ...rest,
      value,
      data: getData,
    } as const as UseSelectProps<K, E>);
    const selectedRef = useSyncedRef(value);
    const ctxValue: SelectContextValue<K> = useMemo(
      () => ({ selected: nullToArr(value) }),
      [value],
    );
    const utilCtxValue: SelectUtilsContextValue<K> = useMemo(
      () => ({ onSelect, clear, getSelected: () => nullToArr(selectedRef.current) }),
      [onSelect, clear],
    );
    return (
      <UtilsContext value={utilCtxValue as unknown as SelectUtilsContextValue}>
        <Context value={ctxValue}>{children}</Context>
      </UtilsContext>
    );
  },
);
Base.displayName = "List.Selector";

export const Selector = Base as <K extends Key = Key, E extends Keyed<K> = Keyed<K>>(
  props: SelectorProps<K, E>,
) => ReactElement;
