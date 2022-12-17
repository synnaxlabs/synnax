import { createContext, useContext } from "react";

import { LayoutRenderer } from "../types";

export type LayoutRenderers = Record<string, LayoutRenderer>;

const LayoutRendererContext = createContext<LayoutRenderers>({});

export const useLayoutRenderer = (key: string): LayoutRenderer => {
  const r = useContext(LayoutRendererContext)[key];
  if (r == null) throw new Error(`no renderer for layout type ${key}`);
  return r;
};

export const LayoutRendererProvider = LayoutRendererContext.Provider;
