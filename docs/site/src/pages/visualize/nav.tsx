import { Icon } from "@synnaxlabs/media";

import { PageNavLeaf } from "@/components/PageNav";

export const visualizeNav: PageNavLeaf = {
  key: "visualize",
  name: "Visualize",
  icon: <Icon.Visualize />,
  children: [
    {
      key: "/visualize/get-started",
      url: "/visualize/get-started",
      name: "Get Started",
    },
  ],
};
