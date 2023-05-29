import { PropsWithChildren, ReactElement, useCallback, useState } from "react";

import { Box } from "@synnaxlabs/x";

import { ExtendedVisProvider, useVisElement } from "../Context";

import { LinePlotProps } from "./LinePlot";

import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";

export interface LinePlotCProps extends PropsWithChildren {}

export const LinePlotC = ({ children, ...props }: LinePlotCProps): ReactElement => {
  const [container, setContainer] = useState<Box>(Box.ZERO);

  const { key } = useVisElement<Omit<LinePlotProps, "key">>("line", {
    region: container,
    viewport: Box.DECIMAL,
    clearOverscan: 15,
  });

  const handleResize = useCallback((box: Box) => setContainer(box), [setContainer]);

  const resizeRef = useResize(handleResize, { debounce: 100 });

  return (
    <ExtendedVisProvider key={key}>
      <div className={CSS.B("line-plot")} ref={resizeRef}>
        {children}
      </div>
    </ExtendedVisProvider>
  );
};
