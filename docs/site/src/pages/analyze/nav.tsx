import { FaBezierCurve } from "react-icons/fa";

import { PageNavLeaf } from "@/components/PageNav";

export const analyzeNav: PageNavLeaf = {
  key: "analyze",
  name: "Analyze",
  icon: <FaBezierCurve />,
  children: [
    {
      key: "/analyze/get-started",
      url: "/analyze/get-started",
      name: "Get Started",
    },
  ],
};
