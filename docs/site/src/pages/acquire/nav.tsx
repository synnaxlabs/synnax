import { FaStream } from "react-icons/fa";

import { PageNavLeaf } from "@/components/PageNav";

export const acquireNav: PageNavLeaf = {
  key: "acquire",
  name: "Acquire",
  icon: <FaStream />,
  children: [
    {
      key: "/acquire/get-started",
      url: "/acquire/get-started",
      name: "Get Started",
    },
  ],
};
