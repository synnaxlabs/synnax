import React, { useState } from "react";
import { useMemo } from "react";
import Column from "./ListColumn";
import ListCore from "./ListCore";
import { ListContext, ListContextProvider } from "./ListContext";
import ListSearch from "./ListSearch";
import {
  Key,
  TypedListColumn,
  TypedListEntry,
  TypedListTransform,
} from "./Types";
import { useMultiSelect } from "./useMultiSelect";

export interface ListProps<K extends Key, E extends TypedListEntry<K>>
  extends React.PropsWithChildren<any> {
  data: E[];
}

function List<K extends Key, E extends TypedListEntry<K>>({
  children,
  data,
}: ListProps<K, E>) {
  const [transforms, setTransforms] = useState<
    Record<string, TypedListTransform<K, E> | undefined>
  >({});
  const [columns, setColumns] = useState<TypedListColumn<K, E>[]>([]);

  const setTransform = (key: string, transform: TypedListTransform<K, E>) => {
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

/**
 * Context is a React Context that contains state used by various List components.
 */
List.Context = ListContext;

/**
 * Search is a component that renders a search bar for filtering the list.
 */
List.Search = ListSearch;

/**
 * Core is a set of components that can be used to render list items.
 */
List.Core = ListCore;

/**
 * Column is a component that renders a list in columnar format.
 */
List.Column = Column;

export default List;
