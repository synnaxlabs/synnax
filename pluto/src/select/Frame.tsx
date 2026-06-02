// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { List } from "@/list";
import { Provider } from "@/select/Provider";
import {
  useMultiple,
  type UseMultipleProps,
  useSingle,
  type UseSingleProps,
} from "@/select/use";

interface MultipleProviderProps<
  K extends record.Key = record.Key,
> extends PropsWithChildren<UseMultipleProps<K>> {}

const MultipleProvider = <K extends record.Key = record.Key>({
  children,
  ...rest
}: MultipleProviderProps<K>): ReactElement => {
  const { value } = rest;
  const res = useMultiple(rest);
  return (
    <Provider value={value} {...res}>
      {children}
    </Provider>
  );
};

interface SingleProviderProps<
  K extends record.Key = record.Key,
> extends PropsWithChildren<UseSingleProps<K>> {}

const SingleProvider = <K extends record.Key = record.Key>({
  children,
  ...rest
}: SingleProviderProps<K>): ReactElement => {
  const { value } = rest;
  const res = useSingle(rest);
  return (
    <Provider<K> value={value} {...res}>
      {children}
    </Provider>
  );
};

export interface TriggerProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> {
  value: K | null;
  useItem: (key: K) => E;
  onClick: () => void;
}

interface BaseFrameProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
> extends Omit<List.FrameProps<K, E>, "onChange"> {}

export interface MultipleFrameProps<
  K extends record.Key,
  E extends record.Keyed<K> | undefined,
>
  extends BaseFrameProps<K, E>, UseMultipleProps<K> {
  multiple: true;
}

export interface SingleFrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K> | undefined,
>
  extends BaseFrameProps<K, E>, UseSingleProps<K> {
  multiple?: false;
}

export type FrameProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
> = MultipleFrameProps<K, E> | SingleFrameProps<K, E>;

export const Frame = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  data,
  getItem,
  subscribe,
  itemHeight,
  multiple,
  onFetchMore,
  virtual = false,
  value,
  onChange,
  ...rest
}: FrameProps<K, E>): ReactElement => (
  <List.Frame<K, E>
    data={data}
    getItem={getItem}
    subscribe={subscribe}
    onFetchMore={onFetchMore}
    itemHeight={itemHeight}
    virtual={virtual}
  >
    {multiple ? (
      <MultipleProvider value={value} onChange={onChange} {...rest} />
    ) : (
      <SingleProvider value={value} onChange={onChange} {...rest} />
    )}
  </List.Frame>
);
Frame.displayName = "Select.Frame";
