// Copyright 2026 Synnax Labs, Inc.
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
  type PropsWithChildren,
  type ReactElement,
  type RefCallback,
  type RefObject,
  useCallback,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";

import { memo } from "@/component/memo";
import { context } from "@/context";
import { Dialog } from "@/dialog";
import { useCombinedRefs, usePrevious, useSyncedRef } from "@/hooks";

/** Function interface for getting items from a list by key(s). */
export interface GetItem<K extends record.Key, E extends record.Keyed<K> | undefined>
  extends GetSingleItem<K, E>, GetMultipleItems<K, E> {}

/** Reads one item by key. */
export interface GetSingleItem<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> {
  (key: K): E | undefined;
}

/** Reads many items at once, dropping any key with no item. */
export interface GetMultipleItems<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> {
  (keys: K[]): E[];
}

/** Joins a single-key and a multi-key reader into one {@link GetItem}. */
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

/** One item the enclosing frame asks its children to render. */
export interface ItemSpec<K extends record.Key = record.Key> {
  key: K;
  index: number;
  /** Pixel offset from the top of the list, set only when virtualized. */
  translate?: number;
}

export interface DataContextValue<K extends record.Key = record.Key> {
  data: K[];
  getItems: () => ItemSpec<K>[];
  getTotalSize: () => number | undefined;
  sentinelRef?: RefCallback<HTMLDivElement>;
}

export interface UtilContextValue<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> {
  ref: RefCallback<HTMLDivElement | null>;
  getItem?: GetItem<K, E>;
  subscribe?: (callback: () => void, key: K) => () => void;
  scrollToIndex: (index: number, direction?: location.Y) => void;
  itemHeight?: number;
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

/** Props for {@link Frame}. A data hook such as `useStaticData` supplies most of them. */
export interface FrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>
  extends PropsWithChildren, Pick<UtilContextValue<K, E>, "getItem" | "subscribe"> {
  /** The keys to render, in order. */
  data: K[];
  /** Whether to render only the visible window. Needed above a few hundred items. */
  virtual?: boolean;
  /** Extra items to render past each edge of the visible window. */
  overscan?: number;
  /** Row height in pixels. Virtualization estimates from it. */
  itemHeight?: number;
  /** Called when the list scrolls near its end. */
  onFetchMore?: () => void;
}

/** @returns a scroller for the enclosing {@link Frame}, stable as the list scrolls. */
export const useScroller = <K extends record.Key = record.Key>(): Pick<
  UtilContextValue<K>,
  "scrollToIndex"
> => {
  const { scrollToIndex } = useUtilCtx("List.useScroller");
  return useMemo(() => ({ scrollToIndex }), [scrollToIndex]);
};

/**
 * useItemHeight returns the row height the enclosing Frame was given. It reads the
 * util context, which does not change as the list scrolls.
 */
export const useItemHeight = (): number | undefined =>
  useUtilCtx("List.useItemHeight").itemHeight;

/**
 * Reads the item for a key from the enclosing {@link Frame} and re-renders the caller
 * when that one item changes. Use it inside a list item, so the list does not re-render
 * on every entry update.
 */
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
    () => undefined as E | undefined,
  );
};

/**
 * Reads the full state of the enclosing {@link Frame}: the keys, the visible window,
 * and the item readers. It re-renders on every scroll, so prefer {@link useItem} inside
 * an item.
 */
export const useData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(): DataContextValue<K> & UtilContextValue<K, E> => {
  const { data, getItems, getTotalSize, sentinelRef } = useDataContext(
    "List.useData",
  ) as DataContextValue<K>;
  const { ref, getItem, scrollToIndex, subscribe, itemHeight } = useUtilCtx(
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
      sentinelRef,
    }),
    [
      data,
      getItems,
      getTotalSize,
      ref,
      getItem,
      scrollToIndex,
      subscribe,
      itemHeight,
      sentinelRef,
    ],
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

interface UseIntersectionFetchMoreReturn {
  containerRef: RefCallback<HTMLDivElement>;
  sentinelRef: RefCallback<HTMLDivElement>;
}

const SCROLL_THRESHOLD_PX = 100;

const useIntersectionFetchMore = (
  onFetchMore: (() => void) | undefined,
  dataLength: number,
): UseIntersectionFetchMoreReturn => {
  const onFetchMoreRef = useSyncedRef(onFetchMore);
  const isFetchingRef = useRef(false);
  const observerRef = useRef<IntersectionObserver | null>(null);
  const containerElRef = useRef<HTMLDivElement | null>(null);
  const sentinelElRef = useRef<HTMLDivElement | null>(null);

  const prevDataLength = usePrevious(dataLength);
  if (prevDataLength !== undefined && dataLength !== prevDataLength)
    isFetchingRef.current = false;

  const setupObserver = useCallback(() => {
    observerRef.current?.disconnect();
    observerRef.current = null;

    const container = containerElRef.current;
    const sentinel = sentinelElRef.current;
    if (container == null || sentinel == null) return;

    observerRef.current = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && !isFetchingRef.current) {
          isFetchingRef.current = true;
          onFetchMoreRef.current?.();
        }
      },
      {
        root: container,
        rootMargin: `0px 0px ${SCROLL_THRESHOLD_PX}px 0px`,
        threshold: 0,
      },
    );
    observerRef.current.observe(sentinel);
  }, [onFetchMoreRef]);

  const containerRef = useCallback(
    (el: HTMLDivElement | null) => {
      containerElRef.current = el;
      setupObserver();
    },
    [setupObserver],
  );

  const sentinelRef = useCallback(
    (el: HTMLDivElement | null) => {
      sentinelElRef.current = el;
      setupObserver();
    },
    [setupObserver],
  );

  return { containerRef, sentinelRef };
};

const INITIAL_WINDOW_HEIGHT = 800;

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
  itemHeight = 33,
}: FrameProps<K, E>): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  const hasData = data.length > 0;
  const refCallback = useFetchMoreRefCallback(ref, hasData, onFetchMore);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => ref.current,
    estimateSize: () => itemHeight,
    getItemKey: useCallback((index: number) => data[index] ?? index, [data]),
    // The container has no measured rect until an effect runs, and an unmeasured
    // window renders nothing. Assuming one keeps the mount commit from painting empty.
    initialRect: { width: 0, height: INITIAL_WINDOW_HEIGHT },
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
    }),
    [refCallback, virtualizer, data, getItem, items],
  );

  const utilCtxValue = useMemo<UtilContextValue<K, E>>(
    () => ({
      ref: refCallback,
      getItem,
      scrollToIndex: (index) => virtualizer.scrollToIndex(index),
      subscribe,
      itemHeight,
    }),
    [refCallback, virtualizer, getItem, subscribe, itemHeight],
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
    if (container == null) return;
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

  const initialFetchCallback = useFetchMoreRefCallback(ref, hasData, onFetchMore);
  const { containerRef: intersectionContainerRef, sentinelRef } =
    useIntersectionFetchMore(onFetchMore, data.length);
  const refCallback = useCombinedRefs(initialFetchCallback, intersectionContainerRef);

  const items = useMemo(() => data.map((key, index) => ({ key, index })), [data]);
  const dataCtxValue = useMemo<DataContextValue<K>>(
    () => ({
      ref: refCallback,
      getItem,
      data,
      subscribe,
      getTotalSize: () => undefined,
      getItems: () => items,
      sentinelRef,
    }),
    [refCallback, data, getItem, subscribe, sentinelRef, items],
  );
  const utilCtxValue = useMemo<UtilContextValue<K, E>>(
    () => ({
      ref: refCallback,
      getItem,
      scrollToIndex,
      subscribe,
      itemHeight,
    }),
    [refCallback, getItem, subscribe, scrollToIndex, itemHeight],
  );
  return (
    <DataContext value={dataCtxValue}>
      <UtilContext value={utilCtxValue as unknown as UtilContextValue}>
        {children}
      </UtilContext>
    </DataContext>
  );
};

/** {@link Frame} before memoization. Prefer `Frame`. */
export const BaseFrame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>({
  virtual = false,
  ...rest
}: FrameProps<K, E>): ReactElement =>
  virtual ? <VirtualFrame {...rest} /> : <StaticFrame {...rest} />;

/**
 * Holds the data for a list and hands it to its children through context. It renders no
 * element of its own: pair it with {@link Items} for the scroll container.
 *
 * @example
 * <List.Frame {...List.useStaticData({ data })}>
 *   <List.Items>{(p) => <List.Item {...p} />}</List.Items>
 * </List.Frame>
 */
export const Frame = memo(BaseFrame);
