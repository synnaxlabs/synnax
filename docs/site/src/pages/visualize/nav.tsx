import { HiChartBar } from "react-icons/hi";

import { PageNavLeaf } from "@/components/PageNav";

export const visualizeNav: PageNavLeaf = {
  key: "visualize",
  name: "Visualize",
  icon: <HiChartBar />,
  children: [
    {
      key: "/visualize/get-started",
      url: "/visualize/get-started",
      name: "Get Started",
    },
  ],
};
