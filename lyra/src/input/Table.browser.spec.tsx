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
});
