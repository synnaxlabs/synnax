// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type scale, xy } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { type z } from "zod";

import { type Cell, Table, type tableStateZ } from "@/table/aether/Table";
import { type MountChild, renderAether } from "@/testutil/renderAether";
import { type render } from "@/vis/render";
import { canvasTest } from "@/vis/render/test";
import { value } from "@/vis/value/aether";

const REGION = box.construct({ x: 100, y: 50 }, { width: 400, height: 300 });

const cell = (x: number, y: number, width = 60, height = 30): box.Box =>
  box.construct({ x, y }, { width, height });

// The request a cleanup receives. The table ignores it.
const CLEANUP_REQUEST: render.Request = {
  key: "table",
  priority: "high",
  canvases: [],
  render: () => {},
};

// clip makes each drawn cell scissor its own box, which is how drawn() sees it.
const cellChild = (b: box.Box): MountChild => ({
  type: value.Value.TYPE,
  state: { box: b, clip: true },
});

const mount = (
  cells: Record<string, box.Box>,
  state: Partial<z.input<typeof tableStateZ>> = {},
) => {
  const recorder = canvasTest.record();
  const h = renderAether(Table, {
    state: { region: REGION, ...state },
    render: recorder,
    registry: value.REGISTRY,
    children: Object.fromEntries(
      Object.entries(cells).map(([key, b]) => [key, cellChild(b)]),
    ),
  });
  const draw = (): void => {
    recorder.clear();
    h.component.render();
  };
  const drawn = (): box.Box[] =>
    recorder.upper2d.calls
      .filter((c) => c.op === "scissor")
      .map((c) => c.args[0] as box.Box);
  const origin = (): xy.XY => {
    const applied = recorder.upper2d.calls.find((c) => c.op === "applyScale");
    if (applied == null) throw new Error("no cell drew");
    return (applied.args[0] as scale.XY).pos(xy.ZERO);
  };
  return { h, recorder, draw, drawn, origin };
};

describe("aether Table", () => {
  describe("origin", () => {
    it("draws cells relative to the top left of the region", () => {
      const { draw, origin } = mount({ a: cell(0, 0) });
      draw();
      expect(origin()).toEqual(box.topLeft(REGION));
    });

    it("shifts the origin back by the scroll offset", () => {
      const { draw, origin } = mount(
        { a: cell(40, 120) },
        { scroll: { x: 40, y: 120 } },
      );
      draw();
      expect(origin()).toEqual({ x: 100 - 40, y: 50 - 120 });
    });

    it("moves the origin when the scroll changes", () => {
      const { h, draw, origin } = mount({ a: cell(0, 0) });
      h.setState((p) => ({ ...p, scroll: { x: 10, y: 20 } }));
      draw();
      expect(origin()).toEqual({ x: 90, y: 30 });
    });
  });

  describe("clipping", () => {
    it("clips to the region", () => {
      const { recorder, draw } = mount({ a: cell(0, 0) });
      draw();
      expect(recorder.scissorCalls.at(-1)?.region).toEqual(REGION);
    });

    it("keeps the clip on the region while scrolled", () => {
      const { recorder, draw } = mount(
        { a: cell(40, 120) },
        { scroll: { x: 40, y: 120 } },
      );
      draw();
      expect(recorder.scissorCalls.at(-1)?.region).toEqual(REGION);
    });

    it("clips to both 2d canvases", () => {
      const { recorder, draw } = mount({ a: cell(0, 0) });
      draw();
      expect(recorder.scissorCalls.at(-1)?.canvases).toEqual(["upper2d", "lower2d"]);
    });

    it("widens the clip by the clear overscan", () => {
      const { recorder, draw } = mount({ a: cell(0, 0) }, { clearOverScan: 5 });
      draw();
      expect(recorder.scissorCalls.at(-1)?.overScan).toEqual({ x: 5, y: 5 });
    });
  });

  describe("culling", () => {
    it("draws every cell inside the region", () => {
      const boxes = { a: cell(0, 0), b: cell(60, 0), c: cell(0, 30) };
      const { draw, drawn } = mount(boxes);
      draw();
      expect(drawn()).toEqual([boxes.a, boxes.b, boxes.c]);
    });

    it("skips a cell past the bottom of the region", () => {
      const boxes = { a: cell(0, 0), b: cell(0, 400) };
      const { draw, drawn } = mount(boxes);
      draw();
      expect(drawn()).toEqual([boxes.a]);
    });

    it("skips a cell past the right of the region", () => {
      const boxes = { a: cell(0, 0), b: cell(500, 0) };
      const { draw, drawn } = mount(boxes);
      draw();
      expect(drawn()).toEqual([boxes.a]);
    });

    it("draws a cell that only partly overlaps the region", () => {
      const boxes = { a: cell(0, 290) };
      const { draw, drawn } = mount(boxes);
      draw();
      expect(drawn()).toEqual([boxes.a]);
    });

    it("skips a cell that only touches the edge of the region", () => {
      const boxes = { a: cell(0, 300) };
      const { draw, drawn } = mount(boxes);
      draw();
      expect(drawn()).toEqual([]);
    });

    it("draws a cell the scroll brings into view", () => {
      const boxes = { a: cell(0, 400) };
      const { h, draw, drawn } = mount(boxes);
      draw();
      expect(drawn()).toEqual([]);
      h.setState((p) => ({ ...p, scroll: { x: 0, y: 200 } }));
      draw();
      expect(drawn()).toEqual([boxes.a]);
    });

    it("skips a cell the scroll takes out of view", () => {
      const boxes = { a: cell(0, 0), b: cell(0, 250) };
      const { h, draw, drawn } = mount(boxes);
      draw();
      expect(drawn()).toEqual([boxes.a, boxes.b]);
      h.setState((p) => ({ ...p, scroll: { x: 0, y: 200 } }));
      draw();
      expect(drawn()).toEqual([boxes.b]);
    });
  });

  describe("visibility", () => {
    it("draws no cells while hidden", () => {
      const { draw, drawn } = mount({ a: cell(0, 0) }, { visible: false });
      draw();
      expect(drawn()).toEqual([]);
    });

    it("erases the region while hidden", () => {
      const { recorder, h } = mount({ a: cell(0, 0) }, { visible: false });
      recorder.clear();
      h.component.render()?.(CLEANUP_REQUEST);
      expect(recorder.eraseCalls.at(-1)?.region).toEqual(REGION);
    });

    it("erases the region after a visible draw", () => {
      const { recorder, h } = mount({ a: cell(0, 0) });
      recorder.clear();
      h.component.render()?.(CLEANUP_REQUEST);
      expect(recorder.eraseCalls.at(-1)?.region).toEqual(REGION);
    });
  });

  describe("render requests", () => {
    it("requests a render when the state changes", () => {
      const { recorder, h } = mount({ a: cell(0, 0) });
      recorder.clear();
      h.setState((p) => ({ ...p, scroll: { x: 0, y: 10 } }));
      expect(recorder.loopCalls).toHaveLength(1);
    });

    it("requests one last render on becoming hidden, then stops", () => {
      const { recorder, h } = mount({ a: cell(0, 0) });
      recorder.clear();
      h.setState((p) => ({ ...p, visible: false }));
      expect(recorder.loopCalls).toHaveLength(1);
      recorder.clear();
      h.setState((p) => ({ ...p, scroll: { x: 0, y: 10 } }));
      expect(recorder.loopCalls).toHaveLength(0);
    });
  });

  describe("cell failures", () => {
    it("reports a failing cell instead of throwing", () => {
      const { h, draw } = mount({ a: cell(0, 0) });
      vi.spyOn(h.child<Cell>("a"), "render").mockImplementation(() => {
        throw new Error("cell blew up");
      });
      expect(() => draw()).not.toThrow();
      expect(h.providers.status?.state.statuses.at(-1)?.message).toEqual(
        "Failed to render table",
      );
    });
  });
});
