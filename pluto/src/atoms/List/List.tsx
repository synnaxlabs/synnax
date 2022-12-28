import { PropsWithChildren, useMemo, useState } from "react";

import { ListContextProvider } from "./ListContext";
import { ListColumn } from "./types";

import { useTransforms } from "@/hooks/useTransforms";
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
