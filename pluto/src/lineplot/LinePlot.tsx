// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/LinePlot.css";

import {
  box,
  type color,
  deep,
  type destructor,
  direction,
  location,
  xy,
} from "@synnaxlabs/x";
import {
  type CSSProperties,
  type DetailedHTMLProps,
  type HTMLAttributes,
  type PropsWithChildren,
  type ReactElement,
  type Ref,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { context } from "@/context";
import { CSS } from "@/css";
import { useEffectCompare } from "@/hooks";
import { lineplot } from "@/lineplot/aether";
import { useMemoDeepEqual } from "@/memo";
import { type Viewport } from "@/viewport";
import { Canvas } from "@/vis/canvas";
import { grid } from "@/vis/grid";

type HTMLDivProps = DetailedHTMLProps<HTMLAttributes<HTMLDivElement>, HTMLDivElement>;

export interface ContextValue {
  id: string;
  setGridEntry: (meta: grid.Region) => void;
  removeGridEntry: (key: string) => void;
  setLine: (meta: LineSpec) => void;
  removeLine: (key: string) => void;
  lines: LineSpec[];
  setViewport: (viewport: Viewport.UseEvent) => void;
  addViewportHandler: (handler: Viewport.UseHandler) => destructor.Destructor;
  setHold: (hold: boolean) => void;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "LinePlot.Context",
  providerName: "LinePlot.LinePlot",
});
export { useContext };

export const useViewport = (handle: Viewport.UseHandler, component: string): void => {
  const { addViewportHandler } = useContext(component);
  useEffect(() => addViewportHandler(handle), [addViewportHandler, handle]);
};

export const useGridEntry = (meta: grid.Region, component: string): CSSProperties => {
  const { setGridEntry, removeGridEntry } = useContext(component);
  const { key } = meta;
  useEffectCompare(
    () => {
      location.outerZ.parse(meta.loc);
      setGridEntry(meta);
    },
    ([a], [b]) => deep.equal(a, b),
    [meta],
  );
  useEffect(() => () => removeGridEntry(key), []);
  const dir = direction.swap(location.direction(meta.loc));
  const gridArea =
    dir === "x"
      ? `axis-start-${key} / plot-start / axis-end-${key} / plot-end`
      : `plot-start / axis-start-${key} / plot-end / axis-end-${key}`;
  return { gridArea };
};

export interface LineSpec {
  key: string;
  legendGroup: string;
  color: color.Crude;
  label: string;
  visible: boolean;
}

/** Ref handle exposed by LinePlot for imperative access */
export interface LinePlotRef {
  /** Returns the current bounds for all axes */
  getBounds: () => Promise<lineplot.AxesBounds>;
}

type LineState = LineSpec[];

export interface LinePlotProps
  extends
    PropsWithChildren,
    Partial<
      Pick<
        z.input<typeof lineplot.linePlotStateZ>,
        "clearOverScan" | "hold" | "visible"
      >
    >,
    Omit<HTMLDivProps, "ref">,
    Aether.ComponentProps {
  resizeDebounce?: number;
  onHold?: (hold: boolean) => void;
  ref?: Ref<LinePlotRef>;
}

export const LinePlot = ({
  aetherKey,
  style,
  resizeDebounce: debounce = 0,
  clearOverScan = xy.ZERO,
  children,
  hold = false,
  onHold,
  visible,
  ref,
  ...rest
}: LinePlotProps): ReactElement => {
  const [lines, setLines] = useState<LineState>([]);

  const memoProps = useMemoDeepEqual({ clearOverScan, hold, visible });

  const [{ path }, { grid }, setState, methods] = Aether.use({
    aetherKey,
    type: lineplot.LinePlot.TYPE,
    schema: lineplot.linePlotStateZ,
    initialState: {
      container: box.ZERO,
      viewport: box.DECIMAL,
      grid: {},
      ...memoProps,
    },
    methods: lineplot.linePlotMethodsZ,
  });

  // We use a single resize handler for both the container and plotting region because
  // the container is guaranteed to only resize if the plotting region does. This allows
  // us to save a window observer.
  const handleResize = useCallback(
    (container: box.Box) => {
      if (visible) setState((prev) => ({ ...prev, container }));
    },
    [setState, visible],
  );

  const regionRef = Canvas.useRegion(handleResize, { debounce });

  useImperativeHandle(ref, () => ({ getBounds: methods.getBounds }), [
    methods.getBounds,
  ]);

  useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);

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
      const { mode, box, stage } = args;
      if (
        (mode === "pan" && stage !== "start") ||
        mode === "zoom" ||
        (mode === "zoomReset" && stage === "start")
      )
        setState((prev) => ({ ...prev, viewport: box }));
      viewportHandlers.current.forEach((_, handler) => handler(args));
    },
    [setState],
  );

  const setGridEntry = useCallback(
    (meta: grid.Region) => {
      setState((prev) => ({
        ...prev,
        grid: { ...prev.grid, [meta.key]: meta },
      }));
    },
    [setState],
  );

  const removeGridEntry = useCallback(
    (key: string) =>
      setState((prev) => {
        const { [key]: _, ...grid } = prev.grid;
        return { ...prev, grid };
      }),
    [setState],
  );

  const setLine = useCallback(
    (meta: LineSpec) => {
      setLines((prev) => [...prev.filter(({ key }) => key !== meta.key), meta]);
    },
    [setLines],
  );

  const removeLine = useCallback(
    (key: string) => setLines((prev) => prev.filter(({ key: k }) => k !== key)),
    [setLine],
  );

  const cssGrid = useMemo(() => buildPlotGrid(grid), [grid]);

  const setHold = useCallback(
    (hold: boolean) => {
      setState((prev) => ({ ...prev, hold }));
      onHold?.(hold);
    },
    [setState, onHold],
  );

  const id = `line-plot-${aetherKey}`;

  const contextValue = useMemo<ContextValue>(
    () => ({
      lines,
      setGridEntry,
      removeGridEntry,
      setLine,
      removeLine,
      setViewport,
      addViewportHandler,
      setHold,
      id,
    }),
    [
      id,
      lines,
      setGridEntry,
      removeGridEntry,
      setLine,
      removeLine,
      setViewport,
      addViewportHandler,
      setHold,
    ],
  );

  return (
    <div
      id={id}
      className={CSS.B("line-plot")}
      style={{ ...style, ...cssGrid }}
      ref={regionRef}
      {...rest}
    >
      <Context value={contextValue}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </Context>
    </div>
  );
};

const buildPlotGrid = (g: grid.Grid): CSSProperties => {
  const b = CSS.newGridBuilder();
  grid
    .regions("top", g)
    .forEach(({ key, size }) => b.row(`axis-start-${key}`, `axis-end-${key}`, size));
  b.row("plot-start", "plot-end", "minmax(0, 1fr)");
  grid
    .regions("bottom", g)
    .reverse()
    .forEach(({ key, size }) => b.row(`axis-start-${key}`, `axis-end-${key}`, size));
  grid
    .regions("left", g)
    .forEach(({ key, size }) => b.col(`axis-start-${key}`, `axis-end-${key}`, size));
  b.col("plot-start", "plot-end", "minmax(0, 1fr)");
  grid
    .regions("right", g)
    .forEach(({ key, size }) => b.col(`axis-start-${key}`, `axis-end-${key}`, size));
  return b.build();
};
