import { context, Panel } from "@synnaxlabs/pluto";
import { type FC } from "react";

export interface UseNameReturn {
  name: string;
  rename?: (name: string) => void;
}

export interface TabContentRenderer extends FC {}

export interface ToolbarRenderer extends FC {}

export interface UseName {
  (): UseNameReturn;
}

interface TabRendererContextValue extends Record<string, TabRenderer> {}

const [TabRendererContext, useTabRendererContext] =
  context.create<TabRendererContextValue>({
    displayName: "TabRendererContext.Provider",
    providerName: "TabRendererContext.Provider",
  });

export const useTabRenderer = (): TabRenderer => {
  const type = Panel.useSelectTabType({});
  return useTabRendererContext("useTabRenderer")[type];
};
