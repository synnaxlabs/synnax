import { PropsWithChildren, ReactElement, memo } from "react";

import { Optional, ZERO_XY } from "@synnaxlabs/x";

import { Bob } from "@/core/bob/main";
import { Theming } from "@/core/theming";
import {
  YAxisState as WorkerYAxisState,
  YAxis as WorkerYAxis,
} from "@/core/vis/LinePlot/worker";

export interface YAxisProps
  extends PropsWithChildren,
    Optional<Omit<WorkerYAxisState, "position">, "color" | "font"> {}

export const YAxis = memo(({ children, ...props }: YAxisProps): ReactElement => {
  const theme = Theming.use();
  const { path } = Bob.useComponent<WorkerYAxisState>(
    WorkerYAxis.TYPE,
    {
      position: ZERO_XY,
      color: theme.colors.gray.p2,
      font: `${theme.typography.tiny.size}px ${theme.typography.family}`,
      ...props,
    },
    [],
    "y-axis"
  );
  return <Bob.Composite path={path}>{children}</Bob.Composite>;
});
YAxis.displayName = "YAxisC";
