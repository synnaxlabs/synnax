// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CSSProperties,
  type DetailedHTMLProps,
  type HTMLAttributes,
  type PropsWithChildren,
  type ReactElement,
  createContext,
  useCallback,
  useContext as reactUseContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { location, type Destructor, deep, direction, box } from "@synnaxlabs/x";
import { type z } from "zod";

import { Aether } from "@/aether";
import { type Color } from "@/color";
import { CSS } from "@/css";
import { useMemoDeepEqualProps, useResize } from "@/hooks";
import { useEffectCompare } from "@/hooks/useEffectCompare";
import { type Viewport } from "@/viewport";
import { lineplot } from "@/vis/lineplot/aether";
import { type GridPositionSpec, filterGridPositions } from "@/vis/lineplot/aether/grid";

import "@/vis/lineplot/LinePlot.css";

type HTMLDivProps = DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement>;

export interface LinePlotContextValue {
  setAxis: (meta: GridPositionSpec) => void;
  removeAxis: (key: string) => void;
  setLine: (meta: LineSpec) => void;
  removeLine: (key: string) => void;
  lines: LineSpec[];
  setViewport: (viewport: Viewport.UseEvent) => void;
  addViewportHandler: (handler: Viewport.UseHandler) => Destructor;
}

const Context = createContext<LinePlotContextValue | null>(null);

export const useContext = (component: string): LinePlotContextValue => {
  const ctx = reactUseContext(Context);
  if (ctx == null)
    throw new Error(`Cannot to use ${component} as a non-child of LinePlot.`);
  return ctx;
};

export const useViewport = (handle: Viewport.UseHandler): void => {
  const ctx = useContext("Viewport");
  const { addViewportHandler } = ctx;
  useEffect(() => addViewportHandler(handle), [addViewportHandler, handle]);
};

export const useGridPosition = (
  meta: GridPositionSpec,
  component: string,
): CSSProperties => {
  const { setAxis, removeAxis } = useContext(component);
  const { key } = meta;
  useEffectCompare(
    () => {
      location.outer.parse(meta.loc);
      setAxis(meta);
      return () => removeAxis(meta.key);
    },
    ([a], [b]) => deep.equal(a, b),
    [meta],
  );
  const dir = direction.swap(location.direction(meta.loc));
  const gridArea =
    dir === "x"
      ? `axis-start-${key} / plot-start / axis-end-${key} / plot-end`
      : `plot-start / axis-start-${key} / plot-end / axis-end-${key}`;
  return { gridArea };
};

export interface LineSpec {
  key: string;
  color: Color.Crude;
  label: string;
}

type LineState = LineSpec[];

export interface LinePlotProps
  extends PropsWithChildren,
    Pick<z.input<typeof lineplot.linePlotStateZ>, "clearOverscan" | "hold">,
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
    hold,
    ...props
  }): ReactElement => {
    const [lines, setLines] = useState<LineState>([]);

    const aetherMemoProps = useMemoDeepEqualProps({ clearOverscan, hold });

    const [{ path }, { grid }, setState] = Aether.use({
      aetherKey,
      type: lineplot.LinePlot.TYPE,
      schema: lineplot.linePlotStateZ,
      initialState: {
        container: box.ZERO,
        viewport: box.DECIMAL,
        grid: [],
        ...aetherMemoProps,
      },
    });

    useEffect(
      () => setState((prev) => ({ ...prev, ...aetherMemoProps })),
      [aetherMemoProps],
    );

    const viewportHandlers = useRef<Map<Viewport.UseHandler, null>>(new Map());

    const addViewportHandler = useCallback(
      (handler: Viewport.UseHandler) => {
        viewportHandlers.current.set(handler, null);
        return () => viewportHandlers.current.delete(handler);
      },
      [viewportHandlers],
    );

    const setViewport: Viewport.UseHandler = useCallback(
      (args) => {
        const { mode, box } = args;
        setState((prev) => {
          if (["pan", "zoom", "zoomReset"].includes(mode as string))
            return { ...prev, viewport: box };
          return prev;
        });
        viewportHandlers.current.forEach((_, handler) => handler(args));
      },
      [setState],
    );

    // We use a single resize handler for both the container and plotting region because
    // the container is guaranteed to only resize if the plotting region does. This allows
    // us to save a window observer.
    const handleResize = useCallback(
      (container: box.Box) => setState((prev) => ({ ...prev, container })),
      [setState],
    );

    const ref = useResize(handleResize, { debounce });

    const setAxis: LinePlotContextValue["setAxis"] = useCallback(
      (meta: GridPositionSpec) =>
        setState((prev) => ({
          ...prev,
          grid: [...prev.grid.filter(({ key }) => key !== meta.key), meta],
        })),
      [setState],
    );

    const removeAxis = useCallback(
      (key: string) =>
        setState((prev) => ({
          ...prev,
          grid: prev.grid.filter(({ key: k }) => k !== key),
        })),
      [setState],
    );

    const setLine = useCallback(
      (meta: LineSpec) =>
        setLines((prev) => [...prev.filter(({ key }) => key !== meta.key), meta]),
      [setLines],
    );

    const removeLine = useCallback(
      (key: string) => setLines((prev) => prev.filter(({ key: k }) => k !== key)),
      [setLine],
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
        addViewportHandler,
      }),
      [
        lines,
        setAxis,
        removeAxis,
        setLine,
        removeLine,
        setViewport,
        addViewportHandler,
      ],
    );

    return (
      <div
        className={CSS.B("line-plot")}
        style={{ ...style, ...cssGrid }}
        ref={ref}
        {...props}
      >
        <Context.Provider value={contextValue}>
          <Aether.Composite path={path}>{children}</Aether.Composite>
        </Context.Provider>
      </div>
    );
  },
);

const buildPlotGrid = (grid: GridPositionSpec[]): CSSProperties => {
  const builder = CSS.newGridBuilder();
  filterGridPositions("top", grid).forEach(({ key, size }) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, size),
  );
  builder.addRow("plot-start", "plot-end", "minmax(0, 1fr)");
  filterGridPositions("bottom", grid).forEach(({ key, size }) =>
    builder.addRow(`axis-start-${key}`, `axis-end-${key}`, size),
  );
  filterGridPositions("left", grid).forEach(({ key, size }) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, size),
  );
  builder.addColumn("plot-start", "plot-end", "minmax(0, 1fr)");
  filterGridPositions("right", grid).forEach(({ key, size }) =>
    builder.addColumn(`axis-start-${key}`, `axis-end-${key}`, size),
  );
  return builder.build();
};
