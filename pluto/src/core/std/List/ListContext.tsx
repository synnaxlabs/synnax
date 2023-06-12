// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, useContext, createContext, ReactElement } from "react";

import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { UseTransformsReturn } from "@/core/hooks/useTransforms";
import { ListColumn } from "@/core/std/List/types";

export interface ListContextProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<UseTransformsReturn<E>, "transform"> {
  columnar: {
    columns: Array<ListColumn<K, E>>;
    setColumns: (
      cbk: (columns: Array<ListColumn<K, E>>) => Array<ListColumn<K, E>>
    ) => void;
  };
  data: E[];
  sourceData: E[];
  select: {
    onSelect?: (key: K) => void;
    clear?: () => void;
    setOnSelect: (cbk: (key: K) => void) => void;
    setClear: (cbk: () => void) => void;
  };
  emptyContent?: ReactElement;
}

export const ListContext = createContext<ListContextProps>({
  columnar: {
    columns: [],
    setColumns: () => undefined,
  },
  sourceData: [],
  data: [],
  setTransform: () => undefined,
  deleteTransform: () => undefined,
  select: {
    onSelect: undefined,
    setOnSelect: () => undefined,
    clear: undefined,
    setClear: () => undefined,
  },
  emptyContent: undefined,
});

export const useListContext = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>(): ListContextProps<K, E> => {
  return useContext(ListContext) as unknown as ListContextProps<K, E>;
};

export interface ListContextProviderProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends PropsWithChildren<unknown> {
  value: ListContextProps<K, E>;
}

export const ListContextProvider = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  value,
  children,
}: ListContextProviderProps<K, E>): ReactElement => {
  return (
    <ListContext.Provider value={value as unknown as ListContextProps}>
      {children}
    </ListContext.Provider>
  );
};
