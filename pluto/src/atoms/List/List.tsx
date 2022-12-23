import { PropsWithChildren, useState, useMemo } from "react";

import { ListContextProvider } from "./ListContext";
import { RenderableRecord, TypedListColumn, TypedListTransform } from "./types";
import { useMultiSelect, useMultiSelectProps } from "./useMultiSelect";

export interface ListProps<E extends RenderableRecord<E>>
  extends PropsWithChildren<unknown>,
    useMultiSelectProps<E> {
  data: E[];
}

export const List = <E extends RenderableRecord<E>>({
  children,
  data,
  selectMultiple = true,
  selected: selectedProp,
  onSelect: onSelectProp,
}: ListProps<E>): JSX.Element => {
  const [transforms, setTransforms] = useState<
    Record<string, TypedListTransform<E> | undefined>
  >({});
  const [columns, setColumns] = useState<Array<TypedListColumn<E>>>([]);

  const setTransform = (key: string, transform: TypedListTransform<E>): void =>
    setTransforms((transforms) => ({ ...transforms, [key]: transform }));

  const removeTransform = (key: string): void =>
    setTransforms((transforms) => ({ ...transforms, [key]: undefined }));

  const transformedData = useMemo(() => {
    return Object.values(transforms).reduce(
      (data, transform) => (transform != null ? transform(data) : data),
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
