// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  CSSProperties,
  DetailedHTMLProps,
  HTMLAttributes,
  PropsWithChildren,
  ReactElement,
  createContext,
  memo,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";

import { Box, CrudeOuterLocation, Location } from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/core/aether/main";
import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";
import { X_AXIS_SIZE, Y_AXIS_SIZE } from "@/core/vis/Axis/core";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { UseViewportHandler, Viewport } from "@/core/vis/viewport";

import "@/core/vis/LinePlot/main/LinePlot.css";

type HTMLDivProps = DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement>;

export interface LinePlotContextValue {
  setAxis: (loc: CrudeOuterLocation, key: string) => void;
  deleteAxis: (key: string) => void;
}

const LinePlotContext = createContext<LinePlotContextValue | null>(null);

export const useLinePlotContext = (component: string): LinePlotContextValue => {
  const ctx = useContext(LinePlotContext);
  if (ctx == null)
    throw new Error(`Cannot to use ${component} as a non-child of LinePlot.`);
  return ctx;
};

export const useAxisPosition = (
  loc: CrudeOuterLocation,
  key: string,
  component: string
): CSSProperties => {
  const { setAxis, deleteAxis } = useLinePlotContext(component);
  useEffect(() => {
    Location.strictOuterZ.parse(loc);
    setAxis(loc, key);
    return () => deleteAxis(key);
  }, [setAxis, deleteAxis, loc]);
  const dir = new Location(loc).direction.inverse;
  const gridArea = dir.equals("x")
    ? `axis-start-${key} / plot-start / axis-end-${key} / plot-end`
    : `plot-start / axis-start-${key} / plot-end / axis-end-${key}`;
  return { gridArea };
};

type AxisState = Array<[CrudeOuterLocation, string]>;

export interface LinePlotProps
  extends PropsWithChildren,
    Pick<z.input<typeof AetherLinePlot.stateZ>, "clearOverscan">,
    HTMLDivProps {
  resizeDebounce?: number;
}

export const LinePlot = memo(
  ({
    children,
    style,
    resizeDebounce: debounce = 100,
    ...props
  }: LinePlotProps): ReactElement => {
    const [axes, setAxes] = useState<AxisState>([]);
    const [{ path }, , setState] = Aether.use(
      AetherLinePlot.TYPE,
      AetherLinePlot.stateZ,
      {
        plot: Box.ZERO,
        container: Box.ZERO,
        viewport: Box.DECIMAL,
        ...props,
      }
    );

    const onViewportChange = useCallback<UseViewportHandler>(
      ({ box }) => setState((prev) => ({ ...prev, viewport: box })),
      []
    );

    const { ref: viewportRef, ...viewportProps } = Viewport.use({
      onChange: onViewportChange,
    });

    const containerRef = useRef<HTMLDivElement>(null);

    // We use a single resize handler for both the container and plotting region because
    // the container is guaranteed to only resize if the plotting region does. This allows
    // us to save a window observer.
    const handleResize = useCallback(
      (box: Box) =>
        setState((prev) => {
          const { current: container } = containerRef;
          if (container == null) return prev;
          return {
            ...prev,
            plot: box,
            container: new Box(container),
          };
        }),
      []
    );

    const resizeRef = useResize(handleResize, { debounce });

    const setAxis = useCallback(
      (loc: CrudeOuterLocation, key: string) =>
        setAxes((prev) => [...prev.filter(([, k]) => k !== key), [loc, key]]),
      []
    );

    const deleteAxis = useCallback(
      (key: string) => setAxes((prev) => prev.filter(([, k]) => k !== key)),
      []
    );

    const grid = buildPlotGrid(axes);

    const viewportRefCallback = useCallback(
      (el: HTMLDivElement | null) => {
        viewportRef.current = el;
        resizeRef(el);
      },
      [viewportRef, resizeRef]
    );

    return (
      <div
        className={CSS.B("line-plot")}
        style={{ ...style, ...grid }}
        ref={containerRef}
        {...props}
      >
        <LinePlotContext.Provider value={{ setAxis, deleteAxis }}>
          <Aether.Composite path={path}>{children}</Aether.Composite>
        </LinePlotContext.Provider>
        <Viewport.Mask
          className={CSS.BE("line-plot", "viewport")}
          {...viewportProps}
          ref={viewportRefCallback}
        />
      </div>
    );
  }
);
LinePlot.displayName = "LinePlot";

const buildPlotGrid = (axisCounts: AxisState): CSSProperties => {
  const builder = CSS.newGridBuilder();
  const filterAxisLoc = (loc: CrudeOuterLocation): AxisState =>
    axisCounts.filter(([l]) => l === loc);
  filterAxisLoc("top").forEach(([, key]) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, X_AXIS_SIZE)
  );
  builder.addRow("plot-start", "plot-end", "auto");
  filterAxisLoc("bottom").forEach(([loc, key]) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, X_AXIS_SIZE)
  );
  filterAxisLoc("left").forEach(([, key]) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, Y_AXIS_SIZE)
  );
  builder.addColumn("plot-start", "plot-end", "auto");
  filterAxisLoc("right").forEach(([, key]) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, Y_AXIS_SIZE)
  );
  return builder.build();
};
