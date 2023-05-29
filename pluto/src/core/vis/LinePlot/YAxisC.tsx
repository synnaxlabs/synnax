import { PropsWithChildren, ReactElement } from "react";

import { Optional } from "@synnaxlabs/x";

import { ExtendedVisProvider, useVisElement } from "../Context";

import { Theming } from "@/core/theming";
import { YAxisProps, YAxis } from "@/core/vis/LinePlot/YAxis";

export interface YAxisCProps
  extends PropsWithChildren<
    Optional<
      Omit<YAxisProps, "tickFont">,
      "tickSpacing" | "location" | "type" | "label" | "key"
    >
  > {}

export const YAxisC = ({ children, ...props }: YAxisCProps): ReactElement => {
  const theme = Theming.use();
  const { key } = useVisElement(YAxis.TYPE, {
    tickSpacing: 50,
    location: "bottom",
    type: "linear",
    label: "",
    font: `${theme.typography.tiny.size} ${theme.typography.family}`,
    ...props,
  });
  return <ExtendedVisProvider key={key}>{children}</ExtendedVisProvider>;
};
