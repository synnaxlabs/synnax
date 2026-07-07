import { Explorer } from "@/feature/arc/explorer/Explorer";
import { TabName } from "@/feature/arc/explorer/TabName";
import { Panel } from "@/platform/panel";

export { Explorer };

export const TAB_TYPE = "arc_explorer";

const TAB: Panel.Tab = {
  Content: Explorer,
  Name: TabName,
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};

export const useOpenTab = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return () => openTab({ variant: "view", type: TAB_TYPE, args: {} });
};
