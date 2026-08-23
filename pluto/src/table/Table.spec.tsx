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
import { type border, box, type scale, xy } from "@synnaxlabs/x";
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
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";
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

const THEME = Theming.themeZ.parse(Theming.SYNNAX_THEMES.synnaxLight);
const RADIUS = THEME.sizes.border.radius.small * THEME.sizes.base;

// Single-hook bootstrap so the suspending useEnsure is not followed by other
// hooks, which trips a React 19 concurrent replay warning (same pattern as
// table queries.spec.tsx).
const loadTable = async (
  wrapper: React.FC<PropsWithChildren>,
  key: table.Key,
): Promise<void> => {
  const testID = `loaded-${key}`;
  const Bootstrap = (): ReactElement => {
    Table.useEnsure({ key });
    return <div data-testid={testID} />;
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
  await utils.findByTestId(testID);
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

  const scissors = (): canvasTest.Call[] =>
    recorder.upper2d.calls.filter((c) => c.op === "scissor");

  const lastScissor = (): canvasTest.Call => {
    const scissor = scissors().at(-1);
    if (scissor == null) throw new Error("no cell draw was recorded");
    return scissor;
  };

  // The value cell clips its canvas draw to its own box, so the latest scissor
  // call on the upper2d canvas carries the box the cell last drew at.
  const lastCellBox = (): box.Box => lastScissor().args[0] as box.Box;

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

    it("shifts drawn values by the scroll offset", async () => {
      const { scroller } = renderTable(false);
      await expectDrawnAt(ORIGIN);
      scrollTo(scroller, { x: 10, y: 20 });
      await expectDrawnAt({ x: ORIGIN.x - 10, y: ORIGIN.y - 20 });
    });
  });

  describe("undo and redo triggers", () => {
    // The shortcut used to be filtered by the table's own region, so it only fired
    // while the pointer happened to sit over the table. Undo is a tab-wide action;
    // the pointer is parked far outside the table here to hold that line.
    const pressUndo = (): void => {
      fireEvent.mouseMove(window, { clientX: 5000, clientY: 5000 });
      fireEvent.keyDown(window, { key: "Control", code: "ControlLeft" });
      fireEvent.keyDown(window, { code: "KeyZ" });
      fireEvent.keyUp(window, { code: "KeyZ" });
      fireEvent.keyUp(window, { key: "Control", code: "ControlLeft" });
    };

    const createTextTable = async (): Promise<table.Key> => {
      const project = await client.projects.create({ name: "undo", layout: {} });
      const created = await client.tables.create(project.key, {
        name: "undo_table",
        rows: [{ size: ROW_SIZE, cells: ["a"] }],
        columns: [{ size: COL_SIZE }],
        cells: { a: { key: "a", variant: "text", props: { value: "before" } } },
      });
      await loadTable(wrapper, created.key);
      return created.key;
    };

    it("should undo an edit while the pointer sits outside the table", async () => {
      const textKey = await createTextTable();
      // Without a Triggers.Provider the context's listen is a no-op, so every
      // shortcut assertion below would pass no matter what the table binds.
      const c = render(
        <Triggers.Provider>
          <Table.Suspended tableKey={textKey}>
            <Table.Table editable visible />
          </Table.Suspended>
        </Triggers.Provider>,
        { wrapper },
      );
      const { result } = renderHook(() => Table.useDispatch(), { wrapper });
      await waitFor(() => expect(c.getByText("before")).toBeTruthy());
      await act(async () => {
        await result.current.dispatchAsync({
          key: textKey,
          actions: [
            table.setCell({
              cell: { key: "a", variant: "text", props: { value: "after" } },
            }),
          ],
        });
      });
      await waitFor(() => expect(c.getByText("after")).toBeTruthy());
      act(pressUndo);
      await waitFor(() => expect(c.getByText("before")).toBeTruthy());
    });

    it("should withhold undo from a table that is not editable", async () => {
      const textKey = await createTextTable();
      const c = render(
        <Triggers.Provider>
          <Table.Suspended tableKey={textKey}>
            <Table.Table visible />
          </Table.Suspended>
        </Triggers.Provider>,
        { wrapper },
      );
      const { result } = renderHook(() => Table.useDispatch(), { wrapper });
      await waitFor(() => expect(c.getByText("before")).toBeTruthy());
      await act(async () => {
        await result.current.dispatchAsync({
          key: textKey,
          actions: [
            table.setCell({
              cell: { key: "a", variant: "text", props: { value: "after" } },
            }),
          ],
        });
      });
      await waitFor(() => expect(c.getByText("after")).toBeTruthy());
      act(pressUndo);
      await act(async () => {});
      expect(c.getByText("after")).toBeTruthy();
    });
  });

  describe("corner radius", () => {
    const corner = (name: keyof border.Radius): Record<string, number> => ({
      topLeft: 0,
      topRight: 0,
      bottomRight: 0,
      bottomLeft: 0,
      [name]: RADIUS,
    });

    // Renders a fresh grid of value cells and returns the clip radius each one drew
    // with, in row-major order.
    const renderGrid = async (
      name: string,
      rows: number,
      cols: number,
      showIndicators: boolean,
    ): Promise<unknown[]> => {
      const source = telemTest.source("1");
      const cells: Record<string, table.Cell> = {};
      const rowSpecs = Array.from({ length: rows }, (_, r) => ({
        size: ROW_SIZE,
        cells: Array.from({ length: cols }, (_, c) => {
          const cellKey = `${r}-${c}`;
          cells[cellKey] = {
            key: cellKey,
            variant: "value",
            props: {
              telem: telemTest.stringSourceSpec(source),
              redline: Value.ZERO_READLINE,
              level: "h5",
              color: "#000000",
              units: "",
            },
          };
          return cellKey;
        }),
      }));
      const project = await client.projects.create({ name, layout: {} });
      const created = await client.tables.create(project.key, {
        name: `${name}_table`,
        rows: rowSpecs,
        columns: Array.from({ length: cols }, () => ({ size: COL_SIZE })),
        cells,
      });
      await loadTable(wrapper, created.key);
      const Wrapped = (): ReactElement => (
        <Table.Suspended tableKey={created.key}>
          <Table.Table visible showIndicators={showIndicators} />
        </Table.Suspended>
      );
      render(<Wrapped />, { wrapper });
      let radii: unknown[] = [];
      await waitFor(() => {
        pumpRender();
        // One scissor per cell, so the last rows * cols are this render's.
        radii = scissors()
          .slice(-rows * cols)
          .map((c) => c.args[2]);
        expect(radii).toHaveLength(rows * cols);
      });
      return radii;
    };

    // CSS rounds the table's outer corners on the corner <td>, so a cell painting its
    // background on the canvas has to clip to the same radius or it fills the corner
    // square. The indicators hold the first row and column when shown, leaving the
    // bottom-right cell as the only data cell at a corner.
    it("rounds only the bottom-right cell when the indicators are shown", async () => {
      const radii = await renderGrid("indicated", 2, 2, true);
      expect(radii).toEqual([undefined, undefined, undefined, corner("bottomRight")]);
    });

    it("rounds all four corners when the indicators are hidden", async () => {
      const radii = await renderGrid("bare", 2, 3, false);
      expect(radii).toEqual([
        corner("topLeft"),
        undefined,
        corner("topRight"),
        corner("bottomLeft"),
        undefined,
        corner("bottomRight"),
      ]);
    });
  });

  describe("sparse cell props", () => {
    // An imported legacy state can carry a value cell with empty props. The
    // renderer merges wire props over the variant's defaults, so the cell
    // renders instead of crashing on the missing fields.
    it("renders a value cell whose wire props are empty", async () => {
      const project = await client.projects.create({ name: "sparse", layout: {} });
      const created = await client.tables.create(project.key, {
        name: "sparse_table",
        rows: [{ size: ROW_SIZE, cells: ["a"] }],
        columns: [{ size: COL_SIZE }],
        cells: { a: { key: "a", variant: "value", props: {} } },
      });
      await loadTable(wrapper, created.key);
      const Wrapped = (): ReactElement => (
        <Table.Suspended tableKey={created.key}>
          <Table.Table visible />
        </Table.Suspended>
      );
      const { container } = render(<Wrapped />, { wrapper });
      await waitFor(() => {
        expect(container.querySelector(".pluto-table-frame")).not.toBeNull();
        expect(container.querySelector(".pluto-table__cell--value")).not.toBeNull();
      });
    });
  });
});
