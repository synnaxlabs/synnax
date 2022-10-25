import { Tabs as CoreTabs } from "./Tabs";

export type { Tab, TabsProps } from "./Tabs";

type CoreTabsType = typeof CoreTabs;

interface TabsType extends CoreTabsType {}

export const Tabs = CoreTabs as TabsType;
