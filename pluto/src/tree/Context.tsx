import { type record } from "@synnaxlabs/x";
import { createContext, type PropsWithChildren, useMemo } from "react";

import { useRequiredContext } from "@/hooks";
import { type Shape } from "@/tree/core";

export interface ContextValue<K extends record.Key = record.Key> {
  shape: Shape<K>;
}

export const Context = createContext<ContextValue | null>(null);

export const useContext = <K extends record.Key>() =>
  useRequiredContext(Context) as ContextValue<K>;

export interface ProviderProps<K extends record.Key>
  extends PropsWithChildren<ContextValue<K>> {}

export const useNodeShape = (index: number) => {
  const { shape } = useContext();
  return shape.nodes[index];
};

export const Provider = <K extends record.Key>({
  shape,
  children,
}: ProviderProps<K>) => {
  const value = useMemo(() => ({ shape }), [shape]);
  return <Context.Provider value={value}>{children}</Context.Provider>;
};
