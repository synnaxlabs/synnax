// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Input } from "@/input";
import { type Status } from "@/status";

const COLUMNS = [
  <Input.TableColumn key="raw" name="Raw" />,
  <Input.TableColumn key="scaled" name="Scaled" />,
];

interface RenderOptions {
  value?: Input.TableCell[][];
  preview?: boolean;
  createRow?: (value: Input.TableCell[][]) => Input.TableCell[];
  rowLabel?: (index: number) => string;
  resourceName?: string;
  cellStatus?: (at: xy.XY) => Status.Crude | undefined;
}

const renderTable = ({ value = [[1, 10]], ...rest }: RenderOptions = {}) => {
  const onChange = vi.fn();
  render(
    <Input.Table value={value} onChange={onChange} {...rest}>
      {COLUMNS}
    </Input.Table>,
  );
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

const addButton = (): HTMLElement => screen.getByRole("button", { name: "Add row" });

interface StatefulProps {
  initial: Input.TableCell[][];
  children: ReactElement<Input.TableColumnProps>[];
}

// Holds the value, so adds, removes, and Esc restores render as they would in a form.
const Stateful = ({ initial, children }: StatefulProps): ReactElement => {
  const [value, setValue] = useState(initial);
  return (
    <Input.Table value={value} onChange={setValue}>
      {children}
    </Input.Table>
  );
};

const removeButton = (row: number): HTMLElement =>
  screen.getAllByRole("row")[row + 1].querySelectorAll("button")[0];

const marked = (input: HTMLInputElement): boolean =>
  input.getAttribute("aria-invalid") === "true";

/** @returns the tooltip text a hover on the input's cell shows. */
const hoverTip = (input: HTMLInputElement): string | null => {
  vi.useFakeTimers();
  try {
    const td = input.closest("td");
    if (td == null) throw new Error("The input is not in a cell");
    td.getBoundingClientRect = () => new DOMRect(0, 0, 100, 20);
    fireEvent.pointerOver(td, { pointerType: "mouse" });
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    return screen.queryByRole("tooltip")?.textContent ?? null;
  } finally {
    vi.useRealTimers();
  }
};

describe("Input.Table", () => {
  it("should render a row per entry and a cell per column", () => {
    renderTable({
      value: [
        [1, 10],
        [2, 20],
      ],
    });
    // Header and add rows bracket the two entries.
    expect(screen.getAllByRole("row")).toHaveLength(4);
    expect(cell("Raw", "2").value).toBe("2");
    expect(cell("Scaled", "1").value).toBe("10");
  });

  it("should show only the add button when there are no rows", () => {
    renderTable({ value: [] });
    expect(screen.queryAllByRole("columnheader")).toHaveLength(0);
    expect(screen.getAllByRole("button")).toEqual([addButton()]);
  });

  it("should show the remove button of every row at all times", () => {
    renderTable();
    expect(removeButton(0).classList.contains("lyra--reveal")).toBe(false);
  });

  it("should name the row in the add button", () => {
    renderTable({ resourceName: "header" });
    expect(screen.getByRole("button", { name: "Add header" })).toBeTruthy();
  });

  it("should render every cell as a shadow input", () => {
    renderTable();
    const chassis = cell("Raw", "1").closest(".lyra-input");
    expect(chassis?.classList.contains("lyra--shadow")).toBe(true);
  });

  it("should render no row header until a labeler asks for one", () => {
    renderTable({
      value: [
        [1, 10],
        [2, 20],
      ],
    });
    expect(screen.queryAllByRole("rowheader")).toHaveLength(0);
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
      createRow: (value) => [Number(value.at(-1)?.[0] ?? 0) + 1, 0],
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

    it("should move Enter to the row below", () => {
      renderTable({ value: VALUE });
      const from = cell("Scaled", "1");
      from.focus();
      fireEvent.keyDown(from, { key: "Enter" });
      expect(document.activeElement).toBe(cell("Scaled", "2"));
    });

    it("should move Shift+Enter to the row above", () => {
      renderTable({ value: VALUE });
      const from = cell("Raw", "2");
      from.focus();
      fireEvent.keyDown(from, { key: "Enter", shiftKey: true });
      expect(document.activeElement).toBe(cell("Raw", "1"));
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

    it("should skip the row header column when rows have labels", () => {
      renderTable({ value: VALUE, rowLabel: (i) => `c${i}` });
      const from = cell("Raw", "c0");
      from.focus();
      fireEvent.keyDown(from, { key: "ArrowDown" });
      expect(document.activeElement).toBe(cell("Raw", "c1"));
    });

    it("should leave a key pressed outside a cell alone", () => {
      renderTable({ value: VALUE });
      expect(fireEvent.keyDown(addButton(), { key: "Enter" })).toBe(true);
      expect(fireEvent.keyDown(cell("Raw", "1"), { key: "Enter" })).toBe(false);
    });

    it("should hold focus on Enter in the last row", () => {
      renderTable({ value: VALUE });
      const from = cell("Scaled", "2");
      from.focus();
      fireEvent.keyDown(from, { key: "Enter" });
      expect(document.activeElement).toBe(from);
    });
  });

  describe("focus", () => {
    const VALUE = [
      [1, 10],
      [2, 20],
    ];

    it("should focus the first cell of an added row", () => {
      render(<Stateful initial={VALUE}>{COLUMNS}</Stateful>);
      fireEvent.click(addButton());
      expect(document.activeElement).toBe(cell("Raw", "3"));
    });

    it("should focus the row that takes a removed row's place", () => {
      render(<Stateful initial={VALUE}>{COLUMNS}</Stateful>);
      fireEvent.click(removeButton(0));
      expect(document.activeElement).toBe(cell("Raw", "1"));
      expect(cell("Raw", "1").value).toBe("2");
    });

    it("should focus the row above when the last row is removed", () => {
      render(<Stateful initial={VALUE}>{COLUMNS}</Stateful>);
      fireEvent.click(removeButton(1));
      expect(document.activeElement).toBe(cell("Raw", "1"));
    });

    it("should focus the add button when the only row is removed", () => {
      render(<Stateful initial={[[1, 10]]}>{COLUMNS}</Stateful>);
      fireEvent.click(removeButton(0));
      expect(document.activeElement).toBe(addButton());
    });
  });

  describe("escape", () => {
    const TEXT_COLUMNS = [
      <Input.TableColumn key="name" name="Name" type="string" />,
      <Input.TableColumn key="value" name="Value" type="string" />,
    ];

    it("should put back the value a cell held when it took focus", () => {
      render(<Stateful initial={[["a", "b"]]}>{TEXT_COLUMNS}</Stateful>);
      const input = cell("Name", "1");
      input.focus();
      fireEvent.change(input, { target: { value: "changed" } });
      fireEvent.keyDown(input, { key: "Escape" });
      expect(input.value).toBe("a");
      expect(document.activeElement).toBe(input);
    });

    it("should leave the cell on a second Esc", () => {
      render(<Stateful initial={[["a", "b"]]}>{TEXT_COLUMNS}</Stateful>);
      const input = cell("Name", "1");
      input.focus();
      fireEvent.change(input, { target: { value: "changed" } });
      fireEvent.keyDown(input, { key: "Escape" });
      fireEvent.keyDown(input, { key: "Escape" });
      expect(document.activeElement).not.toBe(input);
    });

    it("should put back a cell pasted into after an earlier Esc", () => {
      render(<Stateful initial={[["a", "b"]]}>{TEXT_COLUMNS}</Stateful>);
      const input = cell("Name", "1");
      input.focus();
      fireEvent.change(input, { target: { value: "changed" } });
      fireEvent.keyDown(input, { key: "Escape" });
      fireEvent.input(input, { target: { value: "pasted" } });
      fireEvent.keyDown(input, { key: "Escape" });
      expect(input.value).toBe("a");
      expect(document.activeElement).toBe(input);
    });

    it("should leave an unchanged cell on the first Esc", () => {
      render(<Stateful initial={[["a", "b"]]}>{TEXT_COLUMNS}</Stateful>);
      const input = cell("Name", "1");
      input.focus();
      fireEvent.keyDown(input, { key: "Escape" });
      expect(document.activeElement).not.toBe(input);
    });

    it("should discard a typed number and keep focus", () => {
      render(<Stateful initial={[[1, 10]]}>{COLUMNS}</Stateful>);
      const input = cell("Raw", "1");
      input.focus();
      fireEvent.change(input, { target: { value: "15" } });
      fireEvent.keyDown(input, { key: "Escape" });
      expect(input.value).toBe("1");
      expect(document.activeElement).toBe(input);
    });

    it("should keep an Esc that puts a cell back from reaching a parent", () => {
      const onParentKeyDown = vi.fn();
      render(
        <div onKeyDown={onParentKeyDown}>
          <Stateful initial={[["a", "b"]]}>{TEXT_COLUMNS}</Stateful>
        </div>,
      );
      const input = cell("Name", "1");
      input.focus();
      fireEvent.change(input, { target: { value: "changed" } });
      fireEvent.keyDown(input, { key: "Escape" });
      expect(onParentKeyDown).not.toHaveBeenCalled();
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
          value={[
            [1, 10],
            [2, 20],
            [3, 30],
          ]}
          onChange={onChange}
        >
          {COLUMNS}
        </Input.Table>,
      );
      fireEvent.focus(cell("Raw", "3"));
      rerender(
        <Input.Table value={[[1, 10]]} onChange={onChange}>
          {COLUMNS}
        </Input.Table>,
      );
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

    it("should keep the values that parse and mark the cells that do not", () => {
      render(<Stateful initial={[[1, 10]]}>{COLUMNS}</Stateful>);
      paste(cell("Raw", "1"), "1\t10\n2\tnot a number");
      expect(cell("Raw", "2").value).toBe("2");
      expect(cell("Scaled", "2").value).toBe("0");
      expect(marked(cell("Scaled", "2"))).toBe(true);
      expect(marked(cell("Raw", "2"))).toBe(false);
      expect(marked(cell("Scaled", "1"))).toBe(false);
    });

    it("should name the text that did not parse in the cell's tooltip", () => {
      render(<Stateful initial={[[1, 10]]}>{COLUMNS}</Stateful>);
      paste(cell("Raw", "1"), "5\tabc");
      expect(hoverTip(cell("Scaled", "1"))).toBe('"abc" is not a number');
    });

    it("should keep a first row in which a number parses", () => {
      render(<Stateful initial={[[1, 10]]}>{COLUMNS}</Stateful>);
      paste(cell("Raw", "1"), "5\tabc");
      expect(cell("Raw", "1").value).toBe("5");
      expect(cell("Scaled", "1").value).toBe("10");
      expect(marked(cell("Scaled", "1"))).toBe(true);
    });

    it("should leave a lone row in which no number parses to the cell", () => {
      const onChange = renderTable();
      paste(cell("Raw", "1"), "abc\tdef");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should clear a mark when its cell is edited", () => {
      render(<Stateful initial={[[1, 10]]}>{COLUMNS}</Stateful>);
      paste(cell("Raw", "1"), "5\tabc");
      commit(cell("Scaled", "1"), "11");
      expect(marked(cell("Scaled", "1"))).toBe(false);
    });

    it("should clear a mark when an edit commits the value its cell kept", () => {
      render(<Stateful initial={[[1, 10]]}>{COLUMNS}</Stateful>);
      paste(cell("Raw", "1"), "5\tabc");
      const input = cell("Scaled", "1");
      fireEvent.change(input, { target: { value: "" } });
      commit(input, "10");
      expect(marked(cell("Scaled", "1"))).toBe(false);
    });

    it("should clear a mark when a later paste reads its cell", () => {
      render(<Stateful initial={[[1, 10]]}>{COLUMNS}</Stateful>);
      paste(cell("Raw", "1"), "5\tabc");
      paste(cell("Raw", "1"), "6\t10");
      expect(marked(cell("Scaled", "1"))).toBe(false);
    });

    it("should move a mark up with its row when a row above is removed", () => {
      render(
        <Stateful
          initial={[
            [1, 10],
            [2, 20],
            [3, 30],
          ]}
        >
          {COLUMNS}
        </Stateful>,
      );
      paste(cell("Raw", "3"), "4\tabc\n5\t50");
      fireEvent.click(removeButton(0));
      expect(cell("Scaled", "2").value).toBe("30");
      expect(marked(cell("Scaled", "2"))).toBe(true);
      expect(marked(cell("Scaled", "1"))).toBe(false);
    });

    it("should drop a mark for good once the caller replaces its value", () => {
      const onChange = vi.fn();
      const { rerender } = render(
        <Input.Table value={[[1, 10]]} onChange={onChange}>
          {COLUMNS}
        </Input.Table>,
      );
      paste(cell("Raw", "1"), "5\tabc");
      expect(marked(cell("Scaled", "1"))).toBe(true);
      rerender(
        <Input.Table value={[[5, 20]]} onChange={onChange}>
          {COLUMNS}
        </Input.Table>,
      );
      expect(marked(cell("Scaled", "1"))).toBe(false);
      rerender(
        <Input.Table value={[[5, 10]]} onChange={onChange}>
          {COLUMNS}
        </Input.Table>,
      );
      expect(marked(cell("Scaled", "1"))).toBe(false);
    });
  });

  describe("status", () => {
    const DUPLICATE: Status.Crude = { variant: "error", message: "Duplicate value" };
    const markScaled = ({ x, y }: xy.XY): Status.Crude | undefined =>
      x === 1 && y === 0 ? DUPLICATE : undefined;

    it("should mark the cells that the caller gives an error", () => {
      renderTable({ cellStatus: markScaled });
      expect(marked(cell("Scaled", "1"))).toBe(true);
      expect(marked(cell("Raw", "1"))).toBe(false);
    });

    it("should show a cell's message in its tooltip", () => {
      renderTable({ cellStatus: markScaled });
      expect(hoverTip(cell("Scaled", "1"))).toBe("Duplicate value");
    });

    it("should show no tooltip on a cell without a status", () => {
      renderTable({ cellStatus: markScaled });
      expect(hoverTip(cell("Raw", "1"))).toBeNull();
    });

    it("should keep focus in a cell whose status comes and goes", () => {
      const onChange = vi.fn();
      const { rerender } = render(
        <Input.Table value={[[1, 10]]} onChange={onChange}>
          {COLUMNS}
        </Input.Table>,
      );
      const input = cell("Scaled", "1");
      input.focus();
      rerender(
        <Input.Table value={[[1, 10]]} onChange={onChange} cellStatus={markScaled}>
          {COLUMNS}
        </Input.Table>,
      );
      expect(document.activeElement).toBe(input);
      rerender(
        <Input.Table value={[[1, 10]]} onChange={onChange}>
          {COLUMNS}
        </Input.Table>,
      );
      expect(document.activeElement).toBe(input);
    });
  });

  describe("columns", () => {
    const renderMixed = (value: Input.TableCell[][] = [["on", 1]]) => {
      const onChange = vi.fn();
      render(
        <Input.Table value={value} onChange={onChange}>
          <Input.TableColumn name="Label" type="string" />
          <Input.TableColumn name="Value" />
        </Input.Table>,
      );
      return onChange;
    };

    it("should default a text column to a text input", () => {
      renderMixed();
      expect(cell("Label", "1").getAttribute("type")).not.toBe("number");
      expect(cell("Label", "1").value).toBe("on");
    });

    it("should render the cell a column's children build", () => {
      const onChange = vi.fn();
      render(
        <Input.Table value={[["on", 1]]} onChange={onChange}>
          <Input.TableColumn name="Label" type="string">
            {(p) => <Input.Text {...p} placeholder="the label" />}
          </Input.TableColumn>
          <Input.TableColumn name="Value" />
        </Input.Table>,
      );
      expect(screen.getByPlaceholderText("the label")).toBe(cell("Label", "1"));
    });

    it("should commit a text cell's edit into its column", () => {
      const onChange = renderMixed();
      commit(cell("Label", "1"), "off");
      expect(onChange).toHaveBeenCalledWith([["off", 1]]);
    });

    it("should append a row of the column default values", () => {
      const onChange = renderMixed();
      fireEvent.click(addButton());
      expect(onChange).toHaveBeenCalledWith([
        ["on", 1],
        ["", 0],
      ]);
    });

    it("should paste text into a column that accepts it", () => {
      const onChange = renderMixed();
      paste(cell("Label", "1"), "off\t2\nidle\t3");
      expect(onChange).toHaveBeenCalledWith([
        ["off", 2],
        ["idle", 3],
      ]);
    });

    it("should mark the number cells of a block that do not parse", () => {
      const onChange = renderMixed();
      paste(cell("Label", "1"), "off\t2\nidle\tnot a number");
      expect(onChange).toHaveBeenCalledWith([
        ["off", 2],
        ["idle", 0],
      ]);
    });

    it("should drop a heading row a text column would otherwise accept", () => {
      const onChange = renderMixed();
      paste(cell("Label", "1"), "Label\tValue\nON\t1\nOFF\t2");
      expect(onChange).toHaveBeenCalledWith([
        ["ON", 1],
        ["OFF", 2],
      ]);
    });

    it("should move focus into a text cell", () => {
      renderMixed([
        ["on", 1],
        ["off", 2],
      ]);
      const from = cell("Label", "1");
      from.focus();
      fireEvent.keyDown(from, { key: "ArrowDown" });
      expect(document.activeElement).toBe(cell("Label", "2"));
    });
  });
});
