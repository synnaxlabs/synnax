import { ranger } from "@synnaxlabs/client";

import { Overview } from "@/feature/range/overview/Overview";
import { TabName } from "@/feature/range/overview/TabName";
import { type Panel } from "@/platform/panel";

export { Overview };

const TAB: Panel.Tab = {
  Content: Overview,
  Name: TabName,
};

export const TABS: Panel.Tabs = {
  [ranger.TYPE_ONTOLOGY_ID.type]: TAB,
};
