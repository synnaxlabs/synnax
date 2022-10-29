import { resetTabSelection, Tabs as CoreTabs, useStaticTabs } from "./Tabs";

export type { Tab, TabsProps } from "./Tabs";

type CoreTabsType = typeof CoreTabs;

interface TabsType extends CoreTabsType {
  useStatic: typeof useStaticTabs;
  resetSelection: typeof resetTabSelection;
}

export const Tabs = CoreTabs as TabsType;

Tabs.useStatic = useStaticTabs;
Tabs.resetSelection = resetTabSelection;
