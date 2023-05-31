import { PropsWithChildren, ReactElement, memo } from "react";

import { Optional, ZERO_XY } from "@synnaxlabs/x";

import { useAxisPosition } from "./LinePlot";

import { Bob } from "@/core/bob/main";
import { useResize } from "@/core/hooks";
import { Theming } from "@/core/theming";
import {
  XAxisState as WorkerXAxisState,
  XAxis as WorkerXAxis,
} from "@/core/vis/LinePlot/worker";

export interface XAxisCProps
  extends PropsWithChildren,
    Optional<Omit<WorkerXAxisState, "position">, "color" | "font"> {}

export const XAxis = memo(
  ({ children, location = "bottom", ...props }: XAxisCProps): ReactElement => {
    const theme = Theming.use();
    const font = `${theme.typography.tiny.size * theme.sizes.base}px ${
      theme.typography.family
    }`;
    const {
      key,
      path,
      state: [, setState],
    } = Bob.useComponent<WorkerXAxisState>(
      WorkerXAxis.TYPE,
      {
        color: theme.colors.gray.p3,
        position: ZERO_XY,
        font,
        location,
        ...props,
      },
      "x-axis"
    );
    const gridStyle = useAxisPosition(location, key);
    const resizeRef = useResize(
      (box) => {
        setState((state) => ({
          ...state,
          position: box.topLeft,
        }));
      },
      { debounce: 100 }
    );
    return (
      <Bob.Composite path={path}>
        <div
          className="x-axis"
          style={{ ...gridStyle, backgroundColor: "var(--pluto-gray-z)" }}
          ref={resizeRef}
        />
        {children}
      </Bob.Composite>
    );
  }
);
XAxis.displayName = "XAxisC";
