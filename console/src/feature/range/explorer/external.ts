import { Explorer } from "@/feature/range/explorer/Explorer";
import { TabName } from "@/feature/range/explorer/TabName";
import { Panel } from "@/platform/panel";

export { Explorer };

export const TAB: Panel.Tab = {
  Name: TabName,
  Content: Explorer,
};

export const TAB_TYPE = "range_explorer";

export const TABS: Panel.Tabs = { [TAB_TYPE]: TAB };

export const useOpenTab = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return () => openTab({ variant: "view", type: TAB_TYPE, args: {} });
};
