import { createContext, useContext } from "react";
import { LayoutRenderer } from "../types";

export type LayoutRenderers = Record<string, LayoutRenderer>;

const LayoutRendererContext = createContext<LayoutRenderers>({});

export const useLayoutRenderer = (key: string) =>
  useContext(LayoutRendererContext)[key];

export const LayoutRendererProvider = LayoutRendererContext.Provider;
