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
      key: "/acquire/creating-channels",
      url: "/acquire/creating-channels",
      name: "Creating Channels",
    },
    {
      key: "/acquire/writing-telemetry",
      url: "/acquire/writing-telemetry",
      name: "Writing Telemetry",
    },
  ],
};
