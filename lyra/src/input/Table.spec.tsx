// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { xy } from "@synnaxlabs/x";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { createPortal } from "react-dom";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Input } from "@/input";
import { Select } from "@/select";
import type { Status } from "@/status";

const COLUMNS = [
  <Input.TableColumn key="raw" name="Raw" />,
  <Input.TableColumn key="scaled" name="Scaled" />,
];

// Reads only 0x-prefixed text, so a decimal paste is refused.
const parseHex = (text: string): number | null =>
  text.startsWith("0x") ? Number.parseInt(text.slice(2), 16) : null;

const HEX_COLUMNS = [
  <Input.TableColumn key="hex" name="Hex" parse={parseHex} />,
  <Input.TableColumn key="raw" name="Raw" />,
];

// A field with a picker beside it, as a color cell has.
const UNIT_COLUMNS = [
  <Input.TableColumn key="raw" name="Raw" />,
  <Input.TableColumn key="unit" name="Unit" type="string">
    {({ value, onChange, "aria-label": ariaLabel }) => (
      <>
        <Input.Text value={value} onChange={onChange} aria-label={ariaLabel} />
        <Select.Simple<string>
          value={value}
          onChange={onChange}
          resourceName="unit"
          triggerProps={{ "aria-label": `Pick ${ariaLabel}` }}
        >
          <Select.Item itemKey="volts">Volts</Select.Item>
          <Select.Item itemKey="amps">Amps</Select.Item>
        </Select.Simple>
      </>
    )}
  </Input.TableColumn>,
];

interface RenderOptions {
  value?: Input.TableCell[][];
  preview?: boolean;
  createRow?: (value: Input.TableCell[][]) => Input.TableCell[];
  rowLabel?: (index: number) => string;
  resourceName?: string;
  cellStatus?: (at: xy.XY) => Status.Crude | undefined;
  onRemove?: (index: number) => void;
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
    const frame = input.closest<HTMLElement>(".lyra-input");
    if (frame == null) throw new Error("The input has no frame");
    frame.getBoundingClientRect = () => new DOMRect(0, 0, 100, 20);
    fireEvent.pointerOver(frame, { pointerType: "mouse" });
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
    expect(chassis?.classList.contains("lyra-btn--shadow")).toBe(true);
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

  it("should pass the removed row's index to onRemove in place of onChange", () => {
    const onRemove = vi.fn();
    const onChange = renderTable({
      value: [
        [1, 10],
        [2, 20],
      ],
      onRemove,
    });
    fireEvent.click(removeButton(1));
    expect(onRemove).toHaveBeenCalledExactlyOnceWith(1);
    expect(onChange).not.toHaveBeenCalled();
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

    it("should keep each cell of a block paste on Esc", () => {
      render(<Stateful initial={[["a", "b"]]}>{TEXT_COLUMNS}</Stateful>);
      const input = cell("Name", "1");
      input.focus();
      paste(input, "x\ty\nz\tw");
      fireEvent.keyDown(input, { key: "Escape" });
      expect(cell("Name", "1").value).toBe("x");
      expect(cell("Name", "2").value).toBe("z");
      expect(document.activeElement).not.toBe(input);
    });

    it("should put back the pasted value of a cell typed into after a block paste", () => {
      render(<Stateful initial={[["a", "b"]]}>{TEXT_COLUMNS}</Stateful>);
      const input = cell("Name", "1");
      input.focus();
      paste(input, "x\ty\nz\tw");
      fireEvent.change(input, { target: { value: "typed" } });
      fireEvent.keyDown(input, { key: "Escape" });
      expect(input.value).toBe("x");
      expect(document.activeElement).toBe(input);
    });

    it("should leave an unchanged cell on the first Esc", () => {
      render(<Stateful initial={[["a", "b"]]}>{TEXT_COLUMNS}</Stateful>);
      const input = cell("Name", "1");
      input.focus();
      fireEvent.keyDown(input, { key: "Escape" });
      expect(document.activeElement).not.toBe(input);
    });

    it("should commit nothing on two Escs in a cell that commits on blur", () => {
      const onChange = vi.fn();
      render(
        <Input.Table value={[["a"]]} onChange={onChange}>
          {[
            <Input.TableColumn key="name" name="Name" type="string">
              {(props) => <Input.Text {...props} onlyChangeOnBlur />}
            </Input.TableColumn>,
          ]}
        </Input.Table>,
      );
      const input = cell("Name", "1");
      act(() => input.focus());
      fireEvent.change(input, { target: { value: "changed" } });
      fireEvent.keyDown(input, { key: "Escape" });
      expect(input.value).toBe("a");
      expect(document.activeElement).toBe(input);
      fireEvent.keyDown(input, { key: "Escape" });
      expect(document.activeElement).not.toBe(input);
      expect(onChange).not.toHaveBeenCalled();
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

  describe("row key", () => {
    // Holds a flag that a click sets, as an open dialog does.
    const Toggle = ({ label }: { label: string }): ReactElement => {
      const [on, setOn] = useState(false);
      return (
        <button aria-label={label} onClick={() => setOn(true)}>
          {on ? "on" : "off"}
        </button>
      );
    };
    const TOGGLE_COLUMNS = [
      <Input.TableColumn key="name" name="Name" type="string">
        {({ value }) => <Toggle label={value} />}
      </Input.TableColumn>,
    ];

    it("should keep the state of a cell with its row when a row above leaves", () => {
      const { rerender } = render(
        <Input.Table value={[["a"], ["b"]]} onChange={vi.fn()} rowKey={(i) => "ab"[i]}>
          {TOGGLE_COLUMNS}
        </Input.Table>,
      );
      fireEvent.click(screen.getByLabelText("b"));
      rerender(
        <Input.Table value={[["b"]]} onChange={vi.fn()} rowKey={() => "b"}>
          {TOGGLE_COLUMNS}
        </Input.Table>,
      );
      expect(screen.getByLabelText("b").textContent).toBe("on");
    });
  });

  describe("portal", () => {
    // A cell that opens a portal, as an options dialog does.
    const PORTAL_COLUMNS = [
      <Input.TableColumn key="name" name="Name" type="string">
        {(props) => (
          <>
            <Input.Text {...props} />
            {createPortal(<input aria-label="Portal" />, document.body)}
          </>
        )}
      </Input.TableColumn>,
    ];

    it("should ignore a paste into a portal that a cell opens", () => {
      const onChange = vi.fn();
      render(
        <Input.Table value={[["a"]]} onChange={onChange}>
          {PORTAL_COLUMNS}
        </Input.Table>,
      );
      paste(screen.getByLabelText("Portal"), "x\ny\nz");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should let an Esc in a portal that a cell opens reach a parent", () => {
      const onParentKeyDown = vi.fn();
      render(
        <div onKeyDown={onParentKeyDown}>
          <Stateful initial={[["a"]]}>{PORTAL_COLUMNS}</Stateful>
        </div>,
      );
      const portal = screen.getByLabelText("Portal");
      fireEvent.focus(portal);
      fireEvent.change(portal, { target: { value: "typed" } });
      fireEvent.keyDown(portal, { key: "Escape" });
      expect(onParentKeyDown).toHaveBeenCalled();
    });
  });

  describe("dialog", () => {
    /** Opens the unit select of the first row and returns its search. */
    const openUnitSearch = (onChange = vi.fn()): HTMLInputElement => {
      render(
        <Input.Table
          value={[
            [1, "volts"],
            [2, "amps"],
          ]}
          onChange={onChange}
        >
          {UNIT_COLUMNS}
        </Input.Table>,
      );
      fireEvent.click(screen.getByLabelText("Pick Unit 1"));
      const search = screen.getByPlaceholderText<HTMLInputElement>("Search units...");
      search.focus();
      return search;
    };

    it("should leave Enter in a dialog that a cell opens to the dialog", () => {
      const search = openUnitSearch();
      fireEvent.keyDown(search, { key: "Enter" });
      expect(document.activeElement).toBe(search);
    });

    it("should leave Escape in a dialog that a cell opens to the dialog", () => {
      render(<Stateful initial={[[1, "volts"]]}>{UNIT_COLUMNS}</Stateful>);
      fireEvent.click(screen.getByLabelText("Pick Unit 1"));
      const search = screen.getByPlaceholderText<HTMLInputElement>("Search units...");
      act(() => search.focus());
      fireEvent.change(search, { target: { value: "am" } });
      fireEvent.keyDown(search, { key: "Escape" });
      expect(search.closest(".lyra-select__dialog")?.classList).toContain(
        "lyra--hidden",
      );
      const input = cell("Unit", "1");
      act(() => input.focus());
      fireEvent.change(input, { target: { value: "changed" } });
      fireEvent.keyDown(input, { key: "Escape" });
      expect(input.value).toBe("volts");
    });

    it("should leave the cell on Esc after Escape closes the cell's dialog", () => {
      render(<Stateful initial={[[1, "volts"]]}>{UNIT_COLUMNS}</Stateful>);
      fireEvent.click(screen.getByLabelText("Pick Unit 1"));
      const search = screen.getByPlaceholderText<HTMLInputElement>("Search units...");
      act(() => search.focus());
      fireEvent.change(search, { target: { value: "am" } });
      fireEvent.keyDown(search, { key: "Escape" });
      const input = cell("Unit", "1");
      act(() => input.focus());
      fireEvent.keyDown(input, { key: "Escape" });
      expect(document.activeElement).not.toBe(input);
    });

    it("should leave a paste in a dialog that a cell opens to the dialog", () => {
      const onChange = vi.fn();
      const search = openUnitSearch(onChange);
      paste(search, "volts\namps");
      expect(onChange).not.toHaveBeenCalled();
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

    it("should build the rows that a paste adds with createRow", () => {
      const onChange = renderTable({ createRow: () => [7, 70] });
      paste(cell("Raw", "1"), "1\t10\n2");
      expect(onChange).toHaveBeenCalledWith([
        [1, 10],
        [2, 70],
      ]);
    });

    it("should pass the rows a paste has added so far to createRow", () => {
      const onChange = renderTable({
        createRow: (value) => [Number(value.at(-1)?.[0] ?? 0) + 1, -1],
      });
      paste(cell("Scaled", "1"), "11\n21\n31");
      expect(onChange).toHaveBeenCalledWith([
        [1, 11],
        [2, 21],
        [3, 31],
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

    it("should name the text and the column of a refused cell in its tooltip", () => {
      render(<Stateful initial={[[1, 10]]}>{COLUMNS}</Stateful>);
      paste(cell("Raw", "1"), "5\tabc");
      expect(hoverTip(cell("Scaled", "1"))).toBe(
        '"abc" is not a valid value for Scaled',
      );
    });

    it("should call an unnamed column 'this column' in a refused cell's tooltip", () => {
      render(
        <Stateful initial={[[1]]}>{[<Input.TableColumn key="value" />]}</Stateful>,
      );
      paste(screen.getByLabelText("1"), "1\nabc");
      expect(hoverTip(screen.getByLabelText("2"))).toBe(
        '"abc" is not a valid value for this column',
      );
    });

    it("should read a pasted cell with its column's parse", () => {
      const onChange = vi.fn();
      render(
        <Input.Table value={[[0, 0]]} onChange={onChange}>
          {HEX_COLUMNS}
        </Input.Table>,
      );
      paste(cell("Hex", "1"), "0x10\t5");
      expect(onChange).toHaveBeenCalledWith([[16, 5]]);
    });

    it("should name the column whose parse refused a pasted cell", () => {
      render(<Stateful initial={[[0, 0]]}>{HEX_COLUMNS}</Stateful>);
      paste(cell("Hex", "1"), "16\t5");
      expect(cell("Hex", "1").value).toBe("0");
      expect(hoverTip(cell("Hex", "1"))).toBe('"16" is not a valid value for Hex');
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

  describe("in an item", () => {
    it("should not mark its cells with the item's status", () => {
      render(
        <Input.Item label="Coefficients" status="error" helpText="Too few">
          <Input.Table value={[[1, 10]]} onChange={vi.fn()}>
            {COLUMNS}
          </Input.Table>
        </Input.Item>,
      );
      expect(marked(cell("Raw", "1"))).toBe(false);
      expect(hoverTip(cell("Raw", "1"))).toBeNull();
    });
  });

  describe("columns", () => {
    const renderWidths = () =>
      render(
        <Input.Table value={[[1, 2]]} onChange={vi.fn()}>
          <Input.TableColumn name="Raw" width={10} />
          <Input.TableColumn name="Scaled" />
        </Input.Table>,
      );

    it("should give a column the width it declares", () => {
      const { container } = renderWidths();
      const widths = Array.from(container.querySelectorAll("col")).map(
        (col) => col.style.width,
      );
      expect(widths).toEqual(["10rem", "", ""]);
    });

    it("should size the table to fit every column", () => {
      const { container } = renderWidths();
      const table = container.querySelector("table");
      expect(table?.style.getPropertyValue("--pluto-input-table-columns-width")).toBe(
        "26rem",
      );
    });

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

    it("should keep the first pasted row of a table with only text columns", () => {
      const onChange = vi.fn();
      render(
        <Input.Table value={[["on"]]} onChange={onChange}>
          <Input.TableColumn name="Label" type="string" />
        </Input.Table>,
      );
      paste(cell("Label", "1"), "Label\nON");
      expect(onChange).toHaveBeenCalledWith([["Label"], ["ON"]]);
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

  describe("boolean columns", () => {
    const renderFlags = (value: Input.TableCell[][] = [["pump", true]]) => {
      const onChange = vi.fn();
      render(
        <Input.Table value={value} onChange={onChange}>
          <Input.TableColumn name="Name" type="string" />
          <Input.TableColumn name="Enabled" type="boolean" />
        </Input.Table>,
      );
      return onChange;
    };

    it("should default a boolean column to a checkbox", () => {
      renderFlags();
      const box = cell("Enabled", "1");
      expect(box.checked).toBe(true);
      expect(box.classList).toContain(CSS.BE("input", "checkbox", "input"));
    });

    it("should show a read-only boolean cell as a disabled checkbox", () => {
      render(
        <Input.Table value={[["pump", true]]} onChange={vi.fn()} preview>
          <Input.TableColumn name="Name" type="string" />
          <Input.TableColumn name="Enabled" type="boolean" />
        </Input.Table>,
      );
      const box = cell("Enabled", "1");
      expect(box.checked).toBe(true);
      expect(box.disabled).toBe(true);
    });

    it("should commit a toggled checkbox into its column", () => {
      const onChange = renderFlags();
      fireEvent.click(cell("Enabled", "1"));
      expect(onChange).toHaveBeenCalledWith([["pump", false]]);
    });

    it("should append a row whose boolean cell is false", () => {
      const onChange = renderFlags();
      fireEvent.click(addButton());
      expect(onChange).toHaveBeenCalledWith([
        ["pump", true],
        ["", false],
      ]);
    });

    it.each([
      ["true", true],
      ["FALSE", false],
      ["1", true],
      ["0", false],
    ])("should read a pasted %s", (text, expected) => {
      const onChange = renderFlags();
      paste(cell("Name", "1"), `valve\t${text}\nvent\ttrue`);
      expect(onChange).toHaveBeenCalledWith([
        ["valve", expected],
        ["vent", true],
      ]);
    });

    it("should read a single value pasted into a checkbox", () => {
      const onChange = renderFlags();
      paste(cell("Enabled", "1"), "0");
      expect(onChange).toHaveBeenCalledWith([["pump", false]]);
    });

    it("should mark a checkbox whose single pasted value does not parse", () => {
      render(
        <Stateful initial={[["pump", true]]}>
          {[
            <Input.TableColumn key="name" name="Name" type="string" />,
            <Input.TableColumn key="enabled" name="Enabled" type="boolean" />,
          ]}
        </Stateful>,
      );
      paste(cell("Enabled", "1"), "maybe");
      expect(cell("Enabled", "1").checked).toBe(true);
      expect(marked(cell("Enabled", "1"))).toBe(true);
    });

    it("should drop a heading row whose boolean cells do not parse", () => {
      const onChange = renderFlags();
      paste(cell("Name", "1"), "Name\tEnabled\nvalve\tfalse");
      expect(onChange).toHaveBeenCalledWith([["valve", false]]);
    });

    it("should refuse pasted text that names an object property", () => {
      const onChange = renderFlags([
        ["pump", true],
        ["fan", false],
      ]);
      paste(cell("Name", "1"), "valve\ttrue\nvent\tconstructor");
      expect(onChange).toHaveBeenCalledWith([
        ["valve", true],
        ["vent", false],
      ]);
      expect(marked(cell("Enabled", "2"))).toBe(true);
    });

    it("should keep and mark a boolean cell whose pasted text does not parse", () => {
      const onChange = renderFlags([
        ["pump", true],
        ["fan", false],
      ]);
      paste(cell("Name", "1"), "valve\tfalse\nvent\tmaybe");
      expect(onChange).toHaveBeenCalledWith([
        ["valve", false],
        ["vent", false],
      ]);
      expect(marked(cell("Enabled", "2"))).toBe(true);
    });
  });

  describe("custom columns", () => {
    interface Point {
      x: number;
      y: number;
    }

    const parsePoint = (text: string): Point | null => {
      const [x, y] = text.split(";").map(Number);
      return isFinite(x) && isFinite(y) ? { x, y } : null;
    };

    const renderPoints = (parse?: (text: string) => Point | null) => {
      const onChange = vi.fn();
      render(
        <Input.Table value={[["origin", { x: 0, y: 0 }]]} onChange={onChange}>
          <Input.TableColumn name="Name" type="string" />
          <Input.TableColumn<Point>
            name="Point"
            type="custom"
            empty={{ x: 1, y: 1 }}
            parse={parse}
          >
            {({ value, onChange, "aria-label": label }) => (
              <button
                aria-label={label}
                onClick={() => onChange({ x: value.x + 1, y: value.y })}
              >
                {`${value.x};${value.y}`}
              </button>
            )}
          </Input.TableColumn>
        </Input.Table>,
      );
      return onChange;
    };

    const point = (): HTMLElement => screen.getByRole("button", { name: "Point 1" });

    it("should render the cell a custom column builds from the cell's value", () => {
      renderPoints();
      expect(point().textContent).toBe("0;0");
    });

    it("should commit a custom cell's change into its column", () => {
      const onChange = renderPoints();
      fireEvent.click(point());
      expect(onChange).toHaveBeenCalledWith([["origin", { x: 1, y: 0 }]]);
    });

    it("should append a row holding the custom column's empty value", () => {
      const onChange = renderPoints();
      fireEvent.click(addButton());
      expect(onChange).toHaveBeenCalledWith([
        ["origin", { x: 0, y: 0 }],
        ["", { x: 1, y: 1 }],
      ]);
    });

    it("should refuse a paste into a custom column without a parse", () => {
      const onChange = renderPoints();
      paste(cell("Name", "1"), "a\t2;3\nb\t4;5");
      expect(onChange).toHaveBeenCalledWith([
        ["a", { x: 0, y: 0 }],
        ["b", { x: 1, y: 1 }],
      ]);
    });

    it("should read a paste with a custom column's parse", () => {
      const onChange = renderPoints(parsePoint);
      paste(cell("Name", "1"), "a\t2;3");
      expect(onChange).toHaveBeenCalledWith([["a", { x: 2, y: 3 }]]);
    });

    it("should write an undefined value that a custom column's parse reads", () => {
      const onChange = vi.fn();
      render(
        <Input.Table value={[["origin", "red"]]} onChange={onChange}>
          <Input.TableColumn name="Name" type="string" />
          <Input.TableColumn<string | undefined>
            name="Tint"
            type="custom"
            empty={undefined}
            parse={(text) => (text === "" ? undefined : text)}
          >
            {({ value, "aria-label": label }) => (
              <button aria-label={label}>{value ?? "None"}</button>
            )}
          </Input.TableColumn>
        </Input.Table>,
      );
      paste(cell("Name", "1"), "origin\t\nnext\tblue");
      expect(onChange).toHaveBeenCalledWith([
        ["origin", undefined],
        ["next", "blue"],
      ]);
    });
  });

  describe("select cells", () => {
    const ORDERS = ["big", "little"];

    const renderOrders = (parse?: (text: string) => string | null) => {
      const onChange = vi.fn();
      render(
        <Input.Table value={[["big"]]} onChange={onChange}>
          <Input.TableColumn name="Byte order" type="string" parse={parse}>
            {({ value, "aria-label": label }) => (
              <button aria-label={label}>{value}</button>
            )}
          </Input.TableColumn>
        </Input.Table>,
      );
      const order = screen.getByRole("button", { name: "Byte order 1" });
      return { onChange, order };
    };

    const pasteOn = (el: HTMLElement, text: string): void => {
      fireEvent.focus(el);
      fireEvent.paste(el, { clipboardData: { getData: () => text } });
    };

    it("should leave one value pasted on a select whose column has no parse", () => {
      const { onChange, order } = renderOrders();
      pasteOn(order, "foo");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should read a single value pasted on a select with its column's parse", () => {
      const { onChange, order } = renderOrders((text) =>
        ORDERS.includes(text) ? text : null,
      );
      pasteOn(order, "little");
      expect(onChange).toHaveBeenCalledWith([["little"]]);
    });
  });
});
