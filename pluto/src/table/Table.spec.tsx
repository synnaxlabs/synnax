// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { box, type scale, xy } from "@synnaxlabs/x";
import { act, fireEvent, render, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Table } from "@/table";
import { table as aetherTable } from "@/table/aether";
import { INDICATOR_SIZE } from "@/table/Indicator";
import { telemTest } from "@/telem/aether/test";
import { mockBoundingClientRect } from "@/testutil/dom";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/test";
import { Value } from "@/vis/value";
import { value } from "@/vis/value/aether";

const client = createTestClient();

const SURFACE_WIDTH = 1000;
const SURFACE_HEIGHT = 600;
const COL_SIZE = 72;
const ROW_SIZE = 36;

const expectedOffset = (contentWidth: number, contentHeight: number) => ({
  x: Math.floor((SURFACE_WIDTH - contentWidth - INDICATOR_SIZE) / 2),
  y: Math.floor((SURFACE_HEIGHT - contentHeight - INDICATOR_SIZE) / 2),
});

const RECT = mockBoundingClientRect(0, 0, SURFACE_WIDTH, SURFACE_HEIGHT)();

// Single-hook bootstrap so the suspending useEnsure is not followed by other
// hooks, which trips a React 19 concurrent replay warning (same pattern as
// table queries.spec.tsx).
const loadTable = async (
  wrapper: React.FC<PropsWithChildren>,
  key: table.Key,
): Promise<void> => {
  const Bootstrap = (): ReactElement => {
    Table.useEnsure({ key });
    return <div data-testid="loaded" />;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Errors.SuspenseBoundary loading={null}>
        <Bootstrap />
      </Errors.SuspenseBoundary>,
      { wrapper },
    );
  });
  await utils.findByTestId("loaded");
};

// The @juggle polyfill reads computed styles, which jsdom leaves empty, so it
// never reports a size. useResize measures with getBoundingClientRect and
// ignores the entries, so notifying is enough.
class ImmediateResizeObserver {
  constructor(private readonly notify: () => void) {}
  observe(): void {
    this.notify();
  }
  unobserve(): void {}
  disconnect(): void {}
}

describe("Table", () => {
  let wrapper: React.FC<PropsWithChildren>;
  let key: table.Key;
  let recorder: canvasTest.Recorder;

  beforeEach(async () => {
    vi.stubGlobal("ResizeObserver", ImmediateResizeObserver);
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(RECT);
    // Canvas.useRegion measures against the lower2d canvas and bails without it.
    const canvas = document.createElement("div");
    canvas.className = "pluto-canvas--lower2d";
    document.body.appendChild(canvas);
    recorder = canvasTest.record();
    wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: { ...aetherTable.REGISTRY, ...value.REGISTRY },
      renderContext: recorder,
      telemFactories: [new telemTest.TestFactory()],
    });
    const project = await client.projects.create({ name: "center", layout: {} });
    // A value cell, not a text cell: value is the variant that draws on the
    // canvas, so its recorded draw calls pin the centering offset into the
    // canvas path alongside the DOM transform.
    const source = telemTest.source("42.5");
    const created = await client.tables.create(project.key, {
      name: "center_table",
      rows: [{ size: ROW_SIZE, cells: ["a"] }],
      columns: [{ size: COL_SIZE }],
      cells: {
        a: {
          key: "a",
          variant: "value",
          props: {
            telem: telemTest.stringSourceSpec(source),
            redline: Value.ZERO_READLINE,
            level: "h5",
            color: "#000000",
            units: "",
          },
        },
      },
    });
    key = created.key;
    await loadTable(wrapper, key);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    document.querySelector(".pluto-canvas--lower2d")?.remove();
  });

  const renderTable = (centered: boolean) => {
    const Wrapped = (): ReactElement => (
      <Table.Suspended tableKey={key}>
        <Table.Table centered={centered} visible />
      </Table.Suspended>
    );
    const { container } = render(<Wrapped />, { wrapper });
    const query = (selector: string) => (): HTMLElement => {
      const el = container.querySelector<HTMLElement>(selector);
      if (el == null) throw new Error(`${selector} did not render`);
      return el;
    };
    return {
      frame: query(".pluto-table-frame"),
      scroller: query(".pluto-table-surface__scroll"),
      probe: query(".pluto-table-surface__canvas"),
    };
  };

  const resizeRow = async (size: number): Promise<void> => {
    const { result } = renderHook(() => Table.useDispatch(), { wrapper });
    await act(async () => {
      await result.current.dispatchAsync({
        key,
        actions: [table.resizeRow({ index: 0, size })],
      });
    });
  };

  // The recorder captures loop.set but never runs the loop, so drive the
  // latest requested render by hand; the table then renders its cells.
  const pumpRender = (): void => {
    const request = recorder.loopCalls.at(-1)?.args[0] as
      { render: () => unknown } | undefined;
    if (request == null) throw new Error("no render was requested");
    request.render();
  };

  // The value cell clips its canvas draw to its own box, so the latest scissor
  // call on the upper2d canvas carries the box the cell last drew at.
  const lastCellBox = (): box.Box => {
    const scissor = recorder.upper2d.calls.findLast((c) => c.op === "scissor");
    if (scissor == null) throw new Error("no cell draw was recorded");
    return scissor.args[0] as box.Box;
  };

  const expectPlacement = async (
    frame: () => HTMLElement,
    offset: xy.XY,
    rowSize: number = ROW_SIZE,
  ) =>
    await waitFor(() => {
      const transform = xy.equals(offset, xy.ZERO)
        ? ""
        : `translate(${offset.x}px, ${offset.y}px)`;
      expect(frame().style.transform).toEqual(transform);
      pumpRender();
      const b = lastCellBox();
      expect(box.topLeft(b)).toEqual(
        xy.translate(offset, { x: INDICATOR_SIZE, y: INDICATOR_SIZE }),
      );
      expect(box.dims(b)).toEqual({ width: COL_SIZE, height: rowSize });
    });

  describe("centering", () => {
    it("leaves an uncentered table untranslated", async () => {
      const { frame } = renderTable(false);
      await expectPlacement(frame, xy.ZERO);
    });

    it("centers on both axes when the table fits", async () => {
      const { frame } = renderTable(true);
      await expectPlacement(frame, expectedOffset(COL_SIZE, ROW_SIZE));
    });

    it("clamps to zero on an axis the table overflows", async () => {
      const { frame } = renderTable(true);
      await resizeRow(SURFACE_HEIGHT * 2);
      await expectPlacement(
        frame,
        { ...expectedOffset(COL_SIZE, SURFACE_HEIGHT * 2), y: 0 },
        SURFACE_HEIGHT * 2,
      );
    });

    it("recenters when the table changes size", async () => {
      const { frame } = renderTable(true);
      await expectPlacement(frame, expectedOffset(COL_SIZE, ROW_SIZE));
      await resizeRow(200);
      await expectPlacement(frame, expectedOffset(COL_SIZE, 200), 200);
    });

    it("holds the offset while the pointer is down", async () => {
      const { frame } = renderTable(true);
      const settled = expectedOffset(COL_SIZE, ROW_SIZE);
      await expectPlacement(frame, settled);
      fireEvent.pointerDown(frame());
      await resizeRow(200);
      expect(frame().style.transform).toEqual(
        `translate(${settled.x}px, ${settled.y}px)`,
      );
      // The cell grew but its origin must stay held with the frame.
      await expectPlacement(frame, settled, 200);
      fireEvent.pointerUp(window);
      await expectPlacement(frame, expectedOffset(COL_SIZE, 200), 200);
    });

    it("releases the held offset on pointer cancellation", async () => {
      const { frame } = renderTable(true);
      await expectPlacement(frame, expectedOffset(COL_SIZE, ROW_SIZE));
      fireEvent.pointerDown(frame());
      await resizeRow(200);
      fireEvent.pointerCancel(window);
      await expectPlacement(frame, expectedOffset(COL_SIZE, 200), 200);
    });
  });

  describe("scrolling", () => {
    const ORIGIN: xy.XY = { x: INDICATOR_SIZE, y: INDICATOR_SIZE };

    const drawnAt = (): xy.XY => {
      const applied = recorder.upper2d.calls.findLast((c) => c.op === "applyScale");
      if (applied == null) throw new Error("no cell draw was recorded");
      return (applied.args[0] as scale.XY).pos(box.topLeft(lastCellBox()));
    };

    const scrollTo = (scroller: () => HTMLElement, to: xy.XY): void => {
      const el = scroller();
      el.scrollLeft = to.x;
      el.scrollTop = to.y;
      fireEvent.scroll(el);
    };

    const expectDrawnAt = async (expected: xy.XY) =>
      await waitFor(() => {
        pumpRender();
        expect(drawnAt()).toEqual(expected);
      });

    it("measures the region from outside the scroll container", () => {
      const { scroller, probe } = renderTable(false);
      expect(scroller().contains(probe())).toBe(false);
    });

    it("draws values at their layout position before any scroll", async () => {
      renderTable(false);
      await expectDrawnAt(ORIGIN);
    });

    it("shifts drawn values by the scroll offset", async () => {
      const { scroller } = renderTable(false);
      await expectDrawnAt(ORIGIN);
      scrollTo(scroller, { x: 40, y: 120 });
      await expectDrawnAt({ x: ORIGIN.x - 40, y: ORIGIN.y - 120 });
    });

    it("keeps clipping to the region the table occupies on screen", async () => {
      const { scroller } = renderTable(false);
      await expectDrawnAt(ORIGIN);
      scrollTo(scroller, { x: 40, y: 120 });
      await expectDrawnAt({ x: ORIGIN.x - 40, y: ORIGIN.y - 120 });
      const clip = recorder.scissorCalls.at(-1)?.region;
      if (clip == null) throw new Error("the table did not clip its draw");
      expect(box.topLeft(clip)).toEqual(xy.ZERO);
      expect(box.dims(clip)).toEqual({
        width: SURFACE_WIDTH,
        height: SURFACE_HEIGHT,
      });
    });
  });
});
