import {
  createContext,
  type Key,
  type PropsWithChildren,
  type ReactElement,
} from "react";

import { useRequiredContext } from "@/hooks";

const Context = createContext<Key | null>(null);

export const use = <K extends Key = string>(): K => useRequiredContext(Context) as K;

export interface ProviderProps<K extends Key = string> extends PropsWithChildren {
  itemKey: K;
}

export const Provider = <K extends Key = string>({
  children,
  itemKey,
}: ProviderProps<K>): ReactElement => (
  <Context.Provider value={itemKey}>{children}</Context.Provider>
);
