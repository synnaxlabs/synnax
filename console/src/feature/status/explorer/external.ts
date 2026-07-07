import { Explorer } from "@/feature/status/explorer/Explorer";
import { TabName } from "@/feature/status/explorer/TabName";
import { Panel } from "@/platform/panel";

export { Explorer };

export const TAB_TYPE = "status_explorer";

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
