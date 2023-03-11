import { Icon } from "@synnaxlabs/media";

import { PageNavLeaf } from "@/components/PageNav";

export const acquireNav: PageNavLeaf = {
  key: "acquire",
  name: "Acquire",
  icon: <Icon.Acquire />,
  children: [
    {
      key: "/acquire/get-started",
      url: "/acquire/get-started",
      name: "Get Started",
    },
    {
      key: "/acquire/create-channels",
      url: "/acquire/create-channels",
      name: "Create Channels",
    },
    {
      key: "/acquire/write-telemetry",
      url: "/acquire/write-telemetry",
      name: "Write Telemetry",
    },
  ],
};
