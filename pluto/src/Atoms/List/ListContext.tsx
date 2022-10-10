import { PropsWithChildren, useContext } from "react";
import { createContext } from "react";
import {
  UntypedListEntry,
  Key,
  UntypedColumn,
  UntypedTransform,
  TypedListEntry,
  TypedColumn,
  TypedTransform,
} from "./Types";

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
