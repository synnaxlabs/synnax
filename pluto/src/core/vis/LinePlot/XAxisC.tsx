import { PropsWithChildren, ReactElement } from "react";

import { ExtendedVisProvider, useVisElement } from "../Context";

import { YAxisCProps } from "./YAxisC";

import { XAxisProps, XAxis } from "@/core/vis/LinePlot/XAxis";

export interface XAxisCProps extends XAxisProps, PropsWithChildren<YAxisCProps> {}

export const XAxisC = ({ children, ...props }: XAxisCProps): ReactElement => {
  const { key } = useVisElement(XAxis.TYPE, props);
  return <ExtendedVisProvider key={key}>{children}</ExtendedVisProvider>;
};
