import React, { useState } from "react";
import { useMemo } from "react";
import Column from "./ListColumn";
import ListCore from "./ListCore";
import { ListContext, ListContextProvider } from "./ListContext";
import ListSearch from "./ListSearch";
import { ListEntry, TypedListColumn, TypedListTransform } from "./Types";
import { useMultiSelect, useMultiSelectProps } from "./useMultiSelect";

export interface ListProps<E extends ListEntry>
  extends React.PropsWithChildren<any>,
    useMultiSelectProps<E> {
  data: E[];
}

function List<E extends ListEntry>({
  children,
  data,
  selectMultiple = true,
  selected: selectedProp,
  onSelect: onSelectProp,
}: ListProps<E>) {
  const [transforms, setTransforms] = useState<
    Record<string, TypedListTransform<E> | undefined>
  >({});
  const [columns, setColumns] = useState<TypedListColumn<E>[]>([]);

  const setTransform = (key: string, transform: TypedListTransform<E>) => {
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

  const { selected, onSelect, clearSelected } = useMultiSelect<E>({
    data: transformedData,
    selectMultiple,
    selected: selectedProp,
    onSelect: onSelectProp,
  });

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
