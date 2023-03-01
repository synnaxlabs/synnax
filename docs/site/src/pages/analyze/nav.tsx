import { Icon } from "@synnaxlabs/media";

import { PageNavLeaf } from "@/components/PageNav";

export const analyzeNav: PageNavLeaf = {
  key: "analyze",
  name: "Analyze",
  icon: <Icon.Analyze />,
  children: [
    {
      key: "/analyze/get-started",
      url: "/analyze/get-started",
      name: "Get Started",
    },
    {
      key: "/analyze/retrieve-channels",
      url: "/analyze/retrieve-channels",
      name: "Retrieve Channels",
    },
    {
      key: "/analyze/read-telemetry",
      url: "/analyze/read-telemetry",
      name: "Read Telemetry",
    },
  ],
};
