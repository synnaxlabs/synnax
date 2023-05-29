import { PropsWithChildren, ReactElement } from "react";

import { Optional } from "@synnaxlabs/x";

import { ExtendedVisProvider, useVisElement } from "../Context";

import { Theming } from "@/core/theming";
import { XAxisProps, XAxis } from "@/core/vis/LinePlot/XAxis";

export interface XAxisCProps
  extends PropsWithChildren,
    Optional<
      Omit<XAxisProps, "tickFont">,
      "tickSpacing" | "location" | "type" | "label" | "key"
    > {}

export const XAxisC = ({ children, ...props }: XAxisCProps): ReactElement => {
  const theme = Theming.use();
  const { key } = useVisElement(XAxis.TYPE, {
    tickSpacing: 50,
    location: "bottom",
    type: "linear",
    label: "",
    font: `${theme.typography.tiny.size} ${theme.typography.family}`,
    ...props,
  });
  return <ExtendedVisProvider key={key}>{children}</ExtendedVisProvider>;
};
