import { type record } from "@synnaxlabs/x";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  type RefObject,
  useMemo,
  useRef,
} from "react";

import { useRequiredContext } from "@/hooks";

export interface ItemSpec<K extends record.Key = record.Key> {
  key: K;
  index: number;
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
  ref: RefObject<HTMLDivElement | null>;
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
  useListItem: (key?: K) => E | undefined;
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
  const { useListItem: useItem } = useUtilContext<K, E>();
  return useItem(key);
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
}: FrameProps<K, E>): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => ref.current,
    estimateSize: () => 36,
  });
  const dataCtxValue = useMemo<DataContextValue<K>>(
    () => ({
      ref,
      useListItem,
      data,
      getTotalSize: () => virtualizer.getTotalSize(),
      getItems: () =>
        virtualizer
          .getVirtualItems()
          .map((item) => ({ key: data[item.index], index: item.index })),
    }),
    [ref, virtualizer, data, useListItem],
  );
  const utilCtxValue = useMemo<UtilContextValue<K, E>>(
    () => ({ ref, useListItem, scrollToIndex: virtualizer.scrollToIndex }),
    [ref, virtualizer, useListItem],
  );
  return (
    <DataContext.Provider value={dataCtxValue}>
      <UtilContext.Provider value={utilCtxValue as unknown as UtilContextValue}>
        {children}
      </UtilContext.Provider>
    </DataContext.Provider>
  );
};
