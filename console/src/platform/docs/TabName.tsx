import { Icon } from "@synnaxlabs/pluto";

import { Panel } from "@/platform/panel";

export const TabName: Panel.TabName = Panel.createStaticTabName({
  name: "Documentation",
  icon: <Icon.QuestionMark />,
});
