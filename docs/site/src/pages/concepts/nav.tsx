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
      key: "/concepts/clusters-and-nodes",
      url: "/concepts/clusters-and-nodes",
      name: "Clusters and Nodes",
    },
    {
      key: "/concepts/channels",
      url: "/concepts/channels",
      name: "Channels",
    },
    {
      key: "/concepts/arrays-and-frames",
      url: "/concepts/arrays-and-frames",
      name: "Arrays and Frames",
    },
    {
      key: "/concepts/write-domains",
      url: "/concepts/write-domains",
      name: "Write Domains",
    },
    {
      key: "/concepts/read-ranges",
      url: "/concepts/read-ranges",
      name: "Read Ranges",
    },
  ],
};
