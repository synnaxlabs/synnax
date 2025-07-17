import { type record } from "@synnaxlabs/x";
import { useVirtualizer, type Virtualizer } from "@tanstack/react-virtual";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  type RefCallback,
  useCallback,
  useMemo,
  useRef,
  useSyncExternalStore,
} from "react";

import { Dialog } from "@/dialog";
import { useRequiredContext, useSyncedRef } from "@/hooks";

export interface ItemSpec<K extends record.Key = record.Key> {
  key: K;
  index: number;
  translate: number;
}

export interface DataContextValue<K extends record.Key = record.Key> {
  data: K[];
  getItems: () => ItemSpec<K>[];
  getTotalSize: () => number;
}

export interface UtilContextValue<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> {
  ref: RefCallback<HTMLDivElement | null>;
  getItem?: (key?: K) => E | undefined;
  subscribe?: (callback: () => void, key?: K) => () => void;
  scrollToIndex: (index: number) => void;
}

const DataContext = createContext<DataContextValue | null>(null);
const UtilContext = createContext<UtilContextValue | null>(null);

export interface FrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> extends PropsWithChildren,
    Pick<UtilContextValue<K, E>, "getItem" | "subscribe"> {
  data: K[];
  itemHeight?: number;
  onFetchMore?: () => void;
}

const useDataContext = <K extends record.Key = record.Key>(): DataContextValue<K> =>
  useRequiredContext(DataContext) as unknown as DataContextValue<K>;

const useUtilContext = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(): UtilContextValue<K, E> =>
  useRequiredContext(UtilContext) as unknown as UtilContextValue<K, E>;

export const useScroller = <K extends record.Key = record.Key>(): Pick<
  UtilContextValue<K>,
  "scrollToIndex"
> => {
  const { scrollToIndex } = useUtilContext();
  return { scrollToIndex };
};

export const useItem = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(
  key?: K,
): E | undefined => {
  const { getItem, subscribe } = useUtilContext<K, E>();
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
  const { data, getItems, getTotalSize } = useDataContext<K>();
  const { ref, getItem, scrollToIndex, subscribe } = useUtilContext<K, E>();
  return { data, getItems, getTotalSize, ref, getItem, scrollToIndex, subscribe };
};

export const Frame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>({
  data,
  getItem,
  subscribe,
  children,
  onFetchMore,
  itemHeight = 36,
}: FrameProps<K, E>): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  const onFetchMoreRef = useSyncedRef(onFetchMore);
  const { visible } = Dialog.useContext();
  const refCallback = useCallback(
    (el: HTMLDivElement) => {
      ref.current = el;
      if (ref.current == null) return;
      onFetchMoreRef.current?.();
    },
    [onFetchMoreRef, visible],
  );
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => ref.current,
    estimateSize: () => itemHeight,
    overscan: 10,
    onChange: useCallback(
      (v: Virtualizer<HTMLDivElement, HTMLDivElement>) => {
        const items = v.getVirtualItems();
        if (items.length > 0 && items[items.length - 1].index === data.length - 1)
          onFetchMore?.();
      },
      [data.length, onFetchMore],
    ),
  });
  const items = virtualizer.getVirtualItems().map((item) => ({
    key: data[item.index],
    index: item.index,
    translate: item.start,
  }));
  const dataCtxValue = useMemo<DataContextValue<K>>(
    () => ({
      ref: refCallback,
      getItem,
      data,
      subscribe,
      getTotalSize: () => virtualizer.getTotalSize(),
      getItems: () => items,
    }),
    [refCallback, virtualizer, data, getItem, items],
  );
  const utilCtxValue = useMemo<UtilContextValue<K, E>>(
    () => ({
      ref: refCallback,
      getItem,
      scrollToIndex: virtualizer.scrollToIndex,
      subscribe,
    }),
    [refCallback, virtualizer, getItem, subscribe],
  );
  return (
    <DataContext.Provider value={dataCtxValue}>
      <UtilContext.Provider value={utilCtxValue as unknown as UtilContextValue}>
        {children}
      </UtilContext.Provider>
    </DataContext.Provider>
  );
};
