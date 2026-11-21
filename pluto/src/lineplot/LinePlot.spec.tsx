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
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import type { Viewport } from "@/viewport";
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
      const recorder = canvasTest.record();
      (recorder as { gl: unknown }).gl = canvasTest.createGL().gl;
      wrapper = await createAsyncSynnaxWrapper({
        client,
        additionalRegistry: {
          ...lineplot.REGISTRY,
          ...range.REGISTRY,
          ...line.REGISTRY,
          [lineTest.Host.TYPE]: lineTest.Host,
        },
        renderContext: recorder,
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
    // Rooted at the bottom left, like the box.DECIMAL that a zoom reset installs.
    const decimal = (lower: number, upper: number): box.Box => ({
      one: { x: lower, y: 0 },
      two: { x: upper, y: 1 },
      root: location.BOTTOM_LEFT,
    });
    const SELECTION = box.construct({ x: 0.1, y: 0.2 }, { x: 0.2, y: 0.4 });
    const MOVED = box.construct({ x: 0.3, y: 0.2 }, { x: 0.4, y: 0.4 });
    const CHANGE: lineplot.BreaksChange = {
      openBreaks: [5],
      viewport: decimal(0, 0.5),
      held: [decimal(0.5, 1)],
      selection: MOVED,
    };
    const COUNTS = { open: 0, closed: 1 };

    let roots: aether.Root[] = [];
    // The frame finds the x of a click from the left of the canvas container.
    let canvas: HTMLElement;
    beforeEach(async () => {
      roots = [];
      canvas = document.createElement("div");
      canvas.className = "pluto-canvas-container";
      document.body.appendChild(canvas);
      wrapper = await createAsyncSynnaxWrapper({
        client: null,
        additionalRegistry: { ...lineplot.REGISTRY, ...range.REGISTRY },
        renderContext: canvasTest.record(),
        onRoot: (r) => roots.push(r),
      });
    });
    afterEach(() => {
      canvas.remove();
      vi.restoreAllMocks();
    });

    const select = (b: box.Box): Viewport.UseEvent => ({
      box: b,
      mode: "select",
      stage: "end",
      cursor: xy.ZERO,
    });

    // renderFrame renders a frame, with the elements that extra returns, whose plot
    // has a closed break and an open one, and returns change for each break action.
    const renderFrame = async (
      change: lineplot.BreaksChange | null = CHANGE,
      extra: () => ReactNode = () => null,
    ) => {
      const getBreakChange = vi
        .spyOn(lineplot.LinePlot.prototype, "getBreakChange")
        .mockReturnValue(change);
      const onViewport = vi.fn();
      const onViewportChange = vi.fn();
      const onBreaksChange = vi.fn();
      const onOpenBreaksChange = vi.fn();
      const sync = vi.fn();
      const viewportRef = { current: { reset: vi.fn(), sync } };
      const ref = createRef<LinePlot.FrameRef>();
      const Wrapper = wrapper;
      const frame = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <LinePlot.Frame
            ref={ref}
            viewportRef={viewportRef}
            onViewportChange={onViewportChange}
            onBreaksChange={onBreaksChange}
            initialOpenBreaks={[2]}
            onOpenBreaksChange={onOpenBreaksChange}
          >
            {extra()}
            {children}
          </LinePlot.Frame>
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
      act(() => plot.setState((p) => ({ ...p, breaks: COUNTS })));
      await waitFor(() => expect(onBreaksChange).toHaveBeenCalledWith(COUNTS));
      const view = (): box.Box => onViewport.mock.calls.at(-1)?.[0].box;
      const selected = (): box.Box | undefined =>
        onOpenBreaksChange.mock.calls.at(-1)?.[0].selection;
      return {
        result,
        rerender,
        ref,
        plot,
        getBreakChange,
        onViewportChange,
        onBreaksChange,
        onOpenBreaksChange,
        sync,
        view,
        selected,
      };
    };

    it("should report the break counts of the plot", async () => {
      await renderFrame();
    });

    it("should mount with the open breaks it is given", async () => {
      const { plot } = await renderFrame();
      expect(plot.state.openBreaks).toEqual([2]);
    });

    it("should keep the open breaks it mounts with through its first view", async () => {
      const { plot, getBreakChange } = await renderFrame(CHANGE, () => (
        <LinePlot.Viewport initial={box.DECIMAL} />
      ));
      expect(getBreakChange).not.toHaveBeenCalled();
      expect(plot.state.openBreaks).toEqual([2]);
    });

    it("should ask for the change of a click at its x on the canvas", async () => {
      const { result, getBreakChange } = await renderFrame();
      act(() => result.current.toggleBreak(120, "axis"));
      await waitFor(() => expect(getBreakChange).toHaveBeenCalled());
      expect(getBreakChange.mock.calls[0][0]).toEqual({
        action: { variant: "toggle", x: 120, source: "axis" },
        viewport: box.DECIMAL,
        held: [],
        selection: box.ZERO,
      });
    });

    it("should move the view to the times it showed", async () => {
      const { result, sync, view } = await renderFrame();
      act(() => result.current.toggleBreak(120, "axis"));
      await waitFor(() => expect(sync).toHaveBeenCalledWith(CHANGE.viewport, box.ZERO));
      expect(view()).toEqual(CHANGE.viewport);
    });

    it("should return a held zoom to the times it held", async () => {
      const { result, view } = await renderFrame();
      act(() => result.current.toggleZoom("outer", { lower: 0.2, upper: 0.6 }));
      act(() => result.current.toggleBreak(120, "axis"));
      await waitFor(() => expect(view()).toEqual(CHANGE.viewport));
      act(() => result.current.toggleZoom("outer", { lower: 0.2, upper: 0.6 }));
      expect(view()).toEqual(CHANGE.held[0]);
    });

    it("should report the open breaks with the view and the selection", async () => {
      const { result, onOpenBreaksChange } = await renderFrame();
      act(() => result.current.setViewport(select(SELECTION)));
      act(() => result.current.toggleBreak(120, "axis"));
      await waitFor(() =>
        expect(onOpenBreaksChange).toHaveBeenCalledWith({
          openBreaks: CHANGE.openBreaks,
          viewport: CHANGE.viewport,
          selection: MOVED,
        }),
      );
    });

    it("should report the reset view when a zoom reset closes the breaks", async () => {
      const { result, onOpenBreaksChange } = await renderFrame();
      act(() =>
        result.current.setViewport({
          box: decimal(0.1, 0.6),
          mode: "pan",
          stage: "end",
          cursor: xy.ZERO,
        }),
      );
      act(() =>
        result.current.setViewport({
          box: box.DECIMAL,
          mode: "zoomReset",
          stage: "start",
          cursor: xy.ZERO,
        }),
      );
      await waitFor(() => expect(onOpenBreaksChange).toHaveBeenCalled());
      expect(onOpenBreaksChange.mock.calls[0][0]).toEqual({
        openBreaks: CHANGE.openBreaks,
        viewport: box.DECIMAL,
        selection: box.ZERO,
      });
    });

    it("should mask the selection at the times it held", async () => {
      const { result, sync } = await renderFrame();
      act(() => result.current.setViewport(select(SELECTION)));
      act(() => result.current.toggleBreak(120, "axis"));
      await waitFor(() => expect(sync).toHaveBeenCalledWith(CHANGE.viewport, MOVED));
    });

    it("should map no selection after the view moves", async () => {
      const { result, getBreakChange } = await renderFrame();
      act(() => result.current.setViewport(select(SELECTION)));
      act(() =>
        result.current.setViewport({
          box: decimal(0.1, 0.6),
          mode: "pan",
          stage: "end",
          cursor: xy.ZERO,
        }),
      );
      act(() => result.current.toggleBreak(120, "axis"));
      await waitFor(() => expect(getBreakChange).toHaveBeenCalled());
      expect(getBreakChange.mock.calls[0][0].selection).toEqual(box.ZERO);
    });

    it("should keep a selection made while a break change loads", async () => {
      const { result, getBreakChange, selected } = await renderFrame();
      act(() => result.current.setViewport(select(SELECTION)));
      const later = decimal(0.7, 0.8);
      getBreakChange.mockImplementationOnce(() => {
        result.current.setViewport(select(later));
        return CHANGE;
      });
      await act(async () => {
        result.current.toggleBreak(120, "axis");
        await expect.poll(() => getBreakChange.mock.calls.length).toEqual(1);
      });
      await act(async () => {});
      expect(selected()).toEqual(later);
    });

    it("should map again a view set while a break change loads", async () => {
      const { result, getBreakChange, sync } = await renderFrame();
      const panned = decimal(0.1, 0.6);
      getBreakChange.mockImplementationOnce(() => {
        result.current.setViewport({
          box: panned,
          mode: "pan",
          stage: "end",
          cursor: xy.ZERO,
        });
        return CHANGE;
      });
      await act(async () => {
        result.current.toggleBreak(120, "axis");
        await expect.poll(() => getBreakChange.mock.calls.length).toEqual(2);
      });
      expect(getBreakChange.mock.calls[1][0]).toMatchObject({
        action: { variant: "set", open: CHANGE.openBreaks },
        viewport: panned,
      });
      await waitFor(() => expect(sync).toHaveBeenCalledWith(CHANGE.viewport, box.ZERO));
    });

    it("should start each break change from the open breaks of the last", async () => {
      const { result, getBreakChange, plot } = await renderFrame();
      const seen: number[][] = [];
      getBreakChange.mockImplementation(() => {
        seen.push(plot.state.openBreaks);
        return CHANGE;
      });
      await act(async () => {
        result.current.toggleBreak(120, "axis");
        result.current.toggleBreak(300, "axis");
        await expect.poll(() => seen.length).toEqual(2);
      });
      expect(seen).toEqual([[2], CHANGE.openBreaks]);
    });

    it("should toggle the break under a click on the x axis", async () => {
      const { getBreakChange } = await renderFrame(CHANGE, () => (
        <LinePlot.XAxis axisKey="x1" location="bottom" type="time" />
      ));
      const axis = document.querySelector(".pluto-axis");
      expect(axis).not.toBeNull();
      fireEvent.click(axis!, { clientX: 120 });
      await waitFor(() => expect(getBreakChange).toHaveBeenCalled());
      expect(getBreakChange.mock.calls[0][0].action).toEqual({
        variant: "toggle",
        x: 120,
        source: "axis",
      });
    });

    it("should not toggle a break on a click on the label of the x axis", async () => {
      const { getBreakChange } = await renderFrame(CHANGE, () => (
        <LinePlot.XAxis axisKey="x1" location="bottom" type="time" label="Time" />
      ));
      const axis = document.querySelector(".pluto-axis");
      expect(axis).not.toBeNull();
      vi.spyOn(axis!, "getBoundingClientRect").mockReturnValue(
        DOMRect.fromRect({ x: 0, y: 0, width: 400, height: 60 }),
      );
      fireEvent.click(axis!, { clientX: 120, clientY: 55 });
      await act(async () => {});
      expect(getBreakChange).not.toHaveBeenCalled();
    });

    it("should toggle the break under a click on the range strip", async () => {
      const { getBreakChange } = await renderFrame(CHANGE, () => (
        <LinePlot.XAxis axisKey="x1" location="bottom" type="time">
          <LinePlot.Range.Provider />
        </LinePlot.XAxis>
      ));
      const strip = document.querySelector(".pluto-lineplot-range-provider");
      expect(strip).not.toBeNull();
      fireEvent.click(strip!, { clientX: 120 });
      await waitFor(() => expect(getBreakChange).toHaveBeenCalled());
      expect(getBreakChange.mock.calls[0][0].action).toEqual({
        variant: "toggle",
        x: 120,
        source: "plot",
      });
    });

    it("should not toggle a break on a click on the x2 axis", async () => {
      const { getBreakChange } = await renderFrame(CHANGE, () => (
        <LinePlot.XAxis axisKey="x2" location="top" type="time" />
      ));
      const axis = document.querySelector(".pluto-axis");
      expect(axis).not.toBeNull();
      fireEvent.click(axis!, { clientX: 120 });
      await act(async () => {});
      expect(getBreakChange).not.toHaveBeenCalled();
    });

    it("should open every break from its ref", async () => {
      const { ref, getBreakChange, view } = await renderFrame();
      act(() => ref.current?.setAllBreaks(true));
      await waitFor(() => expect(view()).toEqual(CHANGE.viewport));
      expect(getBreakChange.mock.calls[0][0].action).toEqual({
        variant: "all",
        open: true,
      });
    });

    it("should close every break when the home view changes", async () => {
      const home = { initial: box.DECIMAL };
      const { getBreakChange, rerender } = await renderFrame(CHANGE, () => (
        <LinePlot.Viewport initial={home.initial} />
      ));
      getBreakChange.mockClear();
      home.initial = decimal(0.2, 0.8);
      rerender();
      await waitFor(() =>
        expect(getBreakChange.mock.calls[0]?.[0].action).toEqual({
          variant: "all",
          open: false,
        }),
      );
    });

    it("should pass on a click that opens no break", async () => {
      const { result, onViewportChange } = await renderFrame(null);
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
      const { result, getBreakChange } = await renderFrame();
      act(() =>
        result.current.setViewport({
          box: box.DECIMAL,
          mode: "zoomReset",
          stage: "start",
          cursor: xy.ZERO,
        }),
      );
      await waitFor(() =>
        expect(getBreakChange.mock.calls[0]?.[0].action).toEqual({
          variant: "all",
          open: false,
        }),
      );
    });

    describe("select bound", () => {
      it("should bound a select drag by the seams on each side of its start", async () => {
        vi.spyOn(lineplot.LinePlot.prototype, "getSeams").mockReturnValue([0.3, 0.6]);
        const { result } = await renderFrame();
        const bound = result.current.selectBound;
        expect(await bound(0.45)).toEqual({ lower: 0.3, upper: 0.6 });
        expect(await bound(0.3)).toEqual({ lower: 0.3, upper: 0.6 });
        expect(await bound(0.1)).toEqual({ lower: -Infinity, upper: 0.3 });
        expect(await bound(0.9)).toEqual({ lower: 0.6, upper: Infinity });
      });

      it("should not bound a select drag when no break is closed", async () => {
        const getSeams = vi.spyOn(lineplot.LinePlot.prototype, "getSeams");
        const { result, plot, onBreaksChange } = await renderFrame();
        const counts = { open: 1, closed: 0 };
        act(() => plot.setState((p) => ({ ...p, breaks: counts })));
        await waitFor(() => expect(onBreaksChange).toHaveBeenLastCalledWith(counts));
        expect(await result.current.selectBound(0.45)).toEqual(bounds.INFINITE);
        expect(getSeams).not.toHaveBeenCalled();
      });

      it("should not bound a select drag when the seams fail to load", async () => {
        vi.spyOn(lineplot.LinePlot.prototype, "getSeams").mockImplementation(() => {
          throw new Error("no seams");
        });
        const { result } = await renderFrame();
        expect(await result.current.selectBound(0.45)).toEqual({
          lower: -Infinity,
          upper: Infinity,
        });
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
