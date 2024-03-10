// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type PropsWithChildren,
  createContext,
  type ReactElement,
  useState,
  useMemo,
  useEffect,
} from "react";

import { type Keyed, type Key } from "@synnaxlabs/x";

import { useCombinedStateAndRef, useSyncedRef } from "@/hooks";
import { useRequiredContext } from "@/hooks/useRequiredContext";
import { useTransforms, type UseTransformsReturn } from "@/hooks/useTransforms";
import { type state } from "@/state";

export interface DataContextValue<K extends Key = Key, E extends Keyed<K> = Keyed<K>> {
  transformedData: E[];
  sourceData: E[];
  emptyContent?: React.ReactElement;
}

export interface DataUtilContextValue<
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
> extends Omit<UseTransformsReturn<E>, "transform"> {
  setSourceData: state.Set<E[]>;
  getSourceData: () => E[];
  getTransformedData: () => E[];
  setEmptyContent: state.Set<React.ReactElement | undefined>;
}

const DataContext = createContext<DataContextValue | null>({
  transformedData: [],
  sourceData: [],
});

const DataUtilContext = createContext<DataUtilContextValue | null>({
  setSourceData: () => undefined,
  getSourceData: () => [],
  getTransformedData: () => [],
  deleteTransform: () => undefined,
  setTransform: () => undefined,
  setEmptyContent: () => undefined,
});

export const useDataContext = <
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
>(): DataContextValue<K, E> =>
  useRequiredContext(DataContext) as DataContextValue<K, E>;

export const useDataUtilContext = <
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
>(): DataUtilContextValue<K, E> =>
  useRequiredContext(DataUtilContext) as unknown as DataUtilContextValue<K, E>;

export const useTransformedData = <
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
>(): E[] => useDataContext<K, E>().transformedData;

export const useSourceData = <
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
>(): E[] => useDataContext<K, E>().sourceData;

export const useGetTransformedData = <
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
>(): (() => E[]) => useDataUtilContext<K, E>().getTransformedData;

export const useSetSourceData = <
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
>(): state.Set<E[]> => useDataUtilContext<K, E>().setSourceData;

export interface DataProviderProps<K extends Key, E extends Keyed<K>>
  extends PropsWithChildren<{}> {
  data?: E[];
  emptyContent?: React.ReactElement;
}

export const DataProvider = <K extends Key = Key, E extends Keyed<K> = Keyed<K>>({
  data: sourceData,
  emptyContent: emptyContentProp,
  children,
}: DataProviderProps<K, E>): ReactElement => {
  const { transform, setTransform, deleteTransform } = useTransforms<E>({});
  const [data, setData, dataRef] = useCombinedStateAndRef<E[]>(() => sourceData ?? []);

  useEffect(() => {
    if (sourceData != null) setData(sourceData);
  }, [sourceData]);

  const transformedData = useMemo(() => transform(data), [data, transform]);
  const transformedDataRef = useSyncedRef(transformedData);

  const [emptyContent, setEmptyContent] = useState<React.ReactElement | undefined>(
    undefined,
  );
  useEffect(() => {
    if (emptyContentProp != null) setEmptyContent(emptyContentProp);
  }, [emptyContentProp]);

  const utilValue: DataUtilContextValue<K, E> = useMemo(
    () => ({
      setSourceData: setData,
      getSourceData: () => dataRef.current,
      getTransformedData: () => transformedDataRef.current,
      deleteTransform,
      setTransform,
      setEmptyContent,
    }),
    [setData, dataRef, transformedDataRef, deleteTransform, setTransform],
  );

  const ctxValue: DataContextValue<K, E> = useMemo(
    () => ({
      transformedData,
      sourceData: data,
      emptyContent,
    }),
    [transformedData, data, emptyContent],
  );

  return (
    <DataUtilContext.Provider value={utilValue as unknown as DataUtilContextValue}>
      <DataContext.Provider value={ctxValue}>{children}</DataContext.Provider>
    </DataUtilContext.Provider>
  );
};
