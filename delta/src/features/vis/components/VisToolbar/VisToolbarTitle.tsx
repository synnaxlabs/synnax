import { HiChartBar } from "react-icons/hi";

import { ToolbarTitle } from "@/components";

export const VisIcon = HiChartBar;

export const VisToolbarTitle = (): JSX.Element => (
  <ToolbarTitle icon={<VisIcon />}>Visualization</ToolbarTitle>
);
