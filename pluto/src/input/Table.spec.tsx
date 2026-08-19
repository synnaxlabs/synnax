// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Input } from "@/input";

const COLUMNS = [{ name: "Raw" }, { name: "Scaled" }];

interface RenderOptions {
  value?: number[][];
  preview?: boolean;
  createRow?: (value: number[][]) => number[];
  rowLabel?: (index: number) => string;
}

const renderTable = ({ value = [[1, 10]], ...rest }: RenderOptions = {}) => {
  const onChange = vi.fn();
  render(<Input.Table columns={COLUMNS} value={value} onChange={onChange} {...rest} />);
  return onChange;
};

const cell = (column: string, row: string): HTMLInputElement =>
  screen.getByLabelText(`${column} ${row}`);

const commit = (input: HTMLInputElement, value: string): void => {
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const paste = (input: HTMLInputElement, text: string): void => {
  fireEvent.focus(input);
  fireEvent.paste(input, { clipboardData: { getData: () => text } });
};

const addButton = (): HTMLElement => screen.getAllByRole("button")[0];

const removeButton = (row: number): HTMLElement =>
  screen.getAllByRole("row")[row + 1].querySelectorAll("button")[0];

describe("Input.Table", () => {
  it("should render a row per entry and a cell per column", () => {
    renderTable({
      value: [
        [1, 10],
        [2, 20],
      ],
    });
    expect(screen.getAllByRole("row")).toHaveLength(3);
    expect(cell("Raw", "2").value).toBe("2");
    expect(cell("Scaled", "1").value).toBe("10");
  });

  it("should label rows by their one-based index", () => {
    renderTable({
      value: [
        [1, 10],
        [2, 20],
      ],
    });
    expect(screen.getByRole("rowheader", { name: "1" })).toBeTruthy();
    expect(screen.getByRole("rowheader", { name: "2" })).toBeTruthy();
  });

  it("should label rows with the given labeler", () => {
    renderTable({ rowLabel: (i) => `c${i}` });
    expect(screen.getByRole("rowheader", { name: "c0" })).toBeTruthy();
  });

  it("should commit an edited cell without touching the rest of the row", () => {
    const onChange = renderTable({
      value: [
        [1, 10],
        [2, 20],
      ],
    });
    commit(cell("Scaled", "1"), "15");
    expect(onChange).toHaveBeenCalledWith([
      [1, 15],
      [2, 20],
    ]);
  });

  it("should append a row of zeros", () => {
    const onChange = renderTable();
    fireEvent.click(addButton());
    expect(onChange).toHaveBeenCalledWith([
      [1, 10],
      [0, 0],
    ]);
  });

  it("should append the row built by createRow", () => {
    const onChange = renderTable({
      createRow: (value) => [(value.at(-1)?.[0] ?? 0) + 1, 0],
    });
    fireEvent.click(addButton());
    expect(onChange).toHaveBeenCalledWith([
      [1, 10],
      [2, 0],
    ]);
  });

  it("should remove the row whose button is clicked", () => {
    const onChange = renderTable({
      value: [
        [1, 10],
        [2, 20],
        [3, 30],
      ],
    });
    fireEvent.click(removeButton(1));
    expect(onChange).toHaveBeenCalledWith([
      [1, 10],
      [3, 30],
    ]);
  });

  it("should hide the add and remove buttons in preview", () => {
    renderTable({ preview: true });
    expect(screen.queryAllByRole("button")).toHaveLength(0);
  });

  describe("keyboard", () => {
    const VALUE = [
      [1, 10],
      [2, 20],
    ];

    it("should move Enter to the next cell in row-major order", () => {
      renderTable({ value: VALUE });
      const from = cell("Raw", "1");
      from.focus();
      fireEvent.keyDown(from, { key: "Enter" });
      expect(document.activeElement).toBe(cell("Scaled", "1"));
    });

    it("should wrap Enter onto the next row", () => {
      renderTable({ value: VALUE });
      const from = cell("Scaled", "1");
      from.focus();
      fireEvent.keyDown(from, { key: "Enter" });
      expect(document.activeElement).toBe(cell("Raw", "2"));
    });

    it("should move shift+Enter backwards", () => {
      renderTable({ value: VALUE });
      const from = cell("Raw", "2");
      from.focus();
      fireEvent.keyDown(from, { key: "Enter", shiftKey: true });
      expect(document.activeElement).toBe(cell("Scaled", "1"));
    });

    it("should move the down arrow to the row below", () => {
      renderTable({ value: VALUE });
      const from = cell("Scaled", "1");
      from.focus();
      fireEvent.keyDown(from, { key: "ArrowDown" });
      expect(document.activeElement).toBe(cell("Scaled", "2"));
    });

    it("should move the up arrow to the row above", () => {
      renderTable({ value: VALUE });
      const from = cell("Raw", "2");
      from.focus();
      fireEvent.keyDown(from, { key: "ArrowUp" });
      expect(document.activeElement).toBe(cell("Raw", "1"));
    });

    it("should leave a key pressed outside a cell alone", () => {
      renderTable({ value: VALUE });
      expect(fireEvent.keyDown(addButton(), { key: "Enter" })).toBe(true);
      expect(fireEvent.keyDown(cell("Raw", "1"), { key: "Enter" })).toBe(false);
    });

    it("should hold focus at the last cell", () => {
      renderTable({ value: VALUE });
      const from = cell("Scaled", "2");
      from.focus();
      fireEvent.keyDown(from, { key: "Enter" });
      expect(document.activeElement).toBe(from);
    });
  });

  describe("paste", () => {
    it("should fill the grid from the focused cell, adding rows", () => {
      const onChange = renderTable();
      paste(cell("Raw", "1"), "1\t10\n2\t20\n3\t30");
      expect(onChange).toHaveBeenCalledWith([
        [1, 10],
        [2, 20],
        [3, 30],
      ]);
    });

    it("should anchor the paste on the focused cell", () => {
      const onChange = renderTable();
      paste(cell("Scaled", "1"), "11\n21");
      expect(onChange).toHaveBeenCalledWith([
        [1, 11],
        [0, 21],
      ]);
    });

    it("should accept a comma delimited block", () => {
      const onChange = renderTable();
      paste(cell("Raw", "1"), "1,10\n2,20");
      expect(onChange).toHaveBeenCalledWith([
        [1, 10],
        [2, 20],
      ]);
    });

    it("should drop a heading row", () => {
      const onChange = renderTable();
      paste(cell("Raw", "1"), "raw\tscaled\n1\t10\n2\t20");
      expect(onChange).toHaveBeenCalledWith([
        [1, 10],
        [2, 20],
      ]);
    });

    it("should ignore columns past the last one", () => {
      const onChange = renderTable();
      paste(cell("Raw", "1"), "1\t10\t100");
      expect(onChange).toHaveBeenCalledWith([[1, 10]]);
    });

    it("should stay contiguous when the anchored row is gone", () => {
      const onChange = vi.fn();
      const { rerender } = render(
        <Input.Table
          columns={COLUMNS}
          value={[
            [1, 10],
            [2, 20],
            [3, 30],
          ]}
          onChange={onChange}
        />,
      );
      fireEvent.focus(cell("Raw", "3"));
      rerender(<Input.Table columns={COLUMNS} value={[[1, 10]]} onChange={onChange} />);
      fireEvent.paste(cell("Raw", "1"), {
        clipboardData: { getData: () => "9\t90\n8\t80" },
      });
      expect(onChange).toHaveBeenCalledWith([
        [1, 10],
        [9, 90],
        [8, 80],
      ]);
    });

    it("should leave a single value to the cell", () => {
      const onChange = renderTable();
      paste(cell("Raw", "1"), "5");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should leave a block holding a non-numeric row to the cell", () => {
      const onChange = renderTable();
      paste(cell("Raw", "1"), "1\t10\n2\tnot a number");
      expect(onChange).not.toHaveBeenCalled();
    });
  });
});
