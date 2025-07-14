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
  useListItem: (key?: K) => E | undefined;
  scrollToIndex: (index: number) => void;
}

const DataContext = createContext<DataContextValue | null>(null);
const UtilContext = createContext<UtilContextValue | null>(null);

export interface FrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
> extends PropsWithChildren {
  data: K[];
  itemHeight?: number;
  useListItem: (key?: K) => E | undefined;
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
  const { useListItem } = useUtilContext<K, E>();
  return useListItem(key);
};

export const useData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>(): DataContextValue<K> & UtilContextValue<K, E> => {
  const { data, getItems, getTotalSize } = useDataContext<K>();
  const { ref, useListItem, scrollToIndex } = useUtilContext<K, E>();
  return { data, getItems, getTotalSize, ref, useListItem, scrollToIndex };
};

export const Frame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>({
  data,
  useListItem,
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
      useListItem,
      data,
      getTotalSize: () => virtualizer.getTotalSize(),
      getItems: () => items,
    }),
    [refCallback, virtualizer, data, useListItem, items],
  );
  const utilCtxValue = useMemo<UtilContextValue<K, E>>(
    () => ({ ref: refCallback, useListItem, scrollToIndex: virtualizer.scrollToIndex }),
    [refCallback, virtualizer, useListItem],
  );
  return (
    <DataContext.Provider value={dataCtxValue}>
      <UtilContext.Provider value={utilCtxValue as unknown as UtilContextValue}>
        {children}
      </UtilContext.Provider>
    </DataContext.Provider>
  );
};
