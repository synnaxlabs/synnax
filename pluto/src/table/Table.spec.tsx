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
import { mockBoundingClientRect } from "@synnaxlabs/lyra/testutil";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type border, box, type scale, xy } from "@synnaxlabs/x";
import { act, fireEvent, render, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Table } from "@/table";
import { table as aetherTable } from "@/table/aether";
import { INDICATOR_SIZE } from "@/table/Indicator";
import { createAsyncSynnaxWrapper } from "@/testutil";
import { canvasTest } from "@/vis/render/testutil";
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
      telemFactories: [],
    });
    const project = await client.projects.create({ name: "center" });
    // A value cell, not a text cell: value is the variant that draws on the
    // canvas, so its recorded draw calls pin the centering offset into the
    // canvas path alongside the DOM transform.
    const created = await client.tables.create(project.key, {
      name: "center_table",
      rows: [{ key: "r0", size: ROW_SIZE, cells: ["a"] }],
      columns: [{ size: COL_SIZE }],
      cells: { a: { variant: "value", level: "h5", textColor: "#000000" } },
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
        actions: [table.resizeRow({ key: "r0", size })],
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
    // while the pointer happened to sit over the table. Undo is a tab-wide action; the
    // pointer is parked far outside the table here to hold that line.
    const pressUndo = (): void => {
      fireEvent.mouseMove(window, { clientX: 5000, clientY: 5000 });
      fireEvent.keyDown(window, { key: "Control", code: "ControlLeft" });
      fireEvent.keyDown(window, { code: "KeyZ" });
      fireEvent.keyUp(window, { code: "KeyZ" });
      fireEvent.keyUp(window, { key: "Control", code: "ControlLeft" });
    };

    const createTextTable = async (): Promise<table.Key> => {
      const project = await client.projects.create({ name: "undo" });
      const created = await client.tables.create(project.key, {
        name: "undo_table",
        rows: [{ size: ROW_SIZE, cells: ["a"] }],
        columns: [{ size: COL_SIZE }],
        cells: { a: { variant: "text", value: "before" } },
      });
      await loadTable(wrapper, created.key);
      return created.key;
    };

    it("should undo an edit while the pointer sits outside the table", async () => {
      const textKey = await createTextTable();
      // Without a Triggers.Provider the context's listen is a no-op, so every shortcut
      // assertion below would pass no matter what the table binds.
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
              cell: { key: "a", config: { variant: "text", value: "after" } },
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
              cell: { key: "a", config: { variant: "text", value: "after" } },
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
      const cells: Record<string, table.CellConfig> = {};
      const rowSpecs = Array.from({ length: rows }, (_, r) => ({
        size: ROW_SIZE,
        cells: Array.from({ length: cols }, (_, c) => {
          const cellKey = `${r}-${c}`;
          cells[cellKey] = table.cellConfigZ.parse({
            variant: "value",
            level: "h5",
            textColor: "#000000",
          });
          return cellKey;
        }),
      }));
      const project = await client.projects.create({ name });
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

  describe("moving rows and columns", () => {
    const TRANSLATE = /translate\((-?[\d.]+)px, (-?[\d.]+)px\)/;

    const translation = (el: HTMLElement | null): xy.XY => {
      if (el == null) return xy.ZERO;
      const m = TRANSLATE.exec(el.style.transform);
      const own = m == null ? xy.ZERO : { x: Number(m[1]), y: Number(m[2]) };
      return xy.translate(own, translation(el.parentElement));
    };

    // Every element measures at the surface's top left, moved by the translate
    // transforms on it and its ancestors, as a browser measures it.
    beforeEach(() => {
      vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(
        function (this: HTMLElement) {
          const { x, y } = translation(this);
          return mockBoundingClientRect(y, x, SURFACE_WIDTH, SURFACE_HEIGHT)();
        },
      );
    });

    // Pointer positions are relative to where the cells start, one indicator in from
    // the table's corner.
    const POINTER = { pointerId: 1, button: 0, isPrimary: true };
    const at = ({ x, y }: xy.XY) => ({
      ...POINTER,
      clientX: x + INDICATOR_SIZE,
      clientY: y + INDICATOR_SIZE,
    });

    const press = (label: HTMLElement, to: xy.XY): void => {
      fireEvent.pointerDown(label, at({ x: 5, y: 5 }));
      fireEvent.pointerMove(window, at(to));
    };
    const release = (to: xy.XY): void => {
      fireEvent.pointerUp(window, at(to));
    };
    const drag = (label: HTMLElement, to: xy.XY): void => {
      press(label, to);
      release(to);
    };

    // Moves are coalesced and delivered on the next animation frame.
    const frame = async (): Promise<void> =>
      await act(
        async () =>
          await new Promise<void>((resolve) => requestAnimationFrame(() => resolve())),
      );

    const CELLS = [
      ["00", "01", "02"],
      ["10", "11", "12"],
      ["20", "21", "22"],
    ];

    const createGrid = async (): Promise<table.Key> => {
      const project = await client.projects.create({ name: "move" });
      const created = await client.tables.create(project.key, {
        name: "move_table",
        rows: CELLS.map((cells, y) => ({ key: `r${y}`, size: ROW_SIZE, cells })),
        columns: CELLS.map((_, x) => ({ key: `c${x}`, size: COL_SIZE })),
        cells: Object.fromEntries(
          CELLS.flat().map((k) => [k, { variant: "text", value: `cell ${k}` }]),
        ),
      });
      await loadTable(wrapper, created.key);
      return created.key;
    };

    const renderGrid = async (
      tableKey: table.Key,
      editable: boolean,
      onSelectionChange?: (next: string[]) => void,
      selected?: string[],
    ) => {
      const c = render(
        <Table.Suspended tableKey={tableKey}>
          <Table.Table
            editable={editable}
            visible
            onSelectionChange={onSelectionChange}
            selected={selected}
          />
        </Table.Suspended>,
        { wrapper },
      );
      await waitFor(() => expect(c.getByText("cell 00")).toBeTruthy());
      return c;
    };

    const expectStored = async (tableKey: table.Key, cells: string[][]) =>
      await waitFor(async () => {
        const stored = await client.tables.retrieve({ key: tableKey });
        expect(stored.rows.map((r) => r.cells)).toEqual(cells);
      });

    it("should move a column dragged past its neighbors", async () => {
      const tableKey = await createGrid();
      const c = await renderGrid(tableKey, true);
      drag(c.getByText("A"), { x: 200, y: 5 });
      await expectStored(tableKey, [
        ["01", "02", "00"],
        ["11", "12", "10"],
        ["21", "22", "20"],
      ]);
    });

    it("should move a row down past its neighbor", async () => {
      const tableKey = await createGrid();
      const c = await renderGrid(tableKey, true);
      drag(c.getByText("1"), { x: 5, y: 30 });
      await expectStored(tableKey, [CELLS[1], CELLS[0], CELLS[2]]);
    });

    it("should move a row up to the top", async () => {
      const tableKey = await createGrid();
      const c = await renderGrid(tableKey, true);
      drag(c.getByText("3"), { x: 5, y: -60 });
      await expectStored(tableKey, [CELLS[2], CELLS[0], CELLS[1]]);
    });

    it("should move the dragged row when another session adds a row", async () => {
      const tableKey = await createGrid();
      const c = await renderGrid(tableKey, true);
      press(c.getByText("1"), { x: 5, y: 30 });
      await frame();
      const config = table.cellConfigZ.parse({ variant: "text", value: "cell new" });
      await client.tables.dispatch(tableKey, [
        table.addRow({ index: 0, size: ROW_SIZE, cells: [{ key: "new", config }] }),
      ]);
      await waitFor(() => expect(c.getByText("cell new")).toBeTruthy());
      fireEvent.pointerMove(window, at({ x: 5, y: 70 }));
      await frame();
      release({ x: 5, y: 70 });
      await expectStored(tableKey, [["new"], CELLS[1], CELLS[0], CELLS[2]]);
    });

    it("should slide the cells to preview where a column lands", async () => {
      const tableKey = await createGrid();
      const c = await renderGrid(tableKey, true);
      const transformOf = (selector: string): string => {
        const el = c.container.querySelector<HTMLElement>(selector);
        if (el == null) throw new Error(`${selector} did not render`);
        return el.style.transform;
      };
      const frameEl = () => c.container.querySelector(".pluto-table-frame");
      press(c.getByText("A"), { x: 200, y: 5 });
      await frame();
      expect(frameEl()?.classList).toContain("pluto-table-frame--moving");
      expect(transformOf("[id='00']")).toEqual("translate(144px, 0px)");
      expect(transformOf("[id='21']")).toEqual("translate(-72px, 0px)");
      expect(transformOf("#resizer-x-c0")).toEqual("translate(144px, 0px)");
      expect(transformOf("#resizer-x-c2")).toEqual("translate(-72px, 0px)");
      expect(transformOf("#resizer-y-r0")).toEqual("");
      expect(transformOf(".pluto-table-frame__move-source")).toEqual(
        "translate(144px, 0px)",
      );
      release({ x: 200, y: 5 });
      expect(frameEl()?.classList).not.toContain("pluto-table-frame--moving");
      expect(transformOf("[id='00']")).toEqual("");
      expect(c.container.querySelector(".pluto-table-frame__move-source")).toBeNull();
    });

    it("should hold the landing while the pressed indicator slides", async () => {
      const tableKey = await createGrid();
      const c = await renderGrid(tableKey, true);
      const transform = () =>
        c.container.querySelector<HTMLElement>("[id='00']")?.style.transform;
      press(c.getByText("A"), { x: 200, y: 5 });
      await frame();
      expect(transform()).toEqual("translate(144px, 0px)");
      fireEvent.pointerMove(window, at({ x: 201, y: 5 }));
      await frame();
      expect(transform()).toEqual("translate(144px, 0px)");
      release({ x: 201, y: 5 });
      await expectStored(tableKey, [
        ["01", "02", "00"],
        ["11", "12", "10"],
        ["21", "22", "20"],
      ]);
    });

    it("should keep the moved cells selected through the closing click", async () => {
      const tableKey = await createGrid();
      const onSelectionChange = vi.fn();
      const c = await renderGrid(tableKey, true, onSelectionChange);
      const label = c.getByText("1");
      drag(label, { x: 5, y: 30 });
      fireEvent.click(label);
      await expectStored(tableKey, [CELLS[1], CELLS[0], CELLS[2]]);
      expect(onSelectionChange).toHaveBeenCalledTimes(1);
      expect(onSelectionChange).toHaveBeenLastCalledWith(CELLS[0]);
    });

    it("should select a moved column's cells", async () => {
      const tableKey = await createGrid();
      const onSelectionChange = vi.fn();
      const c = await renderGrid(tableKey, true, onSelectionChange);
      const label = c.getByText("C");
      drag(label, { x: -140, y: 5 });
      fireEvent.click(label);
      await expectStored(tableKey, [
        ["02", "00", "01"],
        ["12", "10", "11"],
        ["22", "20", "21"],
      ]);
      expect(onSelectionChange).toHaveBeenCalledTimes(1);
      expect(onSelectionChange).toHaveBeenLastCalledWith(["02", "12", "22"]);
    });

    it("should move every selected row as a block", async () => {
      const tableKey = await createGrid();
      const onSelectionChange = vi.fn();
      const c = await renderGrid(tableKey, true, onSelectionChange, [
        ...CELLS[0],
        ...CELLS[1],
      ]);
      const transform = (selector: string) =>
        c.container.querySelector<HTMLElement>(selector)?.style.transform;
      press(c.getByText("1"), { x: 5, y: 30 });
      await frame();
      expect(transform("[id='00']")).toEqual("translate(0px, 36px)");
      expect(transform("[id='12']")).toEqual("translate(0px, 36px)");
      expect(transform("[id='20']")).toEqual("translate(0px, -72px)");
      expect(
        c.container.querySelectorAll(".pluto-table-frame__move-source"),
      ).toHaveLength(2);
      release({ x: 5, y: 30 });
      await expectStored(tableKey, [CELLS[2], CELLS[0], CELLS[1]]);
      expect(onSelectionChange).toHaveBeenLastCalledWith([...CELLS[0], ...CELLS[1]]);
    });

    it("should close a gapped selection into one block", async () => {
      const tableKey = await createGrid();
      const onSelectionChange = vi.fn();
      const c = await renderGrid(tableKey, true, onSelectionChange, [
        ...CELLS[0],
        ...CELLS[2],
      ]);
      drag(c.getByText("1"), { x: 5, y: 200 });
      await expectStored(tableKey, [CELLS[1], CELLS[0], CELLS[2]]);
      expect(onSelectionChange).toHaveBeenLastCalledWith([...CELLS[0], ...CELLS[2]]);
    });

    it("should move every selected column as a block", async () => {
      const tableKey = await createGrid();
      const c = await renderGrid(tableKey, true, undefined, [
        ...CELLS.map((row) => row[0]),
        ...CELLS.map((row) => row[1]),
      ]);
      drag(c.getByText("A"), { x: 200, y: 5 });
      await expectStored(tableKey, [
        ["02", "00", "01"],
        ["12", "10", "11"],
        ["22", "20", "21"],
      ]);
    });

    it("should move only the pressed row when it is not selected", async () => {
      const tableKey = await createGrid();
      const c = await renderGrid(tableKey, true, undefined, CELLS[1]);
      drag(c.getByText("3"), { x: 5, y: -60 });
      await expectStored(tableKey, [CELLS[2], CELLS[0], CELLS[1]]);
    });

    it("should drop a move when the pointer is cancelled", async () => {
      const tableKey = await createGrid();
      const onSelectionChange = vi.fn();
      const c = await renderGrid(tableKey, true, onSelectionChange);
      press(c.getByText("A"), { x: 200, y: 5 });
      await frame();
      fireEvent.pointerCancel(window, at({ x: 200, y: 5 }));
      expect(c.container.querySelector(".pluto-table-frame--moving")).toBeNull();
      expect(onSelectionChange).not.toHaveBeenCalled();
      await expectStored(tableKey, CELLS);
    });

    it("should blank the value cells a move displaces until it ends", async () => {
      const keys = ["a", "b", "c"];
      const project = await client.projects.create({ name: "move_values" });
      const created = await client.tables.create(project.key, {
        name: "move_values_table",
        rows: [{ size: ROW_SIZE, cells: keys }],
        columns: keys.map(() => ({ size: COL_SIZE })),
        cells: Object.fromEntries(
          keys.map((k) => [k, { variant: "value", level: "h5", textColor: "#000000" }]),
        ),
      });
      await loadTable(wrapper, created.key);
      const c = render(
        <Table.Suspended tableKey={created.key}>
          <Table.Table editable visible />
        </Table.Suspended>,
        { wrapper },
      );
      // The left edges of the cells the next render draws, ascending.
      const drawn = (): number[] => {
        const before = scissors().length;
        pumpRender();
        return scissors()
          .slice(before)
          .map((s) => box.topLeft(s.args[0] as box.Box).x)
          .sort((a, b) => a - b);
      };
      const ALL = keys.map((_, i) => INDICATOR_SIZE + i * COL_SIZE);
      await waitFor(() => expect(drawn()).toEqual(ALL));
      // A lands past B, so A and B slide and C stays put.
      press(c.getByText("A"), { x: 100, y: 5 });
      await frame();
      await waitFor(() => expect(drawn()).toEqual([ALL[2]]));
      release({ x: 100, y: 5 });
      await waitFor(() => expect(drawn()).toEqual(ALL));
    });

    it("should keep a press that never travels a click", async () => {
      const tableKey = await createGrid();
      const onSelectionChange = vi.fn();
      const c = await renderGrid(tableKey, true, onSelectionChange);
      const label = c.getByText("A");
      drag(label, { x: 7, y: 5 });
      fireEvent.click(label);
      expect(onSelectionChange).toHaveBeenLastCalledWith(["00", "10", "20"]);
      expect(c.container.querySelector(".pluto-table-frame--moving")).toBeNull();
      await expectStored(tableKey, CELLS);
    });

    it("should not move anything when the table is not editable", async () => {
      const tableKey = await createGrid();
      const c = await renderGrid(tableKey, false);
      press(c.getByText("A"), { x: 200, y: 5 });
      await frame();
      expect(c.container.querySelector(".pluto-table-frame--moving")).toBeNull();
      release({ x: 200, y: 5 });
      await expectStored(tableKey, CELLS);
    });
  });

  describe("sparse cell configs", () => {
    // A value cell stored with only its variant takes every other field from the
    // schema defaults, so it renders instead of crashing on missing fields.
    it("renders a value cell carrying only its variant", async () => {
      const project = await client.projects.create({ name: "sparse" });
      const created = await client.tables.create(project.key, {
        name: "sparse_table",
        rows: [{ size: ROW_SIZE, cells: ["a"] }],
        columns: [{ size: COL_SIZE }],
        cells: { a: { variant: "value" } },
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
