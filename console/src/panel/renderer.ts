import { context, type Menu, Panel } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

export interface UseNameReturn {
  name: string;
  rename?: (name: string) => void;
}

export interface TabRenderer {
  Content: FC;
  Toolbar: FC;
  useName: () => UseNameReturn;
  icon: ReactElement;
  ContextMenu?: FC<Menu.ContextMenuProps>;
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
