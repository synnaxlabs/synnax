import { PropsWithChildren, useContext, createContext } from "react";

import {
  RenderableRecord,
  TypedListColumn,
  TypedListTransform,
  UntypedListColumn,
  UntypedListTransform,
} from "./types";

export interface ListContextProps {
  data: RenderableRecord[];
  sourceData: RenderableRecord[];
  selected: string[];
  onSelect: (key: string) => void;
  clearSelected: () => void;
  columnar: {
    columns: UntypedListColumn[];
    setColumns: (cbk: (columns: UntypedListColumn) => UntypedListColumn[]) => void;
  };
  setTransform: (key: string, transform: UntypedListTransform) => void;
  removeTransform: (key: string) => void;
}

export interface TypedListContextProps<E extends RenderableRecord<E>> {
  columnar: {
    columns: Array<TypedListColumn<E>>;
    setColumns: (
      cbk: (columns: Array<TypedListColumn<E>>) => Array<TypedListColumn<E>>
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

export const useListContext = <
  E extends RenderableRecord<E>
>(): TypedListContextProps<E> => {
  return useContext(ListContext) as unknown as TypedListContextProps<E>;
};

export interface ListContextProviderProps<E extends RenderableRecord<E>>
  extends PropsWithChildren<unknown> {
  value: TypedListContextProps<E>;
}

export const ListContextProvider = <E extends RenderableRecord<E>>({
  value,
  children,
}: ListContextProviderProps<E>): JSX.Element => {
  return (
    <ListContext.Provider value={value as unknown as ListContextProps}>
      {children}
    </ListContext.Provider>
  );
};
