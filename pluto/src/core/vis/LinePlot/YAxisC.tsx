import { PropsWithChildren, ReactElement } from "react";

import { ExtendedVisProvider, useVisElement } from "../Context";

import { LineCProps } from "@/core/vis/Line/LineC";
import { YAxisProps, YAxis } from "@/core/vis/LinePlot/YAxis";

export interface YAxisCProps extends YAxisProps, PropsWithChildren<LineCProps> {}

export const YAxisC = ({ children, ...props }: YAxisCProps): ReactElement => {
  const el = useVisElement(YAxis.TYPE, props);
  return <ExtendedVisProvider key={el.key}>{children}</ExtendedVisProvider>;
};
