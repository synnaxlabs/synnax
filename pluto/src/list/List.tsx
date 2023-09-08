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
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import { type Key, type KeyedRenderableRecord } from "@synnaxlabs/x";

import { useTransforms } from "@/hooks";
import { Provider } from "@/list/Context";
import { type ColumnSpec } from "@/list/types";

export interface ListProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends PropsWithChildren<unknown> {
  data?: E[];
  emptyContent?: ReactElement;
}

/**
 * The main component for building a List. By itself, it does not render any HTML, and
 * should be used in conjunction with its subcomponents (List.'X') to build a list
 * component to fit your needs.
 *
 * @param props - The props for the List component.
 * @param props.data - The data to be displayed in the list. The values of the object in
 * each entry of the array must satisfy the {@link RenderableValue} interface i.e. they
 * must be a primitive type or implement a 'toString' method.
 * @param props.children - Sub-components of the List component to add additional functionality.
 *
 */
export const List = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
>({
  children,
  data: propsData,
  emptyContent,
}: ListProps<K, E>): ReactElement => {
  const [columns, setColumns] = useState<Array<ColumnSpec<K, E>>>([]);
  const [selected, setSelected] = useState<K[]>([]);
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
    emptyContent,
  );
  return (
    <Provider<K, E>
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
    </Provider>
  );
};
