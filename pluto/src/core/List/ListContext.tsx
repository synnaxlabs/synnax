// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, useContext, createContext } from "react";

import { ListColumn } from "./types";

import { UseTransformsReturn } from "@/hooks/useTransforms";
import { RenderableRecord } from "@/util/record";

export interface ListContextProps<E extends RenderableRecord<E> = RenderableRecord>
  extends Omit<UseTransformsReturn<E>, "transform"> {
  columnar: {
    columns: Array<ListColumn<E>>;
    setColumns: (cbk: (columns: Array<ListColumn<E>>) => Array<ListColumn<E>>) => void;
  };
  data: E[];
  sourceData: E[];
  select: {
    onSelect?: (key: string) => void;
    clear?: () => void;
    setOnSelect: (cbk: (key: string) => void) => void;
    setClear: (cbk: () => void) => void;
  };
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
});

export const useListContext = <
  E extends RenderableRecord<E>
>(): ListContextProps<E> => {
  return useContext(ListContext) as unknown as ListContextProps<E>;
};

export interface ListContextProviderProps<E extends RenderableRecord<E>>
  extends PropsWithChildren<unknown> {
  value: ListContextProps<E>;
}

export const ListContextProvider = <E extends RenderableRecord<E>>({
  value,
  children,
}: ListContextProviderProps<E>): JSX.Element => {
  return (
    <ListContext.Provider value={value as unknown as ListContextProps}>
      {children}
    </ListContext.Provider>
  );
};
