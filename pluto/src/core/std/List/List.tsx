// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  PropsWithChildren,
  ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { useTransforms } from "@/core/hooks";
import { ListContextProvider } from "@/core/std/List/ListContext";
import { ListColumn } from "@/core/std/List/types";

export interface ListProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends PropsWithChildren<unknown> {
  data?: E[];
  emptyContent?: ReactElement;
}

export const List = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  children,
  data: propsData,
  emptyContent,
}: ListProps<K, E>): ReactElement => {
  const [columns, setColumns] = useState<Array<ListColumn<K, E>>>([]);
  const [selected, setSelected] = useState<readonly K[]>([]);
  const [hover, setHover] = useState<number>(-1);
  const [onSelect, setOnSelect] = useState<((key: K) => void) | undefined>(undefined);
  const [clear, setClear] = useState<(() => void) | undefined>(undefined);
  const { transform, setTransform, deleteTransform } = useTransforms<E>({});
  const [data, setData] = useState<E[]>(() => propsData ?? []);
  useEffect(() => {
    if (propsData != null) setData(propsData);
  }, [propsData]);
  const transformedData = useMemo(() => transform(data), [data, transform]);
  const setSourceData = useCallback((data: E[]) => setData(data), [setData]);
  const [emptyContent_, setEmptyContent] = useState<ReactElement | undefined>(
    emptyContent
  );
  return (
    <ListContextProvider<K, E>
      value={{
        setEmptyContent,
        sourceData: data,
        data: transformedData,
        setSourceData,
        deleteTransform,
        setTransform,
        hover: {
          value: hover,
          onChange: setHover,
        },
        emptyContent: emptyContent ?? emptyContent_,
        columnar: {
          columns,
          setColumns,
        },
        select: {
          value: selected,
          onChange: setSelected,
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
