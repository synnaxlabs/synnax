import { createContext, useContext } from "react";

import { LayoutRenderer } from "../types";

export type LayoutRenderers = Record<string, LayoutRenderer>;

const LayoutRendererContext = createContext<LayoutRenderers>({});

export const LayoutRendererProvider = LayoutRendererContext.Provider;

export const useLayoutRenderer = (type: string): LayoutRenderer => {
  const r = useContext(LayoutRendererContext)[type];
  if (r == null) throw new Error(`no renderer for layout type ${type}`);
  return r;
};
