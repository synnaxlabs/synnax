import { PropsWithChildren, ReactElement, memo, useEffect } from "react";

import { Optional, ZERO_XY } from "@synnaxlabs/x";

import { Bob } from "@/core/bob/main";
import { Theming } from "@/core/theming";
import {
  XAxisState as WorkerXAxisState,
  XAxis as WorkerXAxis,
} from "@/core/vis/LinePlot/worker";

export interface XAxisCProps
  extends PropsWithChildren,
    Optional<Omit<WorkerXAxisState, "position">, "color" | "font"> {}

export const XAxis = memo(({ children, ...props }: XAxisCProps): ReactElement => {
  const theme = Theming.use();
  const font = `${theme.typography.tiny.size * theme.sizes.base}px ${
    theme.typography.family
  }`;
  const { path } = Bob.useComponent<WorkerXAxisState>(
    WorkerXAxis.TYPE,
    {
      color: theme.colors.gray.p3,
      position: ZERO_XY,
      font,
      ...props,
    },
    "x-axis",
    []
  );
  return <Bob.Composite path={path}>{children}</Bob.Composite>;
});
XAxis.displayName = "XAxisC";
