import {
  Tabs as CoreTabs,
  resetTabSelection,
  useStaticTabs,
  renameTab,
  TabsContext,
  TabsContent,
  useTabsContext,
} from "./Tabs";
import { TabsSelector } from "./TabsSelector";

export type { Tab, TabsProps } from "./Tabs";

type CoreTabsType = typeof CoreTabs;

interface TabsType extends CoreTabsType {
  useStatic: typeof useStaticTabs;
  resetSelection: typeof resetTabSelection;
  rename: typeof renameTab;
  Provider: typeof TabsContext.Provider;
  Content: typeof TabsContent;
  Selector: typeof TabsSelector;
  useContext: typeof useTabsContext;
}

export const Tabs = CoreTabs as TabsType;

Tabs.useStatic = useStaticTabs;
Tabs.resetSelection = resetTabSelection;
Tabs.rename = renameTab;
Tabs.Provider = TabsContext.Provider;
Tabs.Content = TabsContent;
Tabs.Selector = TabsSelector;
