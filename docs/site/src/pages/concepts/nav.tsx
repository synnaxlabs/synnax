import { Icon } from "@synnaxlabs/media";

import { PageNavLeaf } from "@/components/PageNav";

export const conceptsNav: PageNavLeaf = {
  key: "concepts",
  name: "Concepts",
  icon: <Icon.Concepts />,
  children: [
    {
      key: "/concepts/overview",
      url: "/concepts/overview",
      name: "Overview",
    },
    {
      key: "/concepts/channels",
      url: "/concepts/channels",
      name: "Channels",
    },
    {
      key: "/concepts/writing-domains",
      url: "/concepts/writing-domains",
      name: "Writing Domains",
    },
    {
      key: "/concepts/frames",
      url: "/concepts/frames",
      name: "Frames",
    },
    {
      key: "/concepts/ranges",
      url: "/concepts/ranges",
      name: "Ranges",
    },
    {
      key: "/concepts/clusters-and-nodes",
      url: "/concepts/clusters-and-nodes",
      name: "Clusters and Nodes",
    },
  ],
};
