import CoreTabs from "./Tabs";

export type { TabEntry, TabsProps } from "./Tabs";

type CoreTabsType = typeof CoreTabs;

interface TabsType extends CoreTabsType {}

export const Tabs = CoreTabs as TabsType;
