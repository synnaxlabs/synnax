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
  useMemo,
  useRef,
  useState,
} from "react";

import { Box, Deep, Location } from "@synnaxlabs/x";
import { z } from "zod";

import { GridPositionMeta, filterAxisLoc } from "../aether/LinePlot";

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
    style,
    resizeDebounce: debounce = 0,
    clearOverscan,
    children,
    ...props
  }): ReactElement => {
    const [lines, setLines] = useState<LineState>([]);
    const [{ path }, { error, grid }, setState] = Aether.use({
      aetherKey,
      type: AetherLinePlot.TYPE,
      schema: AetherLinePlot.z,
      initialState: {
        plot: Box.ZERO,
        container: Box.ZERO,
        viewport: Box.DECIMAL,
        grid: [],
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
      (box: Box) => {
        setState((prev) => {
          const { current: container } = containerRef;
          if (container == null) return prev;
          return {
            ...prev,
            plot: box,
            container: new Box(container),
          };
        });
      },
      [setState]
    );

    const resizeRef = useResize(handleResize, { debounce });

    const setAxis: LinePlotContextValue["setAxis"] = useCallback(
      (meta: GridPositionMeta) =>
        setState((prev) => ({
          ...prev,
          grid: [...prev.grid.filter(({ key }) => key !== meta.key), meta],
        })),
      [setState]
    );

    const removeAxis = useCallback(
      (key: string) =>
        setState((prev) => ({
          ...prev,
          grid: prev.grid.filter(({ key: k }) => k !== key),
        })),
      [setState]
    );

    const setLine = useCallback(
      (meta: LineMeta) =>
        setLines((prev) => [...prev.filter(({ key }) => key !== meta.key), meta]),
      [setLines]
    );

    const removeLine = useCallback(
      (key: string) => setLines((prev) => prev.filter(({ key: k }) => k !== key)),
      [setLine]
    );

    const cssGrid = buildPlotGrid(grid);

    const viewportRefCallback = useCallback(
      (el: HTMLDivElement | null) => {
        viewportRef.current = el;
        resizeRef(el);
      },
      [viewportRef, resizeRef]
    );

    const contextValue = useMemo<LinePlotContextValue>(
      () => ({ lines, setAxis, removeAxis, setLine, removeLine }),
      [lines, setAxis, removeAxis, setLine, removeLine]
    );

    return (
      <div
        className={CSS.B("line-plot")}
        style={{ ...style, ...cssGrid }}
        ref={containerRef}
        {...props}
      >
        {error != null && (
          <Status.Text.Centered variant="error" style={{ position: "absolute" }}>
            {error}
          </Status.Text.Centered>
        )}
        <LinePlotContext.Provider value={contextValue}>
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

const buildPlotGrid = (grid: GridPositionMeta[]): CSSProperties => {
  const builder = CSS.newGridBuilder();
  filterAxisLoc("top", grid).forEach(({ key, size }) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  builder.addRow("plot-start", "plot-end", "minmax(0, 1fr)");
  filterAxisLoc("bottom", grid).forEach(({ key, size }) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  filterAxisLoc("left", grid).forEach(({ key, size }) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  builder.addColumn("plot-start", "plot-end", "minmax(0, 1fr)");
  filterAxisLoc("right", grid).forEach(({ key, size }) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  return builder.build();
};
