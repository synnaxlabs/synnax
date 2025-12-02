// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type location, type record } from "@synnaxlabs/x";
import { useVirtualizer, type Virtualizer } from "@tanstack/react-virtual";
import {
  memo,
  type PropsWithChildren,
  type ReactElement,
  type RefCallback,
  type RefObject,
  useCallback,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";

import { context } from "@/context";
import { Dialog } from "@/dialog";
import { useSyncedRef } from "@/hooks";

/**
 * Function interface for getting items from a list by key(s).
 *
 * @template K The type of the key (must be a record key)
 * @template E The type of the entity (must be keyed by K)
 */
export interface GetItem<K extends record.Key, E extends record.Keyed<K> | undefined>
  extends GetSingleItem<K, E>,
    GetMultipleItems<K, E> {}

export interface GetSingleItem<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> {
  (key: K): E | undefined;
}

export interface GetMultipleItems<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> {
  (keys: K[]): E[];
}

export const createGetItem = <
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
>(
  first: GetSingleItem<K, E>,
  second: GetMultipleItems<K, E>,
): GetItem<K, E> =>
  ((key: K | K[]) => {
    if (Array.isArray(key)) return second(key);
    return first(key);
  }) as GetItem<K, E>;

export interface ItemSpec<K extends record.Key = record.Key> {
  key: K;
  index: number;
  translate?: number;
}

export interface DataContextValue<K extends record.Key = record.Key> {
  data: K[];
  getItems: () => ItemSpec<K>[];
  getTotalSize: () => number | undefined;
  itemHeight?: number;
}

export interface UtilContextValue<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> {
  ref: RefCallback<HTMLDivElement | null>;
  getItem?: GetItem<K, E>;
  subscribe?: (callback: () => void, key: K) => () => void;
  scrollToIndex: (index: number, direction?: location.Y) => void;
}

const [DataContext, useDataContext] = context.create<DataContextValue>({
  displayName: "List.DataContext",
  providerName: "List.Frame",
});

const [UtilContext, useUtilCtx] = context.create<UtilContextValue>({
  displayName: "List.UtilContext",
  providerName: "List.Frame",
});

export const useUtilContext = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(): UtilContextValue<K, E> =>
  useUtilCtx("List.useUtilContext") as unknown as UtilContextValue<K, E>;

export interface FrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> extends PropsWithChildren,
    Pick<UtilContextValue<K, E>, "getItem" | "subscribe"> {
  data: K[];
  virtual?: boolean;
  overscan?: number;
  itemHeight?: number;
  onFetchMore?: () => void;
}

export const useScroller = <K extends record.Key = record.Key>(): Pick<
  UtilContextValue<K>,
  "scrollToIndex"
> => {
  const { scrollToIndex } = useUtilCtx("List.useScroller");
  return useMemo(() => ({ scrollToIndex }), [scrollToIndex]);
};

export const useItem = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(
  key: K,
): E | undefined => {
  const { getItem, subscribe } = useUtilCtx(
    "List.useItem",
  ) as unknown as UtilContextValue<K, E>;
  return useSyncExternalStore(
    useCallback(
      (callback) => {
        if (subscribe == null) return () => {};
        return subscribe(callback, key);
      },
      [key, subscribe],
    ),
    useCallback(() => getItem?.(key), [getItem, key]),
  );
};

export const useData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(): DataContextValue<K> & UtilContextValue<K, E> => {
  const { data, getItems, getTotalSize, itemHeight } = useDataContext(
    "List.useData",
  ) as DataContextValue<K>;
  const { ref, getItem, scrollToIndex, subscribe } = useUtilCtx(
    "List.useData",
  ) as unknown as UtilContextValue<K, E>;
  return useMemo(
    () => ({
      data,
      getItems,
      getTotalSize,
      ref,
      getItem,
      scrollToIndex,
      subscribe,
      itemHeight,
    }),
    [data, getItems, getTotalSize, ref, getItem, scrollToIndex, subscribe, itemHeight],
  );
};

const useFetchMoreRefCallback = (
  elRef: RefObject<HTMLDivElement | null>,
  hasData: boolean,
  onFetchMore?: () => void,
) => {
  const onFetchMoreRef = useSyncedRef(onFetchMore);
  const { visible } = Dialog.useContext();
  const initialFetchCalledRef = useRef(false);
  return useCallback(
    (el: HTMLDivElement) => {
      elRef.current = el;
      if (elRef.current == null || initialFetchCalledRef.current) return;
      initialFetchCalledRef.current = true;
      onFetchMoreRef.current?.();
    },
    [onFetchMoreRef, visible, hasData],
  );
};

const VirtualFrame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>({
  data,
  getItem,
  subscribe,
  children,
  onFetchMore,
  overscan = 10,
  itemHeight = 36,
}: FrameProps<K, E>): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  const hasData = data.length > 0;
  const refCallback = useFetchMoreRefCallback(ref, hasData, onFetchMore);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => ref.current,
    estimateSize: () => itemHeight,
    overscan,
    onChange: useCallback(
      (v: Virtualizer<HTMLDivElement, HTMLDivElement>) => {
        const items = v.getVirtualItems();
        if (items.length > 0 && items[items.length - 1].index === data.length - 1)
          onFetchMore?.();
      },
      [data.length, onFetchMore],
    ),
  });

  const items = virtualizer.getVirtualItems();
  const dataCtxValue = useMemo<DataContextValue<K>>(
    () => ({
      ref: refCallback,
      getItem,
      data,
      subscribe,
      getTotalSize: () => virtualizer.getTotalSize(),
      getItems: () =>
        items.map(({ index, start }) => ({
          key: data[index],
          index,
          translate: start,
        })),
      itemHeight,
    }),
    [refCallback, virtualizer, data, getItem, itemHeight, items],
  );

  const utilCtxValue = useMemo<UtilContextValue<K, E>>(
    () => ({
      ref: refCallback,
      getItem,
      scrollToIndex: (index) => virtualizer.scrollToIndex(index),
      subscribe,
    }),
    [refCallback, virtualizer, getItem, subscribe],
  );

  return (
    <DataContext value={dataCtxValue}>
      <UtilContext value={utilCtxValue as unknown as UtilContextValue}>
        {children}
      </UtilContext>
    </DataContext>
  );
};

const StaticFrame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>({
  data,
  getItem,
  subscribe,
  children,
  onFetchMore,
  itemHeight,
}: FrameProps<K, E>): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  const hasData = data.length > 0;
  const scrollToIndex = useCallback((index: number, direction?: location.Y) => {
    const container = ref.current?.children[0];
    if (!container) return;
    const dirMultiplier = direction === "top" ? 1 : -1;
    let scrollTo: number;
    const idealHover = index + dirMultiplier;
    if (bounds.contains({ lower: 0, upper: container.children.length }, idealHover))
      scrollTo = index + dirMultiplier;
    else scrollTo = index;
    const child = container.children[scrollTo] as HTMLElement | undefined;
    if (child != null)
      child.scrollIntoView({ block: "nearest", inline: "nearest", behavior: "smooth" });
  }, []);

  const refCallback = useFetchMoreRefCallback(ref, hasData, onFetchMore);

  const items = data.map((key, index) => ({ key, index }));
  const dataCtxValue = useMemo<DataContextValue<K>>(
    () => ({
      ref: refCallback,
      getItem,
      data,
      subscribe,
      getTotalSize: () => undefined,
      getItems: () => items,
      itemHeight,
    }),
    [refCallback, data, getItem, subscribe, itemHeight],
  );
  const utilCtxValue = useMemo<UtilContextValue<K, E>>(
    () => ({
      ref: refCallback,
      getItem,
      scrollToIndex,
      subscribe,
    }),
    [refCallback, getItem, subscribe, scrollToIndex],
  );
  return (
    <DataContext value={dataCtxValue}>
      <UtilContext value={utilCtxValue as unknown as UtilContextValue}>
        {children}
      </UtilContext>
    </DataContext>
  );
};

export const CoreFrame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>({
  virtual = false,
  ...rest
}: FrameProps<K, E>): ReactElement =>
  virtual ? <VirtualFrame {...rest} /> : <StaticFrame {...rest} />;

export const Frame = memo(CoreFrame) as typeof CoreFrame;
