import { ReactElement } from "react";

import { useVisElement } from "../Context";

import { LineProps } from "@/core/vis/Line/core";

export interface LineCProps extends LineProps {}

export const LineC = (props: LineCProps): ReactElement | null => {
  useVisElement<LineProps>("line", props);
  return null;
};
