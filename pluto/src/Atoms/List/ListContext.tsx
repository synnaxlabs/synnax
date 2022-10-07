import { PropsWithChildren, useContext } from "react";
import { createContext } from "react";

export type Key = string | number;

export type CoreColumn = {
  label: string;
  visible: boolean;
  width: number;
};

export type UntypedColumn = {
  key: string;
} & CoreColumn;

export type TypedColumn<K extends Key, E extends TypedListEntry<K>> = {
  key: keyof E;
} & CoreColumn;

export type CoreListEntry = {
  [key: string]: any;
};

export type UntypedListEntry = {
  key: string;
} & CoreListEntry;

export type TypedListEntry<K extends Key> = {
  key: K;
} & CoreListEntry;

export type UntypedTransform = (data: UntypedListEntry[]) => UntypedListEntry[];

export type TypedTransform<K extends Key, E extends TypedListEntry<K>> = (
  data: E[]
) => E[];

export type ListItemProps<K extends Key, E extends TypedListEntry<K>> = {
  entry: E;
  index: number;
  style: React.CSSProperties;
  selected: boolean;
  columns: TypedColumn<K, E>[];
  onSelect: (key: K) => void;
};

export interface ListContextProps {
  data: UntypedListEntry[];
  sourceData: UntypedListEntry[];
  selected: Key[];
  onSelect: (key: Key) => void;
  clearSelected: () => void;
  columnar: {
    columns: UntypedColumn[];
    setColumns: (cbk: (columns: UntypedColumn) => UntypedColumn[]) => void;
  };
  setTransform: (key: string, transform: UntypedTransform) => void;
  removeTransform: (key: string) => void;
}

export interface TypedListContextProps<
  K extends Key,
  E extends TypedListEntry<K>
> {
  columnar: {
    columns: TypedColumn<K, E>[];
    setColumns: (
      cbk: (columns: TypedColumn<K, E>[]) => TypedColumn<K, E>[]
    ) => void;
  };
  data: E[];
  sourceData: E[];
  selected: K[];
  onSelect: (key: K) => void;
  clearSelected: () => void;
  setTransform: (key: string, transform: TypedTransform<K, E>) => void;
  removeTransform: (key: string) => void;
}

export const ListContext = createContext<ListContextProps>({
  columnar: {
    columns: [],
    setColumns: () => {},
  },
  sourceData: [],
  clearSelected: () => {},
  selected: [],
  data: [],
  setTransform: () => {},
  removeTransform: () => {},
  onSelect: () => {},
});

export const useListContext = <
  K extends Key,
  E extends TypedListEntry<K>
>() => {
  return useContext(ListContext) as unknown as TypedListContextProps<K, E>;
};

export interface ListContextProviderProps<
  K extends Key,
  E extends TypedListEntry<K>
> extends PropsWithChildren<any> {
  value: TypedListContextProps<K, E>;
}

export const ListContextProvider = <
  K extends Key,
  E extends TypedListEntry<K>
>({
  value,
  children,
}: ListContextProviderProps<K, E>) => {
  return (
    <ListContext.Provider value={value as unknown as ListContextProps}>
      {children}
    </ListContext.Provider>
  );
};
