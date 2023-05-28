import { ReactElement } from "react";

import { useVisElement } from "../Context";

import { LineProps } from "@/core/vis/Line/WLine";
import { DynamicXYTelemMeta, XYTelemMeta } from "@/core/vis/telem";

export interface LineProps extends Omit<LineProps, "telem"> {
  telem: XYTelemMeta | DynamicXYTelemMeta;
  color: string;
  strokeWidth: number;
}

export const Line = (props: LineProps): ReactElement | null => {
  useVisElement<LineProps>("line", props);
  return null;
};
