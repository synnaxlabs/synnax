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

describe("Input.Table", () => {
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

  const firePaste = (input: HTMLInputElement, text: string): void => {
    const clipboardData = new DataTransfer();
    clipboardData.setData("text/plain", text);
    fireEvent(
      input,
      new ClipboardEvent("paste", { clipboardData, bubbles: true, cancelable: true }),
    );
  };

  const paste = (input: HTMLInputElement, text: string): void => {
    fireEvent.focus(input);
    firePaste(input, text);
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
      firePaste(cell("Raw", "1"), "9\t90\n8\t80");
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
