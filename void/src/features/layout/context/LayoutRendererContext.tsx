import { createContext, useContext } from "react";

export type LayoutRenderers = Record<string, React.FC>;

const LayoutRenderersContext = createContext<LayoutRenderers>({});

export const useLayoutRenderer = (key: string) =>
  useContext(LayoutRenderersContext)[key];

export const LayoutRenderersProvider = LayoutRenderersContext.Provider;
