import React, { useState } from "react";
import { useMemo } from "react";
import { ListContextProvider } from "./ListContext";
import { ListEntry, TypedListColumn, TypedListTransform } from "./types";
import { useMultiSelect, useMultiSelectProps } from "./useMultiSelect";

export interface ListProps<E extends ListEntry>
  extends React.PropsWithChildren<any>,
    useMultiSelectProps<E> {
  data: E[];
}

export const List = <E extends ListEntry>({
  children,
  data,
  selectMultiple = true,
  selected: selectedProp,
  onSelect: onSelectProp,
}: ListProps<E>) => {
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
};

export default List;
