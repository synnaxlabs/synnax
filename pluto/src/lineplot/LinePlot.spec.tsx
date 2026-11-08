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
import { box, color, location, uuid, xy } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LinePlot } from "@/lineplot";
import { lineplot } from "@/lineplot/aether";
import { range } from "@/lineplot/range/aether";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/testutil";

const client = createTestClient();

const LINE: lineplotClient.Line = {
  key: "line",
  yAxis: "y1",
  yChannel: 1,
  xChannel: 0,
  color: color.construct("#00aaff"),
  strokeWidth: 2,
  aggregation: "min_max",
  detail: "medium",
};

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
    const { key } = await client.lineplots.create(project.key, { name: "plot" });
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
      const viewportRef = { current: { reset: vi.fn(), sync } };
      const Wrapper = wrapper;
      const frame = ({ children }: PropsWithChildren): ReactElement => (
        <Wrapper>
          <LinePlot.Frame onHold={onHold} viewportRef={viewportRef}>
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
      return { result, view, held, sync };
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
