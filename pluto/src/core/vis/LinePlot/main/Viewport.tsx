import { PropsWithChildren, ReactElement, useCallback, useEffect } from "react";

import { Box, XY } from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { useLinePlotContext } from "@/core/vis/LinePlot/main/LinePlot";
import {
  UseViewportProps,
  Viewport as CoreViewport,
  UseViewportEvent,
} from "@/core/vis/viewport";

export interface ViewportProps extends PropsWithChildren<{}>, UseViewportProps { }

export const Viewport = ({
  children,
  initial = Box.DECIMAL,
  onChange,
  ...props
}: ViewportProps): ReactElement => {
  const { setViewport } = useLinePlotContext("Viewport");

  useEffect(() => setViewport({ box: initial, mode: "zoom", cursor: XY.ZERO, stage: "start" }), [setViewport]);

  const updateViewport = useCallback(
    (e: UseViewportEvent): void => {
      setViewport(e);
      onChange?.(e);
    },
    [onChange, setViewport]
  );

  const maskProps = CoreViewport.use({
    onChange: updateViewport,
    initial,
    ...props,
  });

  return (
    <CoreViewport.Mask className={CSS.BE("line-plot", "viewport")} {...maskProps}>
      {children}
    </CoreViewport.Mask>
  );
};
