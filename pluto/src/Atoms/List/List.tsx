import React, { useState } from "react";
import { useMemo } from "react";
import Column from "./ListColumn";
import ListCore from "./Core";
import { ListContext, ListContextProvider } from "./ListContext";
import ListSearch from "./ListSearch";
import { Key, TypedColumn, TypedListEntry, TypedTransform } from "./Types";
import { useMultiSelect } from "./useMultiSelect";

interface ListProps<K extends Key, E extends TypedListEntry<K>>
  extends React.PropsWithChildren<any> {
  data: E[];
}

export default function List<K extends Key, E extends TypedListEntry<K>>({
  children,
  data,
}: ListProps<K, E>) {
  const [transforms, setTransforms] = useState<
    Record<string, TypedTransform<K, E> | undefined>
  >({});
  const [columns, setColumns] = useState<TypedColumn<K, E>[]>([]);

  const setTransform = (key: string, transform: TypedTransform<K, E>) => {
    setTransforms((transforms) => ({ ...transforms, [key]: transform }));
  };

  const removeTransform = (key: string) => {
    setTransforms((transforms) => ({ ...transforms, [key]: undefined }));
  };

  const transformedData = useMemo(() => {
    return Object.values(transforms).reduce(
      (data, transform) => (transform ? transform(data) : data),
      data
    );
  }, [data, transforms]);

  const { selected, onSelect, clearSelected } = useMultiSelect<K, E>(
    transformedData
  );

  return (
    <ListContextProvider
      value={{
        clearSelected,
        sourceData: data,
        selected,
        onSelect,
        data: transformedData,
        columnar: {
          columns,
          setColumns,
        },
        setTransform,
        removeTransform,
      }}
    >
      {children}
    </ListContextProvider>
  );
}

List.Context = ListContext;
List.Search = ListSearch;
List.Core = ListCore;
List.Column = Column;
