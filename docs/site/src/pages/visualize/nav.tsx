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
    {
      key: "/visualize/connect-a-cluster",
      url: "/visualize/connect-a-cluster",
      name: "Connect a Cluster",
    },
    {
      key: "/visualize/browse-resources",
      url: "/visualize/browse-resources",
      name: "Browse Resources",
    },
    {
      key: "/visualize/define-a-range",
      url: "/visualize/define-a-range",
      name: "Define a Range",
    },
    {
      key: "/visualize/create-a-visualization",
      url: "/visualize/create-a-visualization",
      name: "Create a Visualization",
    },
    {
      key: "/visualize/split-and-resize-tabs",
      url: "/visualize/split-and-resize-tabs",
      name: "Split and Resize Tabs",
    },
  ],
};
