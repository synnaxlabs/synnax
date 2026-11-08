// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/Frame.css";

import { context } from "@synnaxlabs/lyra/context";
import { useEffectCompare } from "@synnaxlabs/lyra/hooks";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import {
  type bounds,
  box,
  type color,
  type CrudeTimeSpan,
  deep,
  type destructor,
  direction,
  location,
  xy,
  zod,
} from "@synnaxlabs/x";
import {
  type CSSProperties,
  type DetailedHTMLProps,
  type HTMLAttributes,
  type PropsWithChildren,
  type ReactElement,
  type Ref,
  type RefObject,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { lineplot } from "@/lineplot/aether";
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
  viewportElement: HTMLElement | null;
  setViewportElement: (element: HTMLElement | null) => void;
  addViewportHandler: (handler: Viewport.UseHandler) => destructor.Destructor;
  /**
   * Zooms the x axis to the decimal bounds x, shows the full y range, and pauses the
   * plot. A second call with the same key returns to the view and pause state from
   * before the first call, unless the view changed between them. Calls nest, so each
   * one returns one level.
   */
  toggleZoom: (key: string, x: bounds.Bounds) => void;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "LinePlot.Context",
  providerName: "LinePlot.Frame",
});

export interface FetchContextValue {
  loading: boolean;
  fetching: boolean;
  loadingMessage?: string;
}

// Fetch state changes on every pan and zoom, so it has a context of its own.
const [FetchContext, useFetchContext] = context.create<FetchContextValue>({
  displayName: "LinePlot.FetchContext",
  providerName: "LinePlot.Frame",
});

export { useContext, useFetchContext };

export const useViewport = (handle: Viewport.UseHandler, component: string): void => {
  const { addViewportHandler } = useContext(component);
  useEffect(() => addViewportHandler(handle), [addViewportHandler, handle]);
};

export const useGridEntry = (meta: grid.Region, component: string): CSSProperties => {
  const { setGridEntry, removeGridEntry } = useContext(component);
  const { key } = meta;
  useEffectCompare(
    () => {
      zod.parse(location.outerZ, meta.loc, { label: "axis location" });
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

/** Ref handle exposed by Frame for imperative access */
export interface FrameRef {
  /** Returns the current bounds for all axes */
  getBounds: () => Promise<lineplot.AxesBounds>;
}

type LineState = LineSpec[];

interface SavedView {
  key: string;
  viewport: box.Box;
  hold: boolean;
}

const changesView = ({ mode, stage }: Viewport.UseEvent): boolean =>
  (mode === "pan" && stage !== "start") ||
  mode === "zoom" ||
  (mode === "zoomReset" && stage === "start");

export interface FrameProps
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
  resizeDebounce?: CrudeTimeSpan;
  onHold?: (hold: boolean) => void;
  loadingMessage?: string;
  ref?: Ref<FrameRef>;
  /**
   * The plot's viewport. toggleZoom moves it to each view it sets, and a change of hold
   * from true to false resets it.
   */
  viewportRef?: RefObject<Viewport.UseRefValue | null>;
}

export const Frame = ({
  aetherKey,
  style,
  resizeDebounce: debounce = 0,
  clearOverScan = xy.ZERO,
  children,
  hold = false,
  onHold,
  visible,
  loadingMessage,
  ref,
  viewportRef,
  ...rest
}: FrameProps): ReactElement => {
  const [lines, setLines] = useState<LineState>([]);
  const [viewportElement, setViewportElement] = useState<HTMLElement | null>(null);

  const memoProps = useMemoDeepEqual({ clearOverScan, hold, visible });

  const [{ path }, { grid, loading, fetching }, setState, methods] = Aether.use({
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
  const handleResize = (container: box.Box) => {
    if (visible) setState((prev) => ({ ...prev, container }));
  };

  const regionRef = Canvas.useRegion(handleResize, { debounce });

  useImperativeHandle(ref, () => ({ getBounds: methods.getBounds }), [
    methods.getBounds,
  ]);

  // Worker pushes can carry an older view, so the frame tracks the last one it set.
  const appliedRef = useRef<box.Box>(box.DECIMAL);
  const holdRef = useRef(hold);
  const holdPropRef = useRef(hold);

  useEffect(() => {
    // A release from toggleZoom already cleared holdRef and keeps the view it restored.
    const released = holdPropRef.current && !memoProps.hold && holdRef.current;
    holdPropRef.current = memoProps.hold;
    holdRef.current = memoProps.hold;
    setState((prev) => ({ ...prev, ...memoProps }));
    if (released) viewportRef?.current?.reset();
  }, [memoProps]);

  const viewportHandlers = useRef<Map<Viewport.UseHandler, null>>(new Map());

  const addViewportHandler = (handler: Viewport.UseHandler) => {
    viewportHandlers.current.set(handler, null);
    return () => viewportHandlers.current.delete(handler);
  };

  const zoomHistory = useRef<SavedView[]>([]);

  const applyViewport = (args: Viewport.UseEvent) => {
    if (changesView(args)) {
      appliedRef.current = args.box;
      setState((prev) => ({ ...prev, viewport: args.box }));
    }
    viewportHandlers.current.forEach((_, handler) => handler(args));
  };

  const setViewport: Viewport.UseHandler = (args) => {
    if (changesView(args)) zoomHistory.current = [];
    applyViewport(args);
  };

  const setGridEntry = (meta: grid.Region) => {
    setState((prev) => ({
      ...prev,
      grid: { ...prev.grid, [meta.key]: meta },
    }));
  };

  const removeGridEntry = (key: string) =>
    setState((prev) => {
      const grid = { ...prev.grid };
      delete grid[key];
      return { ...prev, grid };
    });

  const setLine = (meta: LineSpec) => {
    setLines((prev) => [...prev.filter(({ key }) => key !== meta.key), meta]);
  };

  const removeLine = (key: string) =>
    setLines((prev) => prev.filter(({ key: k }) => k !== key));

  const cssGrid = buildPlotGrid(grid);

  const frameStyle = { ...style, ...cssGrid };

  const setHold = (hold: boolean) => {
    holdRef.current = hold;
    setState((prev) => ({ ...prev, hold }));
    onHold?.(hold);
  };

  const toggleZoom = (key: string, x: bounds.Bounds) => {
    const history = zoomHistory.current;
    const back = history.at(-1)?.key === key ? history.pop() : undefined;
    if (back == null)
      history.push({ key, viewport: appliedRef.current, hold: holdRef.current });
    const next =
      back?.viewport ??
      box.reRoot(
        box.construct({ x: x.lower, y: 0 }, { x: x.upper, y: 1 }),
        box.DECIMAL.root,
      );
    viewportRef?.current?.sync(next);
    applyViewport({ box: next, mode: "zoom", cursor: xy.ZERO, stage: "start" });
    setHold(back?.hold ?? true);
  };

  const id = `line-plot-${aetherKey}`;

  const contextValue: ContextValue = {
    lines,
    setGridEntry,
    removeGridEntry,
    setLine,
    removeLine,
    setViewport,
    viewportElement,
    setViewportElement,
    addViewportHandler,
    toggleZoom,
    id,
  };

  const fetchContextValue: FetchContextValue = { loading, fetching, loadingMessage };

  return (
    <div
      id={id}
      className={CSS.B("line-plot")}
      style={frameStyle}
      ref={regionRef}
      {...rest}
    >
      <Context value={contextValue}>
        <FetchContext value={fetchContextValue}>
          <Aether.Composite path={path}>{children}</Aether.Composite>
        </FetchContext>
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
