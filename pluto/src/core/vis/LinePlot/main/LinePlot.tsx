import { PropsWithChildren, ReactElement, useCallback, useState } from "react";

import { Box } from "@synnaxlabs/x";

import { Bob } from "@/core/bob/main";
import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";
import {
  LinePlot as WorkerLinePlot,
  LinePlotState as WorkerLinePlotState,
} from "@/core/vis/LinePlot/worker";

export interface LinePlotCProps
  extends PropsWithChildren,
    Pick<WorkerLinePlotState, "clearOverscan"> {}

export const LinePlotC = ({ children, ...props }: LinePlotCProps): ReactElement => {
  const [region, setRegion] = useState<Box>(Box.ZERO);
  const [viewport] = useState<Box>(Box.DECIMAL);

  const { path } = Bob.useComponent<WorkerLinePlotState>(
    WorkerLinePlot.TYPE,
    {
      plottingRegion: region,
      viewport,
      ...props,
    },
    "line-plot"
  );

  const handleResize = useCallback((box: Box) => setRegion(box), [setRegion]);

  const resizeRef = useResize(handleResize, { debounce: 100 });

  return (
    <Bob.Composite path={path}>
      <div className={CSS.B("line-plot")} ref={resizeRef}>
        {children}
      </div>
    </Bob.Composite>
  );
};
