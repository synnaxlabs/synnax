// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/Frame.css";

import type { lineplot as lineplotClient } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/lyra/context";
import { useEffectCompare, useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Status } from "@synnaxlabs/lyra/status";
import {
  bounds,
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
import type { z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { lineplot } from "@/lineplot/aether";
import type { Viewport } from "@/viewport";
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
  /** The key of the last toggleZoom call that the view still shows, or null. */
  zoomedKey: string | null;
  /**
   * Opens or closes the break of x1 under a click at clientX, with the same times kept
   * in view. A click on the plot finds only closed breaks.
   */
  toggleBreak: (clientX: number, source: lineplot.BreakSource) => void;
  /** Closes every break of x1, with the view left as it is. */
  closeBreaks: () => void;
  /**
   * Resolves to the x bounds of the segment of x1 at x, in the decimal space of the
   * view, so that a select drag that starts there stops at its seams.
   */
  selectBound: (x: number) => Promise<bounds.Bounds>;
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
  /** The y axis of the line. */
  axis?: lineplotClient.AxisKey;
}

/** Ref handle exposed by Frame for imperative access */
export interface FrameRef {
  /**
   * Returns the current bounds for all axes. On a relative x axis the x bounds are
   * spans from zero, not timestamps.
   */
  getBounds: () => Promise<lineplot.AxesBounds>;
  /**
   * Returns, for each x axis, the values that a span of decimal positions across the
   * home view holds. On a relative x axis the values are spans from zero.
   */
  getXBounds: (positions: bounds.Bounds) => Promise<lineplot.AxesBounds>;
  /** Zooms to x, in decimal space, like ContextValue.toggleZoom. */
  toggleZoom: ContextValue["toggleZoom"];
  /** Opens or closes every break of x1, with the same times kept in view. */
  setAllBreaks: (open: boolean) => void;
}

type LineState = LineSpec[];

interface SavedView {
  key: string;
  viewport: box.Box;
  hold: boolean;
}

// canvasX returns the x of a client x, from the left of the canvas the plot draws on.
const canvasX = (clientX: number): number => {
  const origin = Canvas.topLeft();
  if (origin == null) throw new Error("The plot has no canvas");
  return clientX - origin.x;
};

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
        "clearOverScan" | "hold" | "visible" | "gapsCollapsed"
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
  /** Called with the key that toggleZoom zoomed to, or null when the plot zooms out. */
  onZoomedChange?: (key: string | null) => void;
  /**
   * Called with each change of the view and the selection by the viewport, and with
   * each selection that a break change maps through time.
   */
  onViewportChange?: Viewport.UseHandler;
  /** Called when the number of open or of closed breaks of x1 changes. */
  onBreaksChange?: (breaks: lineplot.BreakCounts) => void;
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
  gapsCollapsed = false,
  loadingMessage,
  ref,
  viewportRef,
  onZoomedChange,
  onViewportChange,
  onBreaksChange,
  ...rest
}: FrameProps): ReactElement => {
  const [lines, setLines] = useState<LineState>([]);
  const [viewportElement, setViewportElement] = useState<HTMLElement | null>(null);

  const memoProps = useMemoDeepEqual({ clearOverScan, hold, visible, gapsCollapsed });

  const [{ path }, state, setState, methods] = Aether.use({
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
    onAetherChange: ({ breaks: next }) => {
      const { open, closed } = state.breaks;
      if (next.open !== open || next.closed !== closed) onBreaksChange?.(next);
    },
  });
  const { grid, loading, fetching, breaks, openBreaks } = state;

  // We use a single resize handler for both the container and plotting region because
  // the container is guaranteed to only resize if the plotting region does. This allows
  // us to save a window observer.
  const handleResize = (container: box.Box) => {
    if (visible) setState((prev) => ({ ...prev, container }));
  };

  const regionRef = Canvas.useRegion(handleResize, { debounce });

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
  const [zoomedKey, setZoomedKeyState] = useState<string | null>(null);
  const setZoomedKey = (key: string | null) => {
    setZoomedKeyState(key);
    onZoomedChange?.(key);
  };

  const applyViewport = (args: Viewport.UseEvent) => {
    if (changesView(args)) {
      appliedRef.current = args.box;
      setState((prev) => ({ ...prev, viewport: args.box }));
    }
    viewportHandlers.current.forEach((_, handler) => handler(args));
  };

  const handleError = Status.useErrorHandler();
  const handleAsyncError = Status.useAsyncErrorHandler();
  const breaksRef = useSyncedRef(breaks);
  const openBreaksRef = useSyncedRef(openBreaks);
  // The selection the plot last reported, so a break change can map it.
  const selectionRef = useRef<box.Box>(box.ZERO);

  // applyBreaks applies a break action and maps the zoom history and the selection
  // through time, and the view too when moveView is set. It resolves to false when the
  // action changes nothing.
  const applyBreaks = async (
    action: lineplot.BreakAction,
    moveView: boolean,
  ): Promise<boolean> => {
    const history = zoomHistory.current;
    const selection = selectionRef.current;
    const change = await methods.getBreakChange({
      action,
      viewport: appliedRef.current,
      held: history.map((v) => v.viewport),
      selection,
    });
    if (change == null) return false;
    setState((prev) => ({ ...prev, openBreaks: change.openBreaks }));
    if (zoomHistory.current === history)
      zoomHistory.current = history.map((v, i) => ({ ...v, viewport: change.held[i] }));
    // A selection made during the await holds its own times.
    if (selectionRef.current === selection && !box.areaIsZero(selection)) {
      selectionRef.current = change.selection;
      onViewportChange?.({
        box: change.selection,
        mode: "select",
        cursor: xy.ZERO,
        stage: "end",
      });
    }
    if (moveView) {
      viewportRef?.current?.sync(change.viewport, selectionRef.current);
      applyViewport({
        box: change.viewport,
        mode: "zoom",
        cursor: xy.ZERO,
        stage: "end",
      });
    }
    return true;
  };

  const breaksShown = () => breaksRef.current.open + breaksRef.current.closed > 0;

  const toggleBreak: ContextValue["toggleBreak"] = (clientX, source) => {
    if (!breaksShown()) return;
    handleError(async () => {
      await applyBreaks({ variant: "toggle", x: canvasX(clientX), source }, true);
    }, "Failed to open or close a break");
  };

  const closeBreaks = () => {
    if (openBreaksRef.current.length === 0) return;
    handleError(async () => {
      await applyBreaks({ variant: "all", open: false }, false);
    }, "Failed to close breaks");
  };

  const setAllBreaks = (open: boolean) =>
    handleError(
      async () => {
        await applyBreaks({ variant: "all", open }, true);
      },
      `Failed to ${open ? "open" : "close"} breaks`,
    );

  // selectBound reports a failed read of the seams and leaves the drag unbounded.
  const selectBound = async (x: number): Promise<bounds.Bounds> => {
    if (breaksRef.current.closed === 0) return bounds.INFINITE;
    let seams: number[] = [];
    await handleAsyncError(async () => {
      seams = await methods.getSeams();
    }, "Failed to find the breaks of the plot");
    return {
      lower: seams.findLast((s) => s <= x) ?? -Infinity,
      upper: seams.find((s) => s > x) ?? Infinity,
    };
  };

  const dispatchViewport: Viewport.UseHandler = (args) => {
    if (changesView(args) && zoomHistory.current.length > 0) {
      zoomHistory.current = [];
      setZoomedKey(null);
    }
    if (args.mode === "select" && args.stage === "end") selectionRef.current = args.box;
    else if (changesView(args)) selectionRef.current = box.ZERO;
    if (args.mode === "zoomReset") closeBreaks();
    applyViewport(args);
    onViewportChange?.(args);
  };

  // A click on a break opens it and goes no further.
  const setViewport: Viewport.UseHandler = (args) => {
    if (args.mode !== "click" || !breaksShown()) return dispatchViewport(args);
    handleError(async () => {
      const x = canvasX(args.cursor.x);
      const opened = await applyBreaks({ variant: "toggle", x, source: "plot" }, true);
      if (!opened) dispatchViewport(args);
    }, "Failed to open a break");
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
    setZoomedKey(history.at(-1)?.key ?? null);
    const next =
      back?.viewport ??
      box.reRoot(
        box.construct({ x: x.lower, y: 0 }, { x: x.upper, y: 1 }),
        box.DECIMAL.root,
      );
    viewportRef?.current?.sync(next, selectionRef.current);
    applyViewport({ box: next, mode: "zoom", cursor: xy.ZERO, stage: "start" });
    setHold(back?.hold ?? true);
  };
  const toggleZoomRef = useSyncedRef(toggleZoom);
  const setAllBreaksRef = useSyncedRef(setAllBreaks);
  useImperativeHandle(
    ref,
    () => ({
      getBounds: methods.getBounds,
      getXBounds: methods.getXBounds,
      toggleZoom: (key, x) => toggleZoomRef.current(key, x),
      setAllBreaks: (open) => setAllBreaksRef.current(open),
    }),
    [methods.getBounds, methods.getXBounds],
  );

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
    zoomedKey,
    toggleBreak,
    closeBreaks,
    selectBound,
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
