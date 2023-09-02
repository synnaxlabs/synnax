import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useLayoutEffect,
} from "react";

import { Box, XY } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Viewport as Core } from "@/viewport";
import { useContext } from "@/vis/lineplot/LinePlot";

import "@/vis/lineplot/Viewport.css";

export interface ViewportProps extends PropsWithChildren, Core.UseProps {}

export const selectViewportEl = (el: HTMLElement | null): Element | null =>
  el == null
    ? null
    : document.querySelectorAll(".pluto-line-plot__viewport")[0] ?? null;

export const Viewport = ({
  children,
  initial = Box.DECIMAL,
  onChange,
  ...props
}: ViewportProps): ReactElement => {
  const { setViewport } = useContext("Viewport");

  useLayoutEffect(() => {
    setViewport({ box: initial, mode: "zoom", cursor: XY.ZERO, stage: "start" });
  }, [setViewport, initial]);

  const handleChange = useCallback(
    (e: Core.UseEvent): void => {
      setViewport(e);
      onChange?.(e);
    },
    [onChange, setViewport],
  );

  const maskProps = Core.use({
    onChange: handleChange,
    initial,
    ...props,
  });

  return (
    <Core.Mask className={CSS.BE("line-plot", "viewport")} {...maskProps}>
      {children}
    </Core.Mask>
  );
};
