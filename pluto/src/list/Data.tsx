// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useMemo,
} from "react";

import { useRequiredContext } from "@/hooks/useRequiredContext";

export interface Data<K extends record.Key, E extends record.Keyed<K>> {
  items: K[];
  getItem: (key: K) => E | undefined;
  useItem: (key: K) => E | undefined;
}

export interface DataContextValue<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> extends Data<K, E> {}

const Context = createContext<DataContextValue | null>({
  items: [],
  getItem: () => undefined,
  useItem: () => undefined,
});

export const useData = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>() => useRequiredContext(Context) as unknown as Data<K, E>;

export interface DataProviderProps<K extends record.Key, E extends record.Keyed<K>>
  extends PropsWithChildren<{}> {
  data: Data<K, E>;
}

export const DataProvider = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
>({
  data,
  children,
}: DataProviderProps<K, E>): ReactElement => {
  const ctxValue = useMemo(() => data as unknown as DataContextValue, [data]);
  return <Context value={ctxValue}>{children}</Context>;
};
