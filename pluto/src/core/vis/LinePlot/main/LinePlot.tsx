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
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { Box, Deep, Destructor, Location } from "@synnaxlabs/x";
import { z } from "zod";

import { UseViewportEvent, UseViewportHandler } from "../../viewport";

import { Aether } from "@/core/aether/main";
import { CrudeColor } from "@/core/color";
import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";
import { useEffectCompare } from "@/core/hooks/useEffectCompare";
import { Status } from "@/core/std";
import { AetherLinePlot } from "@/core/vis/LinePlot/aether";
import { GridPositionMeta, filterGridPositions } from "@/core/vis/LinePlot/aether/grid";

import "@/core/vis/LinePlot/main/LinePlot.css";

type HTMLDivProps = DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement>;

export interface LinePlotContextValue {
  setAxis: (meta: GridPositionMeta) => void;
  removeAxis: (key: string) => void;
  setLine: (meta: LineMeta) => void;
  removeLine: (key: string) => void;
  lines: LineMeta[];
  setViewport: (viewport: UseViewportEvent) => void;
  addViewportHandler: (handler: UseViewportHandler) => Destructor;
}

const LinePlotContext = createContext<LinePlotContextValue | null>(null);

export const useLinePlotViewport = (handle: UseViewportHandler): void => {
  const ctx = useContext(LinePlotContext);
  if (ctx == null)
    throw new Error(
      "Cannot use useLinePlotViewportHandler as a non-child of LinePlot."
    );
  const { addViewportHandler } = ctx;
  useEffect(() => addViewportHandler(handle), [addViewportHandler, handle]);
};

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
  color: CrudeColor;
  label: string;
}

type LineState = LineMeta[];

export interface LinePlotProps
  extends PropsWithChildren,
    Pick<z.input<typeof AetherLinePlot.stateZ>, "clearOverscan">,
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
      schema: AetherLinePlot.stateZ,
      initialState: {
        container: Box.ZERO,
        viewport: Box.DECIMAL,
        grid: [],
        clearOverscan,
        ...props,
      },
    });
    const viewportHandlers = useRef<Map<UseViewportHandler, null>>(new Map());

    const addViewporHandler = useCallback(
      (handler: UseViewportHandler) => {
        viewportHandlers.current.set(handler, null);
        return () => viewportHandlers.current.delete(handler);
      },
      [viewportHandlers]
    );

    const setViewport: UseViewportHandler = useCallback(
      (args) => {
        const { mode, box } = args;
        setState((prev) => {
          if (["pan", "zoom", "zoomReset"].includes(mode as string))
            return { ...prev, viewport: box };
          return prev;
        });
        viewportHandlers.current.forEach((_, handler) => handler(args));
      },
      [setState]
    );

    // We use a single resize handler for both the container and plotting region because
    // the container is guaranteed to only resize if the plotting region does. This allows
    // us to save a window observer.
    const handleResize = useCallback(
      (container: Box) => setState((prev) => ({ ...prev, container })),
      [setState]
    );

    const ref = useResize(handleResize, { debounce });

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

    const contextValue = useMemo<LinePlotContextValue>(
      () => ({
        lines,
        setAxis,
        removeAxis,
        setLine,
        removeLine,
        setViewport,
        addViewportHandler: addViewporHandler,
      }),
      [lines, setAxis, removeAxis, setLine, removeLine, setViewport]
    );

    return (
      <div
        className={CSS.B("line-plot")}
        style={{ ...style, ...cssGrid }}
        ref={ref}
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
      </div>
    );
  }
);

const buildPlotGrid = (grid: GridPositionMeta[]): CSSProperties => {
  const builder = CSS.newGridBuilder();
  filterGridPositions("top", grid).forEach(({ key, size }) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  builder.addRow("plot-start", "plot-end", "minmax(0, 1fr)");
  filterGridPositions("bottom", grid).forEach(({ key, size }) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  filterGridPositions("left", grid).forEach(({ key, size }) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  builder.addColumn("plot-start", "plot-end", "minmax(0, 1fr)");
  filterGridPositions("right", grid).forEach(({ key, size }) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, size)
  );
  return builder.build();
};
