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
  useCallback,
  useContext,
  useRef,
  useState,
} from "react";

import {
  Box,
  Compare,
  CrudeOrder,
  CrudeOuterLocation,
  Deep,
  Location,
} from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/core/aether/main";
import { ColorT } from "@/core/color";
import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";
import { useEffectCompare } from "@/core/hooks/useEffectCompare";
import { Status } from "@/core/std";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { UseViewportHandler, Viewport } from "@/core/vis/viewport";

import "@/core/vis/LinePlot/main/LinePlot.css";

type HTMLDivProps = DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement>;

export interface LinePlotContextValue {
  setAxis: (meta: GridPositionMeta) => void;
  removeAxis: (key: string) => void;
  setLine: (meta: LineMeta) => void;
  removeLine: (key: string) => void;
  lines: LineMeta[];
}

const LinePlotContext = createContext<LinePlotContextValue | null>(null);

export const useLinePlotContext = (component: string): LinePlotContextValue => {
  const ctx = useContext(LinePlotContext);
  if (ctx == null)
    throw new Error(`Cannot to use ${component} as a non-child of LinePlot.`);
  return ctx;
};

export const useGridPosition = (
  meta: GridPositionMeta,
  component: string
): CSSProperties => {
  const { setAxis, removeAxis } = useLinePlotContext(component);
  const { key } = meta;
  useEffectCompare(
    () => {
      Location.strictOuterZ.parse(meta.loc);
      setAxis(meta);
      return () => removeAxis(meta.key);
    },
    ([a], [b]) => Deep.equal(a, b),
    [meta]
  );
  const dir = new Location(meta.loc).direction.inverse;
  const gridArea = dir.equals("x")
    ? `axis-start-${key} / plot-start / axis-end-${key} / plot-end`
    : `plot-start / axis-start-${key} / plot-end / axis-end-${key}`;
  return { gridArea };
};

export interface LineMeta {
  key: string;
  color: ColorT;
  label: string;
}

export interface GridPositionMeta {
  key: string;
  size: number;
  order: CrudeOrder;
  loc: CrudeOuterLocation;
}

type AxisState = GridPositionMeta[];
type LineState = LineMeta[];

export interface LinePlotProps
  extends PropsWithChildren,
    Pick<z.input<typeof AetherLinePlot.z>, "clearOverscan">,
    HTMLDivProps {
  resizeDebounce?: number;
}

export const LinePlot = Aether.wrap<LinePlotProps>(
  "LinePlot",
  ({
    aetherKey,
    children,
    style,
    resizeDebounce: debounce = 100,
    clearOverscan,
    ...props
  }): ReactElement => {
    const [axes, setAxes] = useState<AxisState>([]);
    const [lines, setLines] = useState<LineState>([]);
    const [{ path }, { error }, setState] = Aether.use({
      aetherKey,
      type: AetherLinePlot.TYPE,
      schema: AetherLinePlot.z,
      initialState: {
        plot: Box.ZERO,
        container: Box.ZERO,
        viewport: Box.DECIMAL,
        clearOverscan,
        ...props,
      },
    });

    const handleViewportChange = useCallback<UseViewportHandler>(
      ({ mode, box }) =>
        setState((prev) => {
          if (["pan", "zoom", "zoomReset"].includes(mode as string))
            return { ...prev, viewport: box };
          return prev;
        }),
      []
    );

    const { ref: viewportRef, ...viewportProps } = Viewport.use({
      onChange: handleViewportChange,
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

    const setAxis: LinePlotContextValue["setAxis"] = useCallback(
      (meta: GridPositionMeta) =>
        setAxes((prev) => [...prev.filter(({ key }) => key !== meta.key), meta]),
      []
    );

    const removeAxis = useCallback(
      (key: string) => setAxes((prev) => prev.filter(({ key: k }) => k !== key)),
      []
    );

    const setLine = useCallback(
      (meta: LineMeta) =>
        setLines((prev) => [...prev.filter(({ key }) => key !== meta.key), meta]),
      []
    );

    const removeLine = useCallback(
      (key: string) => setLines((prev) => prev.filter(({ key: k }) => k !== key)),
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
        {error != null && (
          <Status.Text.Centered variant="error" style={{ position: "absolute" }}>
            {error}
          </Status.Text.Centered>
        )}
        <LinePlotContext.Provider
          value={{ lines, setAxis, removeAxis, setLine, removeLine }}
        >
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

const buildPlotGrid = (axisCounts: AxisState): CSSProperties => {
  const builder = CSS.newGridBuilder();
  const filterAxisLoc = (loc: CrudeOuterLocation): AxisState =>
    axisCounts
      .filter(({ loc: l }) => l === loc)
      .sort((a, b) => Compare.order(a.order, b.order));
  filterAxisLoc("top").forEach(({ key, size }) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  builder.addRow("plot-start", "plot-end", "minmax(0, 1fr)");
  filterAxisLoc("bottom").forEach(({ key, size }) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  filterAxisLoc("left").forEach(({ key, size }) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  builder.addColumn("plot-start", "plot-end", "minmax(0, 1fr)");
  filterAxisLoc("right").forEach(({ key, size }) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  return builder.build();
};
