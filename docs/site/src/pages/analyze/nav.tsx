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
      key: "/analyze/retrieving-channels",
      url: "/analyze/retrieving-channels",
      name: "Retrieving Channels",
    },
    {
      key: "/analyze/reading-telemetry",
      url: "/analyze/reading-telemetry",
      name: "Reading Telemetry",
    },
  ],
};
