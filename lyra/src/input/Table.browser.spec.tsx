// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Input } from "@/input";
import { computed, renderThemed, resolve } from "@/styletest";

type ColumnType = NonNullable<Input.TableColumnProps["type"]>;

interface TableOptions {
  type: ColumnType;
  rows: number;
  columns: number;
  named?: boolean;
  rowLabel?: Input.TableProps["rowLabel"];
  width?: number;
}

/** @returns the frame of every cell, one array per row. */
const renderTable = async ({
  type,
  rows,
  columns,
  named = false,
  rowLabel,
  width,
}: TableOptions): Promise<Element[][]> => {
  const value = Array.from({ length: rows }, () => Array<number>(columns).fill(1));
  const { container } = await renderThemed(
    <Input.Table
      value={value}
      onChange={() => {}}
      rowLabel={rowLabel}
      style={{ width }}
    >
      {value[0].map((_, i) => (
        <Input.TableColumn
          key={i}
          name={named ? `Column ${i}` : undefined}
          type={type}
        />
      ))}
    </Input.Table>,
  );
  return Array.from(container.querySelectorAll("tbody tr"), (row) =>
    Array.from(row.querySelectorAll(`.${CSS.B("input")}`)),
  );
};

/** @returns the radii in order: top left, top right, bottom right, bottom left. */
const radii = (el: Element): string[] =>
  ["top-left", "top-right", "bottom-right", "bottom-left"].map((corner) =>
    computed(el, `border-${corner}-radius`),
  );

const SQUARE = "0px";

const rounded = (): string =>
  resolve(
    "calc(var(--pluto-border-radius-tiny) - var(--pluto-border-width))",
    "border-top-left-radius",
  );

/** @returns the table, in a container wider than its columns. */
const renderWide = async (widths: (number | undefined)[]): Promise<HTMLTableElement> =>
  (
    await renderThemed(
      <div style={{ width: "2000px" }}>
        <Input.Table value={[widths.map((_, i) => i)]} onChange={() => {}}>
          {widths.map((width, i) => (
            <Input.TableColumn key={i} name={`Column ${i}`} width={width} />
          ))}
        </Input.Table>
      </div>,
    )
  ).container.querySelector("table")!;

/** @returns the area that the outline of the element covers. */
const outlineOf = (el: Element): DOMRect => {
  const style = getComputedStyle(el);
  const reach = parseFloat(style.outlineOffset) + parseFloat(style.outlineWidth);
  const { x, y, width, height } = el.getBoundingClientRect();
  return new DOMRect(x - reach, y - reach, width + 2 * reach, height + 2 * reach);
};

const expectWithin = (inner: DOMRect, outer: DOMRect): void => {
  expect(inner.left).toBeGreaterThanOrEqual(outer.left);
  expect(inner.top).toBeGreaterThanOrEqual(outer.top);
  expect(inner.right).toBeLessThanOrEqual(outer.right);
  expect(inner.bottom).toBeLessThanOrEqual(outer.bottom);
};

const GUTTERS: [string, Input.TableProps["rowLabel"]][] = [
  ["without", undefined],
  ["with", (i) => `c${i}`],
];

/** @returns the table and its scroll box, in a scrolling parent with no padding. */
const renderInScroller = async (
  rowLabel: Input.TableProps["rowLabel"],
): Promise<{ parent: Element; scroll: Element; table: HTMLTableElement }> => {
  const { container } = await renderThemed(
    <div style={{ overflow: "auto", width: "600px" }}>
      <Input.Table value={[[1], [2]]} onChange={() => {}} rowLabel={rowLabel}>
        <Input.TableColumn />
      </Input.Table>
    </div>,
  );
  const table = container.querySelector("table")!;
  return {
    parent: container.firstElementChild!,
    scroll: table.parentElement!,
    table,
  };
};

describe("Input.Table", () => {
  it("should hold the remove button as far off the box as a row label", async () => {
    const { container } = await renderThemed(
      <Input.Table value={[[1]]} onChange={() => {}} rowLabel={(i) => `c${i}`}>
        <Input.TableColumn />
      </Input.Table>,
    );
    const [gutter, cell, action] = container.querySelector("tbody tr")!.children;
    const label = gutter.firstElementChild!.getBoundingClientRect();
    const box = cell.getBoundingClientRect();
    const button = action.querySelector("button")!.getBoundingClientRect();
    const gap = button.left - box.right;
    expect(gap).toBeGreaterThan(0);
    expect(gap).toBeCloseTo(box.left - label.right);
    expect(button.right).toBeLessThanOrEqual(
      container.querySelector("table")!.getBoundingClientRect().right,
    );
  });

  describe.each<ColumnType>(["string", "number"])("%s cells", (type) => {
    it("should square a cell away from the corners of the box", async () => {
      const cells = await renderTable({ type, rows: 3, columns: 3, named: true });
      expect(radii(cells[1][1])).toEqual([SQUARE, SQUARE, SQUARE, SQUARE]);
    });

    it("should leave the top corners to the headings", async () => {
      const cells = await renderTable({ type, rows: 2, columns: 2, named: true });
      expect(radii(cells[0][0])).toEqual([SQUARE, SQUARE, SQUARE, SQUARE]);
      expect(radii(cells[0][1])).toEqual([SQUARE, SQUARE, SQUARE, SQUARE]);
    });

    it("should round only the outer corner of the bottom cells", async () => {
      const cells = await renderTable({ type, rows: 2, columns: 2, named: true });
      const r = rounded();
      expect(radii(cells[1][0])).toEqual([SQUARE, SQUARE, SQUARE, r]);
      expect(radii(cells[1][1])).toEqual([SQUARE, SQUARE, r, SQUARE]);
    });

    it("should round the top cells when there are no headings", async () => {
      const cells = await renderTable({ type, rows: 2, columns: 2 });
      const r = rounded();
      expect(radii(cells[0][0])).toEqual([r, SQUARE, SQUARE, SQUARE]);
      expect(radii(cells[0][1])).toEqual([SQUARE, r, SQUARE, SQUARE]);
    });

    it("should round every corner of a lone cell beside a gutter", async () => {
      const cells = await renderTable({
        type,
        rows: 1,
        columns: 1,
        rowLabel: (i) => `c${i}`,
      });
      const r = rounded();
      expect(radii(cells[0][0])).toEqual([r, r, r, r]);
    });

    it("should fill a cell wider than its input", async () => {
      const cells = await renderTable({ type, rows: 1, columns: 1, width: 800 });
      const frame = cells[0][0];
      const cell = frame.closest("td")!;
      expect(frame.getBoundingClientRect().width).toBeCloseTo(cell.clientWidth);
      expect(frame.getBoundingClientRect().height).toBeCloseTo(cell.clientHeight);
    });
  });

  describe("wrapped cell controls", () => {
    /** @returns the shadow control in every cell, each inside a wrapper, by row. */
    const renderWrapped = async (
      rows: number,
      columns: number,
    ): Promise<Element[][]> => {
      const value = Array.from({ length: rows }, () => Array<string>(columns).fill(""));
      const { container } = await renderThemed(
        <Input.Table value={value} onChange={() => {}}>
          {value[0].map((_, i) => (
            <Input.TableColumn key={i} name={`Column ${i}`} type="string">
              {({ value, onChange, variant }) => (
                <div>
                  <Input.Text value={value} onChange={onChange} variant={variant} />
                </div>
              )}
            </Input.TableColumn>
          ))}
        </Input.Table>,
      );
      return Array.from(container.querySelectorAll("tbody tr"), (row) =>
        Array.from(row.querySelectorAll(`.${CSS.BM("btn", "shadow")}`)),
      );
    };

    it("should square a control away from the corners of the box", async () => {
      const cells = await renderWrapped(3, 3);
      expect(radii(cells[1][1])).toEqual([SQUARE, SQUARE, SQUARE, SQUARE]);
    });

    it("should round only the outer corner of the bottom controls", async () => {
      const cells = await renderWrapped(2, 2);
      const r = rounded();
      expect(radii(cells[1][0])).toEqual([SQUARE, SQUARE, SQUARE, r]);
      expect(radii(cells[1][1])).toEqual([SQUARE, SQUARE, r, SQUARE]);
    });
  });

  describe.each(GUTTERS)("focus ring %s a gutter", (_, rowLabel) => {
    it("should keep the ring of the add button inside the scroll box", async () => {
      const { scroll, table } = await renderInScroller(rowLabel);
      const add = table.querySelector<HTMLButtonElement>("tfoot button")!;
      // WebKit leaves buttons out of the tab order.
      add.focus({ focusVisible: true });
      expect(computed(add, "outline-style")).toBe("solid");
      expectWithin(outlineOf(add), scroll.getBoundingClientRect());
    });

    it("should keep the frame of a first column cell inside the scroll box", async () => {
      const { scroll, table } = await renderInScroller(rowLabel);
      const frame = table.querySelector(`tbody .${CSS.BM("btn", "shadow")}`)!;
      expectWithin(outlineOf(frame), scroll.getBoundingClientRect());
    });

    it("should take the space of the table and add no overflow", async () => {
      const { parent, table } = await renderInScroller(rowLabel);
      const box = parent.getBoundingClientRect();
      const t = table.getBoundingClientRect();
      expect(t.left).toBe(box.left);
      expect(t.top).toBe(box.top);
      expect(t.width).toBe(parent.clientWidth);
      expect(t.height).toBe(parent.clientHeight);
      expect(parent.scrollWidth).toBe(parent.clientWidth);
      expect(parent.scrollHeight).toBe(parent.clientHeight);
    });
  });

  describe("width", () => {
    it("should keep declared column widths in a wider container", async () => {
      const table = await renderWide([10, 12]);
      expect(table.getBoundingClientRect().width).toBe(
        parseFloat(computed(table, "min-width")),
      );
    });

    it("should fill a wider container when a column has no width", async () => {
      const table = await renderWide([10, undefined]);
      expect(table.getBoundingClientRect().width).toBe(2000);
    });
  });
});
