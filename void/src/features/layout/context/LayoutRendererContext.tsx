import { createContext, useContext } from "react";

export type LayoutRenderers = Record<string, React.FC>;

const LayoutRendererContext = createContext<LayoutRenderers>({});

export const useLayoutRenderer = (key: string) =>
  useContext(LayoutRendererContext)[key];

export const LayoutRendererProvider = LayoutRendererContext.Provider;
