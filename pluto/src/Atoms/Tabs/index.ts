import { Tabs as CoreTabs, useStaticTabs } from "./Tabs";

export type { Tab, TabsProps } from "./Tabs";

type CoreTabsType = typeof CoreTabs;

interface TabsType extends CoreTabsType {
  useStatic: typeof useStaticTabs;
}

export const Tabs = CoreTabs as TabsType;
