import { PropsWithChildren, ReactElement, useCallback, useEffect } from "react";

import { Box, XY } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Viewport as Core } from "@/viewport";
import { useContext } from "@/vis/lineplot/LinePlot";

export interface ViewportProps extends PropsWithChildren<{}>, Core.UseProps {}

export const Viewport = ({
  children,
  initial = Box.DECIMAL,
  onChange,
  ...props
}: ViewportProps): ReactElement => {
  const { setViewport } = useContext("Viewport");

  useEffect(() => {
    setViewport({ box: initial, mode: "zoom", cursor: XY.ZERO, stage: "start" });
  }, [setViewport, initial]);

  const updateViewport = useCallback(
    (e: Core.UseEvent): void => {
      setViewport(e);
      onChange?.(e);
    },
    [onChange, setViewport]
  );

  const maskProps = Core.use({
    onChange: updateViewport,
    initial,
    ...props,
  });

  return (
    <Core.Mask className={CSS.BE("line-plot", "viewport")} {...maskProps}>
      {children}
    </Core.Mask>
  );
};
