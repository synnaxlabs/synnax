import { PropsWithChildren, ReactElement } from "react";

import { ExtendedVisProvider, useVisElement } from "../Context";

import { XAxisCProps } from "./XAxisC";

import { LinePlotProps } from "@/core/vis/LinePlot/LinePlot";

export interface LinePlotCProps extends LinePlotProps, PropsWithChildren<XAxisCProps> {}

export const LinePlotC = (props: LinePlotCProps): ReactElement => {
  const { key } = useVisElement("line", props);
  return <ExtendedVisProvider key={key}></ExtendedVisProvider>;
};
