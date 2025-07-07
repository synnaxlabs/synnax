// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/List.css";

import { type record } from "@synnaxlabs/x";
import { useVirtualizer, type Virtualizer } from "@tanstack/react-virtual";
import {
  type ComponentPropsWithoutRef,
  createContext,
  type PropsWithChildren,
  type ReactElement,
  type RefObject,
  useCallback,
  useMemo,
  useRef,
} from "react";

import { CSS } from "@/css";
import { useRequiredContext } from "@/hooks";
import { type ItemRenderProp } from "@/list/Item";

export interface UseProps<K extends record.Key = record.Key> {
  data: K[];
  itemHeight?: number;
}

export interface UseReturn {
  ref: RefObject<HTMLDivElement | null>;
  virtualizer: Virtualizer<HTMLDivElement, Element>;
}

export interface ItemsProps<K extends record.Key = record.Key>
  extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  children: ItemRenderProp<K>;
  emptyContent?: ReactElement;
}

export const use = <K extends record.Key = record.Key>({
  data,
  itemHeight = 30,
}: UseProps<K>): UseReturn => {
  const ref = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => ref.current,
    estimateSize: () => itemHeight,
  });
  return { ref, virtualizer };
};

export interface ContextValue<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> {
  ref: RefObject<HTMLDivElement | null>;
  virtualizer: Virtualizer<HTMLDivElement, Element>;
  onFetchMore?: () => void;
  data: K[];
  useItem: (key?: K) => E | undefined;
}

const Context = createContext<ContextValue | null>(null);

export const useContext = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(): ContextValue<K, E> => useRequiredContext(Context) as unknown as ContextValue<K, E>;

export const useItem = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(
  key?: K,
): E | undefined => {
  const { useItem } = useContext<K, E>();
  return useItem(key);
};

export interface ListProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> extends PropsWithChildren,
    ContextValue<K, E> {}

export const List = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>({
  children,
  ...rest
}: ListProps<K, E>): ReactElement => (
  <Context.Provider value={rest as unknown as ContextValue}>
    {children}
  </Context.Provider>
);

export const Items = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  className,
  children,
  emptyContent,
  ...rest
}: ItemsProps<K>): ReactElement => {
  const { ref, virtualizer, data } = useContext<K, E>();
  const visibleData = virtualizer.getVirtualItems();
  let content = emptyContent;
  if (data.length > 0)
    content = (
      <div
        className={CSS.BE("list", "virtualizer")}
        style={{ height: virtualizer.getTotalSize() }}
      >
        {visibleData.map(({ index, start }) => {
          const key = data[index];
          return children({ key, index, translate: start, itemKey: key });
        })}
      </div>
    );
  return (
    <div ref={ref} className={CSS(className, CSS.BE("list", "items"))} {...rest}>
      {content}
    </div>
  );
};

export interface UseStaticDataReturn<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> {
  useItem: (key?: K) => E | undefined;
  data: K[];
}

export const useStaticData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>(
  data: E[],
): UseStaticDataReturn<K, E> =>
  useMemo(() => {
    const keys = data.map((d) => d.key);
    const useItem = useCallback((key?: K) => data.find((d) => d.key === key), [data]);
    return { useItem, data: keys };
  }, [data]);
