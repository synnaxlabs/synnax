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
  useState,
} from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { LinePlot } from "@/lineplot";
import { lineplot } from "@/lineplot/aether";
import { range } from "@/lineplot/range/aether";
import { tooltip } from "@/lineplot/tooltip/aether";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
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

  describe("relative", () => {
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

    const renderRelative = async (
      props: Partial<LinePlot.LinePlotProps> = {},
    ): Promise<void> => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: "timestamp",
        isIndex: true,
      });
      const data = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        index: index.key,
      });
      const key = await createPlot([
        { ...LINE, yChannel: data.key, label: "pressure" },
      ]);
      const { result } = renderHook(() => LinePlot.useDispatch(), { wrapper });
      await act(async () => {
        await result.current.dispatchAsync({
          key,
          actions: [
            lineplotClient.setAxisTimeMode({ key: "x1", timeMode: "relative" }),
          ],
        });
      });
      render(
        <GLHost>
          <LinePlot.Scope.Provider value={key}>
            <LinePlot.LinePlot ranges={RANGES} {...props} />
          </LinePlot.Scope.Provider>
        </GLHost>,
        { wrapper },
      );
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
      await renderRelative();
      await waitFor(() => expect([...traces().keys()]).toEqual(["line/r1", "line/r2"]));
      expect(trace("line/r1").state.zero).toEqual(ZERO);
      expect(trace("line/r1").state.color).toEqual(RED);
    });

    it("should name a range that has no label by its key", async () => {
      await renderRelative();
      expect(await screen.findByText("r2")).toBeDefined();
      expect(trace("line/r2").state.tooltipLabel).toEqual("r2");
    });

    it.each([
      ["a hidden range", { hiddenRanges: new Set(["r1"]) }],
      ["a hidden line of a range", { hiddenTraces: new Set(["line/r1"]) }],
    ])("should hide the trace of %s", async (_, props) => {
      await renderRelative(props);
      await waitFor(() => expect(trace("line/r1").state.visible).toBe(false));
      expect(trace("line/r2").state.visible).toBe(true);
    });

    it("should raise a hovered range and dim the others", async () => {
      await renderRelative();
      fireEvent.pointerEnter(await screen.findByText("Range 1"));
      await waitFor(() => expect(trace("line/r1").state.raised).toBe(true));
      expect(trace("line/r1").state.opacity).toEqual(1);
      expect(trace("line/r2").state.opacity).toEqual(0.25);
      expect(trace("line/r2").state.raised).toBe(false);
    });

    it("should rename the stored line from the entry of one range", async () => {
      await renderRelative();
      const [label] = await screen.findAllByText("pressure");
      fireEvent.dblClick(label);
      label.innerText = "flow";
      fireEvent.keyDown(label, { key: "Enter" });
      await waitFor(() => expect(trace("line/r2").state.label).toEqual("flow"));
    });

    it("should report the hovered line with the time of its value", async () => {
      const onHoveredChange = vi.fn();
      await renderRelative({ onHoveredChange });
      const leaf = await waitFor(() => {
        const found = components().find((c) => c instanceof tooltip.Tooltip);
        if (!(found instanceof tooltip.Tooltip)) throw new Error("no tooltip");
        return found;
      });
      leaf.setState((p) => ({ ...p, hovered: { key: "line/r1", x: 1.5e9 } }));
      await waitFor(() =>
        expect(onHoveredChange).toHaveBeenLastCalledWith({
          line: "line",
          range: "r1",
          time: ZERO.add(TimeSpan.milliseconds(1500)),
        }),
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
      expect(sync.mock.calls).toEqual([[OUTER_VIEW], [box.DECIMAL]]);
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
