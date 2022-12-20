import { Tabs as CoreTabs, resetTabSelection, useStaticTabs, renameTab } from "./Tabs";

export type { Tab, TabsProps } from "./Tabs";

type CoreTabsType = typeof CoreTabs;

interface TabsType extends CoreTabsType {
  useStatic: typeof useStaticTabs;
  resetSelection: typeof resetTabSelection;
  rename: typeof renameTab;
}

export const Tabs = CoreTabs as TabsType;

Tabs.useStatic = useStaticTabs;
Tabs.resetSelection = resetTabSelection;
Tabs.rename = renameTab;
