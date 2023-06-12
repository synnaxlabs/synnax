// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, useMemo, useState } from "react";

import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { useTransforms } from "@/core/hooks";
import { ListContextProvider } from "@/core/std/List/ListContext";
import { ListColumn } from "@/core/std/List/types";

export interface ListProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends PropsWithChildren<unknown> {
  data: E[];
  emptyContent?: ReactElement;
}

export const List = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  children,
  data,
  emptyContent,
}: ListProps<K, E>): ReactElement => {
  const [columns, setColumns] = useState<Array<ListColumn<K, E>>>([]);
  const [onSelect, setOnSelect] = useState<((key: K) => void) | undefined>(undefined);
  const [clear, setClear] = useState<(() => void) | undefined>(undefined);
  const { transform, setTransform, deleteTransform } = useTransforms<E>({});
  const transformedData = useMemo(() => transform(data), [data, transform]);
  return (
    <ListContextProvider<K, E>
      value={{
        sourceData: data,
        data: transformedData,
        deleteTransform,
        setTransform,
        emptyContent,
        columnar: {
          columns,
          setColumns,
        },
        select: {
          setOnSelect,
          onSelect,
          clear,
          setClear,
        },
      }}
    >
      {children}
    </ListContextProvider>
  );
};
