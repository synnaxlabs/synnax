// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot as lineplotClient } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import {
  bounds,
  box,
  color,
  id,
  location,
  staticWindow,
  TimeRange,
  TimeSpan,
  TimeStamp,
  uuid,
  type Window,
  xy,
} from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
} from "@testing-library/react";
import {
  createRef,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useState,
} from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { LinePlot } from "@/lineplot";
import { lineplot } from "@/lineplot/aether";
import { range } from "@/lineplot/range/aether";
import { tooltip } from "@/lineplot/tooltip/aether";
import { telemTest } from "@/telem/aether/testutil";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import type { Viewport } from "@/viewport";
import { grid } from "@/vis/grid";
import { line } from "@/vis/line/aether";
import { lineTest } from "@/vis/line/aether/testutil";
import { canvasTest } from "@/vis/render/testutil";

const client = createTestClient();

const LINE: lineplotClient.Line = {
  key: "line",
  yAxis: "y1",
  yChannel: 1,
  xChannel: 0,
  color: { type: "fixed", value: color.construct("#00aaff") },
  strokeWidth: 2,
  aggregation: "min_max",
  detail: "medium",
};

// GLHost supplies the line GL program, as the canvas does in production.
const GLHost = ({ children }: PropsWithChildren): ReactElement => {
  const [{ path }] = Aether.use({
    type: lineTest.Host.TYPE,
    schema: lineTest.Host.stateZ,
    initialState: {},
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};

const GL_REGISTRY: aether.ComponentRegistry = {
  ...lineplot.REGISTRY,
  ...range.REGISTRY,
  ...line.REGISTRY,
  [lineTest.Host.TYPE]: lineTest.Host,
};

// createGLRecorder returns a recorder whose GL context records the draws of lines.
const createGLRecorder = (): canvasTest.Recorder => {
  const recorder = canvasTest.record();
  (recorder as { gl: unknown }).gl = canvasTest.createGL().gl;
  return recorder;
};

// all lists c and every component under it.
const all = (c: aether.Component): aether.Component[] => [
  c,
  ...(c instanceof aether.Composite ? c.children.flatMap(all) : []),
];

describe("LinePlot", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: { ...lineplot.REGISTRY, ...range.REGISTRY },
      renderContext: canvasTest.record(),
    });
  });

  const createPlot = async (lines: lineplotClient.Line[] = []): Promise<string> => {
    const project = await client.projects.create({ name: `lineplot_${uuid.create()}` });
    const { key } = await client.lineplots.create({
      project: project.key,
      name: "plot",
    });
    await renderHookSuspended(() => LinePlot.use({ key }), { wrapper });
    if (lines.length === 0) return key;
    const { result } = renderHook(() => LinePlot.useDispatch(), { wrapper });
    await act(async () => {
      await result.current.dispatchAsync({
        key,
        actions: lines.map((line) => lineplotClient.setLine({ line })),
      });
    });
    return key;
  };

  const plot = (
    key: string,
    hold: boolean,
    onViewportChange: LinePlot.LinePlotProps["onViewportChange"],
  ): ReactElement => (
    <LinePlot.Scope.Provider value={key}>
      <LinePlot.LinePlot hold={hold} onViewportChange={onViewportChange} />
    </LinePlot.Scope.Provider>
  );

  const resets = (onViewportChange: ReturnType<typeof vi.fn>): number =>
    onViewportChange.mock.calls.filter(([e]) => e.mode === "zoomReset").length;

  describe("viewport", () => {
    it("should reset the viewport once when the first line is added", async () => {
      const key = await createPlot();
      const onViewportChange = vi.fn();
      render(plot(key, false, onViewportChange), { wrapper });
      const { result } = renderHook(() => LinePlot.useDispatch(), { wrapper });
      await act(async () => {
        await result.current.dispatchAsync({
          key,
          actions: [lineplotClient.setLine({ line: LINE })],
        });
      });
      await waitFor(() => expect(resets(onViewportChange)).toEqual(1));
    });

    it("should reset the viewport when the x axis changes to relative time", async () => {
      const key = await createPlot([LINE]);
      const onViewportChange = vi.fn();
      render(plot(key, false, onViewportChange), { wrapper });
      const { result } = renderHook(() => LinePlot.useDispatch(), { wrapper });
      await act(async () => {
        await result.current.dispatchAsync({
          key,
          actions: [
            lineplotClient.setAxisTimeMode({ key: "x1", timeMode: "relative" }),
          ],
        });
      });
      await waitFor(() => expect(resets(onViewportChange)).toEqual(1));
    });

    it("should reset the viewport once when hold is released", async () => {
      const key = await createPlot([LINE]);
      const onViewportChange = vi.fn();
      const { rerender } = render(plot(key, false, onViewportChange), { wrapper });
      rerender(plot(key, true, onViewportChange));
      rerender(plot(key, false, onViewportChange));
      await waitFor(() => expect(resets(onViewportChange)).toEqual(1));
    });

    it("should return to a zoomed view the plot was not paused in", async () => {
      const key = await createPlot([LINE]);
      const onViewportChange = vi.fn();
      const onViewport = vi.fn();
      const Wrapper = wrapper;
      const controlled = ({ children }: PropsWithChildren): ReactElement => {
        const [hold, setHold] = useState(false);
        return (
          <Wrapper>
            <LinePlot.Scope.Provider value={key}>
              <LinePlot.LinePlot
                hold={hold}
                onHold={setHold}
                onViewportChange={onViewportChange}
              >
                {children}
              </LinePlot.LinePlot>
            </LinePlot.Scope.Provider>
          </Wrapper>
        );
      };
      const { result } = renderHook(
        () => {
          LinePlot.useViewport(onViewport, "test");
          return LinePlot.useContext("test");
        },
        { wrapper: controlled },
      );
      const zoomed = box.construct({ x: 0.1, y: 0.2 }, { x: 0.9, y: 0.8 });
      act(() =>
        result.current.setViewport({
          box: zoomed,
          mode: "zoom",
          stage: "end",
          cursor: xy.ZERO,
        }),
      );
      act(() => result.current.toggleZoom("range", { lower: 0.2, upper: 0.6 }));
      act(() => result.current.toggleZoom("range", { lower: 0.2, upper: 0.6 }));
      expect(onViewport.mock.calls.at(-1)?.[0].box).toEqual(zoomed);
      expect(resets(onViewportChange)).toEqual(0);
    });
  });

  describe("traces", () => {
    const ZERO = TimeStamp.seconds(1000);
    const RED = color.construct("#ff0000");
    const window = staticWindow(new TimeRange(ZERO, ZERO.add(TimeSpan.seconds(10))));
    const RANGES: LinePlot.RangeSpec[] = [
      { key: "r1", window, label: "Range 1", color: RED, zero: ZERO },
      { key: "r2", window, color: "#00ff00", zero: ZERO },
      { key: "r3", window, label: "Rolling" },
    ];

    // Each mount of the wrapper has its own worker tree.
    let roots: aether.Root[] = [];
    beforeEach(async () => {
      roots = [];
      wrapper = await createAsyncSynnaxWrapper({
        client,
        additionalRegistry: GL_REGISTRY,
        renderContext: createGLRecorder(),
        telemFactories: [],
        onRoot: (r) => roots.push(r),
      });
    });

    const components = (): aether.Component[] => roots.flatMap(all);

    // renderTraces renders a plot of RANGES with one stored line for each label, and
    // returns a function that renders the plot again with more props.
    const renderTraces = async (
      props: Partial<LinePlot.LinePlotProps> = {},
      { relative = true, labels = ["pressure"] } = {},
    ): Promise<(next: Partial<LinePlot.LinePlotProps>) => void> => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: "timestamp",
        isIndex: true,
      });
      const lines = await Promise.all(
        labels.map(async (label, i) => {
          const data = await client.channels.create({
            name: id.create(),
            dataType: "float32",
            index: index.key,
          });
          return { ...LINE, key: `${LINE.key}${i || ""}`, yChannel: data.key, label };
        }),
      );
      const key = await createPlot(lines);
      if (relative) {
        const { result } = renderHook(() => LinePlot.useDispatch(), { wrapper });
        await act(async () => {
          await result.current.dispatchAsync({
            key,
            actions: [
              lineplotClient.setAxisTimeMode({ key: "x1", timeMode: "relative" }),
            ],
          });
        });
      }
      const plot = (p: Partial<LinePlot.LinePlotProps>): ReactElement => (
        <GLHost>
          <LinePlot.Scope.Provider value={key}>
            <LinePlot.LinePlot ranges={RANGES} {...p} />
          </LinePlot.Scope.Provider>
        </GLHost>
      );
      const { rerender } = render(plot(props), { wrapper });
      return (next) => rerender(plot({ ...props, ...next }));
    };

    const traces = (): Map<string, line.Line> =>
      new Map(
        components()
          .filter((c): c is line.Line => c instanceof line.Line)
          .map((l) => [l.key, l]),
      );

    const trace = (key: string): line.Line => {
      const found = traces().get(key);
      if (found == null) throw new Error(`no trace ${key}`);
      return found;
    };

    it("should draw each range that has a zero from its zero, in its color", async () => {
      await renderTraces();
      await waitFor(() => expect([...traces().keys()]).toEqual(["line/r1", "line/r2"]));
      expect(trace("line/r1").state.zero).toEqual(ZERO);
      expect(trace("line/r1").state.color).toEqual(RED);
    });

    it("should draw a relative x axis as absolute when relative is disabled", async () => {
      await renderTraces({ relativeDisabled: true });
      await waitFor(() =>
        expect([...traces().keys()]).toEqual(["line", "line/1", "line/2"]),
      );
      expect(trace("line").state.zero).toBeUndefined();
      expect(await screen.findByText("pressure")).toBeDefined();
      expect(screen.queryByText("Range 1")).toBeNull();
    });

    it("should group the traces of a stored line on an absolute axis", async () => {
      await renderTraces({ relativeDisabled: true });
      await waitFor(() => expect(traces().size).toBe(3));
      [...traces().values()].forEach((t) => expect(t.state.group).toEqual("line"));
    });

    it("should name a range that has no label by its key", async () => {
      await renderTraces();
      expect(await screen.findByText("r2")).toBeDefined();
      expect(trace("line/r2").state.tooltipLabel).toEqual("r2");
    });

    it.each([
      ["a hidden range", { hiddenRanges: new Set(["r1"]) }],
      ["a hidden line of a range", { hiddenTraces: new Set(["line/r1"]) }],
    ])("should hide the trace of %s", async (_, props) => {
      await renderTraces(props);
      await waitFor(() => expect(trace("line/r1").state.visible).toBe(false));
      expect(trace("line/r2").state.visible).toBe(true);
    });

    it("should raise a hovered range and dim the others", async () => {
      await renderTraces();
      fireEvent.pointerEnter(await screen.findByText("Range 1"));
      await waitFor(() => expect(trace("line/r1").state.raised).toBe(true));
      expect(trace("line/r1").state.opacity).toEqual(1);
      expect(trace("line/r2").state.opacity).toEqual(0.25);
      expect(trace("line/r2").state.raised).toBe(false);
    });

    it("should raise a hovered line and dim the others on an absolute axis", async () => {
      await renderTraces({}, { relative: false, labels: ["pressure", "flow"] });
      fireEvent.pointerEnter(await screen.findByText("pressure"));
      await waitFor(() => expect(trace("line").state.raised).toBe(true));
      expect(trace("line").state.opacity).toEqual(1);
      expect(trace("line1").state.opacity).toEqual(0.25);
      expect(trace("line1").state.raised).toBe(false);
    });

    it.each([
      {
        name: "a hidden range",
        props: { hiddenRanges: new Set(["r1"]) },
        options: {},
        from: "r2",
        to: "Range 1",
      },
      {
        name: "a range with no zero",
        props: { ranges: [RANGES[0], { ...RANGES[2], color: "#0000ff" }] },
        options: {},
        from: "Range 1",
        to: "Rolling",
      },
      {
        name: "a hidden line on an absolute axis",
        props: { hiddenLines: new Set(["line1"]) },
        options: { relative: false, labels: ["pressure", "flow"] },
        from: "pressure",
        to: "flow",
      },
    ])(
      "should dim no line on the hover of $name",
      async ({ props, options, from, to }) => {
        await renderTraces(props, options);
        fireEvent.pointerEnter(await screen.findByText(from));
        await waitFor(() =>
          expect([...traces().values()].some((t) => t.state.raised)).toBe(true),
        );
        fireEvent.pointerEnter(screen.getByText(to));
        await waitFor(() => {
          for (const t of traces().values()) {
            expect(t.state.opacity).toEqual(1);
            expect(t.state.raised).toBe(false);
          }
        });
      },
    );

    it("should rename the stored line from the entry of one range", async () => {
      await renderTraces();
      const [label] = await screen.findAllByText("pressure");
      fireEvent.dblClick(label);
      label.innerText = "flow";
      fireEvent.keyDown(label, { key: "Enter" });
      await waitFor(() => expect(trace("line/r2").state.label).toEqual("flow"));
    });

    const hover = async (key: string): Promise<void> => {
      const leaf = await waitFor(() => {
        const found = components().find((c) => c instanceof tooltip.Tooltip);
        if (!(found instanceof tooltip.Tooltip)) throw new Error("no tooltip");
        return found;
      });
      leaf.setState((p) => ({ ...p, hovered: { key, x: 1.5e9 } }));
    };

    it("should report the hovered line with the time of its value", async () => {
      const onHoveredChange = vi.fn();
      await renderTraces({ onHoveredChange });
      await hover("line/r1");
      await waitFor(() =>
        expect(onHoveredChange).toHaveBeenLastCalledWith({
          line: "line",
          range: "r1",
          time: ZERO.add(TimeSpan.milliseconds(1500)),
        }),
      );
    });

    it("should report no hovered line once the tooltip turns off", async () => {
      const onHoveredChange = vi.fn();
      const update = await renderTraces({ onHoveredChange });
      await hover("line/r1");
      await waitFor(() =>
        expect(onHoveredChange).toHaveBeenLastCalledWith(
          expect.objectContaining({ range: "r1" }),
        ),
      );
      update({ enableTooltip: false });
      await waitFor(() => expect(onHoveredChange).toHaveBeenLastCalledWith(null));
    });

    it("should report no ranges once the x axis turns relative", async () => {
      const onHasAnnotationsChange = vi.fn();
      const update = await renderTraces({
        relativeDisabled: true,
        rangeProviderProps: { onHasAnnotationsChange },
      });
      // The test plot draws no strip, so the spec sets the count that a draw sets.
      const [provider] = components().filter(
        (c): c is range.Provider => c instanceof range.Provider,
      );
      act(() => provider.setState((s) => ({ ...s, count: 1 })));
      await waitFor(() =>
        expect(onHasAnnotationsChange).toHaveBeenLastCalledWith(true),
      );
      update({ relativeDisabled: false });
      await waitFor(() =>
        expect(onHasAnnotationsChange).toHaveBeenLastCalledWith(false),
      );
    });
  });

  describe("loading message", () => {
    const renderMessage = async (windows: Window[]): Promise<string | undefined> => {
      const key = await createPlot();
      const Wrapper = wrapper;
      const plotted = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <LinePlot.Scope.Provider value={key}>
            <LinePlot.LinePlot
              ranges={windows.map((window, i) => ({ key: String(i), window }))}
            >
              {children}
            </LinePlot.LinePlot>
          </LinePlot.Scope.Provider>
        </Wrapper>
      );
      const { result } = renderHook(() => LinePlot.useFetchContext("test"), {
        wrapper: plotted,
      });
      return result.current.loadingMessage;
    };

    it("should count the time a running range has run", async () => {
      const start = TimeStamp.now().sub(TimeSpan.hours(2));
      const window = staticWindow(new TimeRange(start, TimeStamp.MAX));
      expect(await renderMessage([window])).toEqual("Fetching 2h of data");
    });

    it("should have no message for a range that has not started", async () => {
      const window = staticWindow(new TimeRange(TimeStamp.MAX, TimeStamp.MAX));
      expect(await renderMessage([window])).toBeUndefined();
    });
  });

  describe("toggleZoom", () => {
    const OUTER = { lower: 0.2, upper: 0.6 };
    const INNER = { lower: 0.3, upper: 0.4 };
    // Rooted at the bottom left, like the box.DECIMAL that a zoom reset installs.
    const OUTER_VIEW = {
      one: { x: 0.2, y: 0 },
      two: { x: 0.6, y: 1 },
      root: location.BOTTOM_LEFT,
    };

    const renderFrame = () => {
      const onViewport = vi.fn();
      const onHold = vi.fn();
      const sync = vi.fn();
      const onZoomedChange = vi.fn();
      const viewportRef = { current: { reset: vi.fn(), sync } };
      const ref = createRef<LinePlot.FrameRef>();
      const Wrapper = wrapper;
      const frame = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <LinePlot.Frame
            ref={ref}
            onHold={onHold}
            viewportRef={viewportRef}
            onZoomedChange={onZoomedChange}
          >
            {children}
          </LinePlot.Frame>
        </Wrapper>
      );
      const { result } = renderHook(
        () => {
          LinePlot.useViewport(onViewport, "test");
          return LinePlot.useContext("test");
        },
        { wrapper: frame },
      );
      const view = (): box.Box => onViewport.mock.calls.at(-1)?.[0].box;
      const held = (): boolean => onHold.mock.calls.at(-1)?.[0];
      return { result, view, held, sync, ref, onZoomedChange };
    };

    it("should zoom to the target and pause the plot", () => {
      const { result, view, held } = renderFrame();
      act(() => result.current.toggleZoom("outer", OUTER));
      expect(view()).toEqual(OUTER_VIEW);
      expect(held()).toBe(true);
    });

    it("should return to the previous view on a second call with the same key", () => {
      const { result, view, held } = renderFrame();
      act(() => result.current.toggleZoom("outer", OUTER));
      act(() => result.current.toggleZoom("outer", OUTER));
      expect(view()).toEqual(box.DECIMAL);
      expect(held()).toBe(false);
    });

    it("should move the viewport to each view it sets", () => {
      const { result, sync } = renderFrame();
      act(() => result.current.toggleZoom("outer", OUTER));
      act(() => result.current.toggleZoom("outer", OUTER));
      expect(sync.mock.calls).toEqual([
        [OUTER_VIEW, box.ZERO],
        [box.DECIMAL, box.ZERO],
      ]);
    });

    it("should keep the mask over the selection in each view it sets", () => {
      const { result, sync } = renderFrame();
      const selection = box.construct({ x: 0.3, y: 0 }, { x: 0.4, y: 1 });
      act(() =>
        result.current.setViewport({
          box: selection,
          mode: "select",
          stage: "end",
          cursor: xy.ZERO,
        }),
      );
      act(() => result.current.toggleZoom("outer", OUTER));
      expect(sync).toHaveBeenCalledWith(OUTER_VIEW, selection);
    });

    it("should return one level per call for nested zooms", () => {
      const { result, view, held } = renderFrame();
      act(() => result.current.toggleZoom("outer", OUTER));
      act(() => result.current.toggleZoom("inner", INNER));
      act(() => result.current.toggleZoom("inner", INNER));
      expect(view()).toEqual(OUTER_VIEW);
      expect(held()).toBe(true);
      act(() => result.current.toggleZoom("outer", OUTER));
      expect(view()).toEqual(box.DECIMAL);
      expect(held()).toBe(false);
    });

    it("should report the key of the zoom that the view shows", () => {
      const { result } = renderFrame();
      expect(result.current.zoomedKey).toBeNull();
      act(() => result.current.toggleZoom("outer", OUTER));
      act(() => result.current.toggleZoom("inner", INNER));
      expect(result.current.zoomedKey).toBe("inner");
      act(() => result.current.toggleZoom("inner", INNER));
      expect(result.current.zoomedKey).toBe("outer");
      act(() =>
        result.current.setViewport({
          box: box.DECIMAL,
          mode: "pan",
          stage: "end",
          cursor: xy.ZERO,
        }),
      );
      expect(result.current.zoomedKey).toBeNull();
    });

    it("should zoom from its ref and report each zoomed key", () => {
      const { ref, view, onZoomedChange } = renderFrame();
      act(() => ref.current?.toggleZoom("outer", OUTER));
      expect(view()).toEqual(OUTER_VIEW);
      expect(onZoomedChange).toHaveBeenLastCalledWith("outer");
      act(() => ref.current?.toggleZoom("outer", OUTER));
      expect(onZoomedChange).toHaveBeenLastCalledWith(null);
    });

    it("should zoom in again when the view changes between calls", () => {
      const { result, view } = renderFrame();
      const panned = box.construct({ x: 0.25, y: 0 }, { x: 0.65, y: 1 });
      act(() => result.current.toggleZoom("outer", OUTER));
      act(() =>
        result.current.setViewport({
          box: panned,
          mode: "pan",
          stage: "end",
          cursor: xy.ZERO,
        }),
      );
      act(() => result.current.toggleZoom("outer", OUTER));
      expect(view()).toEqual(OUTER_VIEW);
      act(() => result.current.toggleZoom("outer", OUTER));
      expect(view()).toEqual(panned);
    });
  });

  describe("breaks", () => {
    const MINUTE = Number(TimeSpan.MINUTE.valueOf());
    const DAY = Number(TimeSpan.DAY.valueOf());
    // The gap starts a margin of 5% of the first line after its end.
    const GAP = MINUTE * 1.05;
    const CONTAINER = box.construct(0, 0, 1000, 500);
    // Rooted at the bottom left, like the box.DECIMAL that a zoom reset installs.
    const decimal = (lower: number, upper: number): box.Box => ({
      one: { x: lower, y: 0 },
      two: { x: upper, y: 1 },
      root: location.BOTTOM_LEFT,
    });
    const PANNED = decimal(0.1, 0.9);
    const SELECTION = box.construct({ x: 0.1, y: 0.2 }, { x: 0.2, y: 0.4 });

    const pan = (b: box.Box): Viewport.UseEvent => ({
      box: b,
      mode: "pan",
      stage: "end",
      cursor: xy.ZERO,
    });
    const select = (b: box.Box): Viewport.UseEvent => ({
      box: b,
      mode: "select",
      stage: "end",
      cursor: xy.ZERO,
    });
    const ZOOM_RESET: Viewport.UseEvent = {
      box: box.DECIMAL,
      mode: "zoomReset",
      stage: "start",
      cursor: xy.ZERO,
    };

    // Positions across the home view hold times to well under a microsecond.
    const expectSameTimes = (actual: bounds.Bounds, expected: bounds.Bounds): void => {
      expect(actual.lower).toBeCloseTo(expected.lower, -3);
      expect(actual.upper).toBeCloseTo(expected.upper, -3);
    };

    let roots: aether.Root[] = [];
    let recorder: canvasTest.Recorder;
    // The frame finds the x of a click from the left of the canvas container.
    let canvas: HTMLElement;
    beforeEach(async () => {
      roots = [];
      canvas = document.createElement("div");
      canvas.className = "pluto-canvas-container";
      document.body.appendChild(canvas);
      recorder = createGLRecorder();
      wrapper = await createAsyncSynnaxWrapper({
        client: null,
        additionalRegistry: GL_REGISTRY,
        renderContext: recorder,
        telemFactories: [new telemTest.TestFactory()],
        onRoot: (r) => roots.push(r),
      });
    });
    afterEach(() => {
      canvas.remove();
      vi.restoreAllMocks();
    });

    interface FrameOptions {
      initialOpenBreaks?: number[];
      breaksHidden?: boolean;
      x1?: { label?: string; children?: ReactNode };
      /** Returns more children of the frame, such as a viewport or a second x axis. */
      extra?: () => ReactNode;
    }

    // renderFrame renders a frame whose x1 holds two minutes of data a day apart, with
    // a break between them at the middle of the plot.
    const renderFrame = async ({
      initialOpenBreaks = [],
      breaksHidden = false,
      x1 = {},
      extra = () => null,
    }: FrameOptions = {}) => {
      const getBreakChange = vi.spyOn(lineplot.LinePlot.prototype, "getBreakChange");
      const onViewport = vi.fn();
      const onViewportChange = vi.fn();
      const onBreakCountsChange = vi.fn();
      const onOpenBreaksChange = vi.fn<(change: LinePlot.OpenBreaksChange) => void>();
      const sync = vi.fn<(view: box.Box, selection?: box.Box) => void>();
      const viewportRef = { current: { reset: vi.fn(), sync } };
      const ref = createRef<LinePlot.FrameRef>();
      const lines = [0, DAY].map((start) => ({
        x: telemTest.seriesSpec([start, start + MINUTE]),
        y: telemTest.seriesSpec([0, 1]),
      }));
      const Wrapper = wrapper;
      const frame = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <GLHost>
            <LinePlot.Frame
              ref={ref}
              viewportRef={viewportRef}
              onViewportChange={onViewportChange}
              onBreakCountsChange={onBreakCountsChange}
              breaksHidden={breaksHidden}
              initialOpenBreaks={initialOpenBreaks}
              onOpenBreaksChange={onOpenBreaksChange}
            >
              <LinePlot.XAxis
                axisKey="x1"
                location="bottom"
                type="time"
                label={x1.label}
              >
                <LinePlot.YAxis axisKey="y1" location="left">
                  {lines.map(({ x, y }, i) => (
                    <LinePlot.Line
                      key={i}
                      aetherKey={`line-${i}`}
                      legendGroup="lines"
                      color="#ff0000"
                      x={x}
                      y={y}
                    />
                  ))}
                </LinePlot.YAxis>
                {x1.children}
              </LinePlot.XAxis>
              {extra()}
              {children}
            </LinePlot.Frame>
          </GLHost>
        </Wrapper>
      );
      const { result, rerender } = renderHook(
        () => {
          LinePlot.useViewport(onViewport, "test");
          return LinePlot.useContext("test");
        },
        { wrapper: frame },
      );
      const plot = await waitFor(() => {
        const found = roots
          .flatMap(all)
          .find((c): c is lineplot.LinePlot => c instanceof lineplot.LinePlot);
        if (found == null) throw new Error("no plot");
        return found;
      });
      // jsdom lays out no element, so the plot takes its size from the spec.
      act(() => plot.setState((p) => ({ ...p, container: CONTAINER })));
      act(() => recorder.renderLast(lineplot.LinePlot.TYPE));
      await waitFor(() => expect(onBreakCountsChange).toHaveBeenCalled());
      const view = (): box.Box => onViewport.mock.lastCall?.[0].box;
      // seamX returns the x on the canvas of the seam of x1 in view.
      const seamX = (view: box.Box = box.DECIMAL): number => {
        const region = grid.visualizationBox(plot.state.grid, plot.state.container);
        const [seam] = plot.getSeams();
        if (seam == null) throw new Error("x1 has no seam");
        const { lower, upper } = box.xBounds(view);
        return (
          box.left(region) + ((seam - lower) / (upper - lower)) * box.width(region)
        );
      };
      // times returns the times of x1 that b, a box across the home view, holds.
      const times = async (b: box.Box): Promise<bounds.Bounds> => {
        const xBounds = await ref.current?.getXBounds(box.xBounds(b));
        if (xBounds == null) throw new Error("no frame");
        return xBounds.x1;
      };
      // openAtSeam opens the break by a click on its seam and waits for it. A break
      // change runs after each change before it, so those have settled too.
      const openAtSeam = async (): Promise<void> => {
        act(() => result.current.toggleBreak(seamX(), "axis"));
        await waitFor(() => expect(plot.state.openBreaks).toEqual([GAP]));
      };
      const actions = (): lineplot.BreakAction[] =>
        getBreakChange.mock.calls.map(([req]) => req.action);
      const reported = (): LinePlot.OpenBreaksChange | undefined =>
        onOpenBreaksChange.mock.lastCall?.[0];
      return {
        result,
        rerender,
        ref,
        plot,
        getBreakChange,
        onViewportChange,
        onBreakCountsChange,
        onOpenBreaksChange,
        sync,
        view,
        seamX,
        times,
        openAtSeam,
        actions,
        reported,
      };
    };

    it("should mount with the open breaks it is given", async () => {
      const { plot, onBreakCountsChange } = await renderFrame({
        initialOpenBreaks: [GAP],
      });
      expect(plot.state.openBreaks).toEqual([GAP]);
      expect(onBreakCountsChange).toHaveBeenLastCalledWith({ open: 1, closed: 0 });
    });

    it("should keep its first open breaks through its first view", async () => {
      const { result, actions, reported } = await renderFrame({
        initialOpenBreaks: [GAP],
        extra: () => <LinePlot.Viewport initial={box.DECIMAL} />,
      });
      // A click away from the break changes nothing, and it runs after any change
      // that the first view made.
      act(() => result.current.toggleBreak(120, "axis"));
      await waitFor(() => expect(actions()).toHaveLength(1));
      expect(actions()[0].variant).toEqual("toggle");
      expect(reported()).toBeUndefined();
    });

    it("should open the break under a click at its x on the canvas", async () => {
      const { result, plot, seamX } = await renderFrame();
      act(() => result.current.toggleBreak(seamX(), "axis"));
      await waitFor(() => expect(plot.state.openBreaks).toEqual([GAP]));
    });

    it("should move the view to the times it showed", async () => {
      const { result, plot, sync, view, seamX, times } = await renderFrame();
      act(() => result.current.setViewport(pan(PANNED)));
      const before = await times(PANNED);
      act(() => result.current.toggleBreak(seamX(PANNED), "axis"));
      await waitFor(() => expect(plot.state.openBreaks).toEqual([GAP]));
      expect(sync).toHaveBeenLastCalledWith(view(), box.ZERO);
      expectSameTimes(await times(view()), before);
    });

    it("should return a held zoom to the times it held", async () => {
      const { result, plot, view, seamX, times } = await renderFrame();
      const zoom = { lower: 0.2, upper: 0.8 };
      act(() => result.current.setViewport(pan(PANNED)));
      const before = await times(PANNED);
      act(() => result.current.toggleZoom("outer", zoom));
      act(() => result.current.toggleBreak(seamX(view()), "axis"));
      await waitFor(() => expect(plot.state.openBreaks).toEqual([GAP]));
      act(() => result.current.toggleZoom("outer", zoom));
      expectSameTimes(await times(view()), before);
    });

    it("should report the open breaks with the view and the selection", async () => {
      const { result, view, seamX, times, reported } = await renderFrame();
      act(() => result.current.setViewport(select(SELECTION)));
      const before = await times(SELECTION);
      act(() => result.current.toggleBreak(seamX(), "axis"));
      await waitFor(() => expect(reported()?.openBreaks).toEqual([GAP]));
      const change = reported();
      if (change == null) throw new Error("no change");
      expect(change.viewport).toEqual(view());
      expectSameTimes(await times(change.selection), before);
    });

    it("should report the reset view when a zoom reset closes the breaks", async () => {
      const { result, reported } = await renderFrame({ initialOpenBreaks: [GAP] });
      act(() => result.current.setViewport(pan(decimal(0.1, 0.6))));
      act(() => result.current.setViewport(ZOOM_RESET));
      await waitFor(() => expect(reported()?.openBreaks).toEqual([]));
      expect(reported()).toEqual({
        openBreaks: [],
        viewport: box.DECIMAL,
        selection: box.ZERO,
      });
    });

    it("should mask the selection at the times it held", async () => {
      const { result, plot, sync, view, seamX, times } = await renderFrame();
      act(() => result.current.setViewport(select(SELECTION)));
      const before = await times(SELECTION);
      act(() => result.current.toggleBreak(seamX(), "axis"));
      await waitFor(() => expect(plot.state.openBreaks).toEqual([GAP]));
      const [synced, mask] = sync.mock.lastCall ?? [];
      if (synced == null || mask == null) throw new Error("no sync");
      expect(synced).toEqual(view());
      expectSameTimes(await times(mask), before);
    });

    it("should map no selection after the view moves", async () => {
      const { result, seamX, reported } = await renderFrame();
      act(() => result.current.setViewport(select(SELECTION)));
      act(() => result.current.setViewport(pan(PANNED)));
      act(() => result.current.toggleBreak(seamX(PANNED), "axis"));
      await waitFor(() => expect(reported()?.openBreaks).toEqual([GAP]));
      expect(reported()?.selection).toEqual(box.ZERO);
    });

    it("should ask for no break change while the breaks are hidden", async () => {
      const getSeams = vi.spyOn(lineplot.LinePlot.prototype, "getSeams");
      const { result, ref, actions } = await renderFrame({ breaksHidden: true });
      act(() => result.current.toggleBreak(500, "plot"));
      act(() =>
        result.current.setViewport({
          box: box.DECIMAL,
          mode: "click",
          stage: "end",
          cursor: { x: 500, y: 40 },
        }),
      );
      expect(await result.current.selectBound(0.45)).toEqual(bounds.INFINITE);
      expect(getSeams).not.toHaveBeenCalled();
      // The ref opens breaks while they are hidden, after any change before it.
      act(() => ref.current?.setAllBreaks(true));
      await waitFor(() => expect(actions()).toHaveLength(1));
      expect(actions()).toEqual([{ variant: "all", open: true }]);
    });

    it("should map no selection after a click dismisses it", async () => {
      const { result, seamX, reported } = await renderFrame();
      act(() => result.current.setViewport(select(SELECTION)));
      act(() =>
        result.current.setViewport({
          box: box.DECIMAL,
          mode: "click",
          stage: "end",
          cursor: { x: 120, y: 40 },
        }),
      );
      act(() => result.current.toggleBreak(seamX(), "axis"));
      await waitFor(() => expect(reported()?.openBreaks).toEqual([GAP]));
      expect(reported()?.selection).toEqual(box.ZERO);
    });

    it("should keep a selection made while a break change loads", async () => {
      const { result, plot, getBreakChange, seamX, reported } = await renderFrame();
      act(() => result.current.setViewport(select(SELECTION)));
      const later = decimal(0.7, 0.8);
      getBreakChange.mockImplementationOnce((req) => {
        result.current.setViewport(select(later));
        return plot.getBreakChange(req);
      });
      await act(async () => {
        result.current.toggleBreak(seamX(), "axis");
        await expect.poll(() => reported()?.openBreaks).toEqual([GAP]);
      });
      expect(reported()?.selection).toEqual(later);
    });

    it("should map again a view set while a break change loads", async () => {
      const { result, plot, getBreakChange, view, seamX, times } = await renderFrame();
      const before = await times(PANNED);
      getBreakChange.mockImplementationOnce((req) => {
        result.current.setViewport(pan(PANNED));
        return plot.getBreakChange(req);
      });
      await act(async () => {
        result.current.toggleBreak(seamX(), "axis");
        await expect.poll(() => plot.state.openBreaks).toEqual([GAP]);
      });
      expectSameTimes(await times(view()), before);
    });

    it("should start each break change from the open breaks of the last", async () => {
      const { ref, onOpenBreaksChange, reported } = await renderFrame();
      act(() => {
        ref.current?.setAllBreaks(true);
        ref.current?.setAllBreaks(false);
      });
      await waitFor(() => expect(onOpenBreaksChange).toHaveBeenCalledTimes(2));
      expect(reported()?.openBreaks).toEqual([]);
    });

    it("should toggle the break under a click on the x axis", async () => {
      const { plot, seamX } = await renderFrame();
      const axis = document.querySelector(".pluto-x-axis");
      expect(axis).not.toBeNull();
      fireEvent.click(axis!, { clientX: seamX() });
      await waitFor(() => expect(plot.state.openBreaks).toEqual([GAP]));
    });

    it("should not toggle a break on a click on the label of the x axis", async () => {
      const { seamX, openAtSeam, actions } = await renderFrame({
        x1: { label: "Time" },
      });
      const axis = document.querySelector(".pluto-x-axis");
      expect(axis).not.toBeNull();
      vi.spyOn(axis!, "getBoundingClientRect").mockReturnValue(
        DOMRect.fromRect({ x: 0, y: 0, width: 400, height: 60 }),
      );
      fireEvent.click(axis!, { clientX: seamX(), clientY: 55 });
      await openAtSeam();
      expect(actions()).toHaveLength(1);
    });

    it("should toggle the break under a click on the range strip", async () => {
      const { plot, seamX } = await renderFrame({
        x1: { children: <LinePlot.Range.Provider /> },
      });
      const strip = document.querySelector(".pluto-lineplot-range-provider");
      expect(strip).not.toBeNull();
      fireEvent.click(strip!, { clientX: seamX() });
      await waitFor(() => expect(plot.state.openBreaks).toEqual([GAP]));
    });

    it("should not toggle a break on a strip with breaks disabled", async () => {
      const { seamX, openAtSeam, actions } = await renderFrame({
        extra: () => (
          <LinePlot.XAxis axisKey="x2" location="top" type="time">
            <LinePlot.Range.Provider breaksDisabled />
          </LinePlot.XAxis>
        ),
      });
      const strip = document.querySelector(".pluto-lineplot-range-provider");
      expect(strip).not.toBeNull();
      fireEvent.click(strip!, { clientX: seamX() });
      await openAtSeam();
      expect(actions()).toHaveLength(1);
    });

    it("should not toggle a break on a click on the x2 axis", async () => {
      const { seamX, openAtSeam, actions } = await renderFrame({
        extra: () => <LinePlot.XAxis axisKey="x2" location="top" type="time" />,
      });
      const axis = document.querySelector(".pluto-x-axis.lyra--location-top");
      expect(axis).not.toBeNull();
      fireEvent.click(axis!, { clientX: seamX() });
      await openAtSeam();
      expect(actions()).toHaveLength(1);
    });

    it("should open every break from its ref", async () => {
      const { ref, reported } = await renderFrame();
      act(() => ref.current?.setAllBreaks(true));
      await waitFor(() => expect(reported()?.openBreaks).toEqual([GAP]));
    });

    it("should close every break when the home view changes", async () => {
      const home = { initial: box.DECIMAL };
      const { rerender, reported } = await renderFrame({
        initialOpenBreaks: [GAP],
        extra: () => <LinePlot.Viewport initial={home.initial} />,
      });
      home.initial = decimal(0.2, 0.8);
      rerender();
      await waitFor(() => expect(reported()?.openBreaks).toEqual([]));
    });

    it("should pass on a click that opens no break", async () => {
      const { result, onViewportChange } = await renderFrame();
      const click: Viewport.UseEvent = {
        box: box.DECIMAL,
        mode: "click",
        stage: "end",
        cursor: { x: 120, y: 40 },
      };
      act(() => result.current.setViewport(click));
      await waitFor(() => expect(onViewportChange).toHaveBeenCalledWith(click));
    });

    it("should close every break when the zoom resets", async () => {
      const { result, reported } = await renderFrame({ initialOpenBreaks: [GAP] });
      act(() => result.current.setViewport(ZOOM_RESET));
      await waitFor(() => expect(reported()?.openBreaks).toEqual([]));
    });

    it("should close the breaks that a change in flight opens", async () => {
      const { result, plot, getBreakChange, onOpenBreaksChange, seamX, reported } =
        await renderFrame();
      getBreakChange.mockImplementationOnce((req) => {
        result.current.setViewport(ZOOM_RESET);
        return plot.getBreakChange(req);
      });
      await act(async () => {
        result.current.toggleBreak(seamX(), "axis");
        await expect.poll(() => onOpenBreaksChange.mock.calls.length).toEqual(2);
      });
      expect(reported()?.openBreaks).toEqual([]);
    });

    describe("select bound", () => {
      it("should bound a select drag by the seams around its start", async () => {
        const { result, plot } = await renderFrame();
        const [seam] = plot.getSeams();
        const bound = result.current.selectBound;
        expect(await bound(seam - 0.1)).toEqual({ lower: -Infinity, upper: seam });
        expect(await bound(seam)).toEqual({ lower: seam, upper: Infinity });
        expect(await bound(seam + 0.1)).toEqual({ lower: seam, upper: Infinity });
      });

      it("should not bound a select drag when no break is closed", async () => {
        const getSeams = vi.spyOn(lineplot.LinePlot.prototype, "getSeams");
        const { result } = await renderFrame({ initialOpenBreaks: [GAP] });
        expect(await result.current.selectBound(0.45)).toEqual(bounds.INFINITE);
        expect(getSeams).not.toHaveBeenCalled();
      });

      it("should not bound a select drag when the seams fail to load", async () => {
        vi.spyOn(lineplot.LinePlot.prototype, "getSeams").mockImplementation(() => {
          throw new Error("no seams");
        });
        const { result } = await renderFrame();
        expect(await result.current.selectBound(0.45)).toEqual(bounds.INFINITE);
      });
    });
  });
});

describe("traceKey", () => {
  const RANGES: LinePlot.RangeSpec[] = [
    { key: "r1", window: staticWindow(new TimeRange(0, 1)) },
  ];

  it("should read back the line and range of a trace key", () => {
    const key = LinePlot.traceKey({ line: "a/b", range: "r1" });
    expect(LinePlot.parseTraceKey(key, RANGES)).toEqual({ line: "a/b", range: "r1" });
  });

  it("should return null for a key that names no range of the plot", () => {
    expect(LinePlot.parseTraceKey("a/r2", RANGES)).toBeNull();
  });
});
