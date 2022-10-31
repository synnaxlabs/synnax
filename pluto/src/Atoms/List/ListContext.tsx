import { PropsWithChildren, useContext } from "react";
import { createContext } from "react";
import {
  ListEntry,
  UntypedListColumn,
  UntypedListTransform,
  TypedListColumn,
  TypedListTransform,
} from "./types";

export interface ListContextProps {
  data: ListEntry[];
  sourceData: ListEntry[];
  selected: string[];
  onSelect: (key: string) => void;
  clearSelected: () => void;
  columnar: {
    columns: UntypedListColumn[];
    setColumns: (
      cbk: (columns: UntypedListColumn) => UntypedListColumn[]
    ) => void;
  };
  setTransform: (key: string, transform: UntypedListTransform) => void;
  removeTransform: (key: string) => void;
}

export interface TypedListContextProps<E extends ListEntry> {
  columnar: {
    columns: TypedListColumn<E>[];
    setColumns: (
      cbk: (columns: TypedListColumn<E>[]) => TypedListColumn<E>[]
    ) => void;
  };
  data: E[];
  sourceData: E[];
  selected: string[];
  onSelect: (key: string) => void;
  clearSelected: () => void;
  setTransform: (key: string, transform: TypedListTransform<E>) => void;
  removeTransform: (key: string) => void;
}

export const ListContext = createContext<ListContextProps>({
  columnar: {
    columns: [],
    setColumns: () => undefined,
  },
  sourceData: [],
  clearSelected: () => undefined,
  selected: [],
  data: [],
  setTransform: () => undefined,
  removeTransform: () => undefined,
  onSelect: () => undefined,
});

export const useListContext = <E extends ListEntry>() => {
  return useContext(ListContext) as unknown as TypedListContextProps<E>;
};

export interface ListContextProviderProps<E extends ListEntry>
  extends PropsWithChildren<unknown> {
  value: TypedListContextProps<E>;
}

export const ListContextProvider = <E extends ListEntry>({
  value,
  children,
}: ListContextProviderProps<E>) => {
  return (
    <ListContext.Provider value={value as unknown as ListContextProps}>
      {children}
    </ListContext.Provider>
  );
};
