// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, useMemo, useState } from "react";

import { ListContextProvider } from "./ListContext";
import { ListColumn } from "./types";

import { useTransforms } from "@/hooks";
import { RenderableRecord } from "@/util/record";

export interface ListProps<E extends RenderableRecord<E>>
  extends PropsWithChildren<unknown> {
  data: E[];
}

export const List = <E extends RenderableRecord<E>>({
  children,
  data,
}: ListProps<E>): JSX.Element => {
  const [columns, setColumns] = useState<Array<ListColumn<E>>>([]);
  const [onSelect, setOnSelect] = useState<((key: string) => void) | undefined>(
    undefined
  );
  const [clear, setClear] = useState<(() => void) | undefined>(undefined);
  const { transform, setTransform, deleteTransform } = useTransforms<E>({});
  const transformedData = useMemo(() => transform(data), [data, transform]);
  return (
    <ListContextProvider<E>
      value={{
        sourceData: data,
        data: transformedData,
        deleteTransform,
        setTransform,
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
