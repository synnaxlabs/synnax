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
import { type ReactElement, type ReactNode, useState } from "react";
import { createPortal } from "react-dom";
import { describe, expect, it, vi } from "vitest";
import { userEvent } from "vitest/browser";

import { CSS } from "@/css";
import { Form } from "@/form";
import { Haul } from "@/haul";
import { Input } from "@/input";
import { Select } from "@/select";
import type { Status } from "@/status";
import { computed, renderThemed, resolve } from "@/styletest";

type ColumnType = "string" | "number";

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

  it("should center the drag icon in its handle", async () => {
    const { container } = await renderThemed(
      <Input.Table value={[[1]]} onChange={() => {}} onMove={() => {}} rowKey={String}>
        <Input.TableColumn />
      </Input.Table>,
    );
    const handle = container.querySelector(`.${CSS.BE("input", "table-handle")}`);
    const icon = handle?.querySelector("svg");
    if (handle == null || icon == null) throw new Error("The row has no drag handle");
    const center = (el: Element): number => {
      const { left, width } = el.getBoundingClientRect();
      return left + width / 2;
    };
    expect(center(icon)).toBeCloseTo(center(handle), 0);
  });

  it("should draw the drag icon in the color of the remove button", async () => {
    const { container } = await renderThemed(
      <Input.Table value={[[1]]} onChange={() => {}} onMove={() => {}} rowKey={String}>
        <Input.TableColumn />
      </Input.Table>,
    );
    const row = container.querySelector("tbody tr");
    const handle = row?.querySelector(`.${CSS.BE("input", "table-handle")}`);
    const remove = row?.lastElementChild?.querySelector("button");
    if (handle == null || remove == null) throw new Error("The row has no controls");
    expect(computed(handle, "color")).toBe(computed(remove, "color"));
  });

  it("should center a boolean cell's checkbox in its cell", async () => {
    const { container } = await renderThemed(
      <Input.Table value={[[true]]} onChange={() => {}}>
        <Input.TableColumn name="Enabled" type="boolean" width={20} />
      </Input.Table>,
    );
    const indicator = container.querySelector(
      `.${CSS.BE("input", "checkbox", "indicator")}`,
    );
    const cell = indicator?.closest("td");
    if (indicator == null || cell == null) throw new Error("The cell has no checkbox");
    const center = (el: Element): number => {
      const { left, width } = el.getBoundingClientRect();
      return left + width / 2;
    };
    expect(center(indicator)).toBeCloseTo(center(cell), 0);
  });

  it("should center a boolean cell's checkbox in a labeled field", async () => {
    const { container } = await renderThemed(
      <Input.Item label="Flags">
        <Input.Table value={[[true]]} onChange={() => {}}>
          <Input.TableColumn name="Enabled" type="boolean" width={20} />
        </Input.Table>
      </Input.Item>,
    );
    const indicator = container.querySelector(
      `.${CSS.BE("input", "checkbox", "indicator")}`,
    );
    const cell = indicator?.closest("td");
    if (indicator == null || cell == null) throw new Error("The cell has no checkbox");
    const center = (el: Element): number => {
      const { left, width } = el.getBoundingClientRect();
      return left + width / 2;
    };
    expect(center(indicator)).toBeCloseTo(center(cell), 0);
  });

  it("should center a switch in its cell inside a form section", async () => {
    const { container } = await renderThemed(
      <Form.Sections>
        <Form.Section>
          <Input.Table value={[[true]]} onChange={() => {}}>
            <Input.TableColumn name="Enabled" type="boolean" width={20}>
              {({ value, onChange, "aria-label": label }) => (
                <Input.Switch value={value} onChange={onChange} aria-label={label} />
              )}
            </Input.TableColumn>
          </Input.Table>
        </Form.Section>
      </Form.Sections>,
    );
    const indicator = container.querySelector(
      `.${CSS.BE("input", "switch", "indicator")}`,
    );
    const cell = indicator?.closest("td");
    if (indicator == null || cell == null) throw new Error("The cell has no switch");
    const center = (el: Element): number => {
      const { left, width } = el.getBoundingClientRect();
      return left + width / 2;
    };
    expect(center(indicator)).toBeCloseTo(center(cell), 0);
  });

  it("should inset read-only text as far as editable text", async () => {
    const { container } = await renderThemed(
      <>
        <Input.Table value={[["pump"]]} onChange={() => {}}>
          <Input.TableColumn name="Name" type="string" />
        </Input.Table>
        <Input.Table value={[["pump"]]} onChange={() => {}} preview>
          <Input.TableColumn name="Name" type="string" />
        </Input.Table>
      </>,
    );
    const [editable, readOnly] = Array.from(container.querySelectorAll("table"));
    const textLeft = (el: Element | null): number => {
      if (el == null) throw new Error("The cell has no text");
      const range = document.createRange();
      range.selectNodeContents(el);
      return range.getBoundingClientRect().left;
    };
    const input = editable.querySelector("tbody input");
    if (input == null) throw new Error("The editable cell has no input");
    const inset =
      input.getBoundingClientRect().left + parseFloat(computed(input, "padding-left"));
    expect(
      textLeft(readOnly.querySelector(`.${CSS.BE("input", "preview-value")}`)),
    ).toBeCloseTo(inset, 0);
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

  it("should fill the height of a cell whose control starts with no text", async () => {
    const { container } = await renderThemed(
      <Input.Table value={[[""]]} onChange={() => {}}>
        <Input.TableColumn name="Color" type="string">
          {({ value, onChange, variant }) => (
            <Input.Text
              value={value}
              onChange={onChange}
              variant={variant}
              startContent={<div style={{ width: "4rem", height: "4rem" }} />}
            />
          )}
        </Input.TableColumn>
      </Input.Table>,
    );
    const frame = container.querySelector(`tbody .${CSS.B("input")}`)!;
    const cell = frame.closest("td")!;
    expect(frame.getBoundingClientRect().height).toBeCloseTo(cell.clientHeight);
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

  describe.each(["checkbox", "switch"] as const)("boolean %s cell frame", (control) => {
    /** @returns the frame of a boolean cell whose status is the given one. */
    const renderCell = async (status?: Status.Crude): Promise<HTMLElement> => {
      const { container } = await renderThemed(
        <Input.Table value={[[true]]} onChange={() => {}} cellStatus={() => status}>
          {control === "checkbox" ? (
            <Input.TableColumn name="Enabled" type="boolean" width={20} />
          ) : (
            <Input.TableColumn name="Enabled" type="boolean" width={20}>
              {(props) => (
                <Input.Switch
                  value={props.value}
                  onChange={props.onChange}
                  variant={props.variant}
                  aria-label={props["aria-label"]}
                />
              )}
            </Input.TableColumn>
          )}
        </Input.Table>,
      );
      const box = container.querySelector<HTMLElement>(`.${CSS.BE("input", control)}`);
      if (box == null) throw new Error(`The cell has no ${control}`);
      return box;
    };

    const frame = (box: Element): string => computed(box, "outline-color");

    it("should hide the frame at rest", async () => {
      const box = await renderCell();
      expect(frame(box)).toBe(resolve("transparent", "outline-color"));
    });

    it("should frame a marked cell in the error color", async () => {
      const box = await renderCell({ variant: "error", message: "Not valid" });
      expect(frame(box)).toBe(resolve("var(--pluto-error-z)", "outline-color"));
    });

    it("should frame a hovered cell", async () => {
      const box = await renderCell();
      await userEvent.hover(box);
      await expect
        .poll(() => frame(box))
        .toBe(resolve("var(--pluto-gray-l7)", "outline-color"));
    });

    it("should frame a focused cell in the primary color", async () => {
      const box = await renderCell();
      box.querySelector("input")?.focus();
      await expect
        .poll(() => frame(box))
        .toBe(resolve("var(--pluto-primary-z)", "outline-color"));
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

    it("should keep the add button in place as the first values arrive", async () => {
      const table = (value: number[][]) => (
        <div style={{ width: "600px" }}>
          <Input.Table
            value={value}
            onChange={() => {}}
            onMove={() => {}}
            rowKey={String}
          >
            <Input.TableColumn name="On" width={3} />
            <Input.TableColumn name="Name" />
          </Input.Table>
        </div>
      );
      const { container, rerender } = await renderThemed(table([]));
      const add = (): number =>
        container.querySelector("tfoot button")!.getBoundingClientRect().left;
      const empty = add();
      await rerender(table([[1, 2]]));
      expect(add()).toBe(empty);
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
    autoFocus?: boolean;
    cellStatus?: (at: xy.XY) => Status.Crude | undefined;
    onRemove?: (index: number) => void;
    onMove?: (from: number, to: number) => void;
  }

  /** @returns a row key that reads the row's first cell. */
  const keyByFirstCell =
    (value: Input.TableCell[][]) =>
    (index: number): string =>
      String(value[index][0]);

  const renderTable = ({ value = [[1, 10]], onMove, ...rest }: RenderOptions = {}) => {
    const onChange = vi.fn();
    const moves = onMove == null ? {} : { onMove, rowKey: keyByFirstCell(value) };
    render(
      <Input.Table value={value} onChange={onChange} {...rest} {...moves}>
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

  const firePaste = (target: HTMLElement, text: string): void => {
    const clipboardData = new DataTransfer();
    clipboardData.setData("text/plain", text);
    fireEvent(
      target,
      new ClipboardEvent("paste", { clipboardData, bubbles: true, cancelable: true }),
    );
  };

  const paste = (el: HTMLElement, text: string): void => {
    fireEvent.focus(el);
    firePaste(el, text);
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

    it("should focus the first cell on mount with autoFocus", () => {
      renderTable({ value: VALUE, autoFocus: true });
      expect(document.activeElement).toBe(cell("Raw", "1"));
    });

    it("should focus the add button on mount with autoFocus and no rows", () => {
      renderTable({ value: [], autoFocus: true });
      expect(document.activeElement).toBe(addButton());
    });

    it("should take no focus on mount without autoFocus", () => {
      renderTable({ value: VALUE });
      expect(document.activeElement).toBe(document.body);
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

  describe("move", () => {
    const ROWS: Input.TableCell[][] = [
      [1, 10],
      [2, 20],
      [3, 30],
    ];

    const handle = (row: string, table = 0): HTMLElement =>
      screen.getAllByRole("button", { name: `Move row ${row}` })[table];

    const rowOf = (row: string, table = 0): HTMLElement => {
      const tr = handle(row, table).closest("tr");
      if (tr == null) throw new Error(`Row ${row} has no element`);
      return tr;
    };

    interface MovableProps {
      initial: Input.TableCell[][];
      /** Ignores each move, as a caller that keeps a row in place would. */
      refused?: boolean;
      preview?: boolean;
      renderHaulPreview?: (index: number) => ReactNode;
    }

    interface KeyedRow {
      key: string;
      cells: Input.TableCell[];
    }

    // Holds keyed rows and applies each edit, removal, and move, as a form would.
    const Movable = ({
      initial,
      refused = false,
      ...rest
    }: MovableProps): ReactElement => {
      const [rows, setRows] = useState<KeyedRow[]>(() =>
        initial.map((cells, i) => ({ key: `row-${i}`, cells })),
      );
      const handleChange = (next: Input.TableCell[][]) =>
        setRows((prev) =>
          next.map((cells, i) => ({ key: prev[i]?.key ?? `new-${i}`, cells })),
        );
      const handleRemove = (index: number) =>
        setRows((prev) => prev.filter((_, i) => i !== index));
      const handleMove = (from: number, to: number) => {
        if (refused) return;
        setRows((prev) => {
          const next = [...prev];
          const [row] = next.splice(from, 1);
          next.splice(to, 0, row);
          return next;
        });
      };
      return (
        <Haul.Provider>
          <Input.Table
            value={rows.map(({ cells }) => cells)}
            onChange={handleChange}
            onRemove={handleRemove}
            onMove={handleMove}
            rowKey={(i) => rows[i].key}
            {...rest}
          >
            {COLUMNS}
          </Input.Table>
        </Haul.Provider>
      );
    };

    const renderDraggable = (onMove: (from: number, to: number) => void) =>
      render(
        <Haul.Provider>
          <Input.Table
            value={ROWS}
            onChange={vi.fn()}
            onMove={onMove}
            rowKey={keyByFirstCell(ROWS)}
          >
            {COLUMNS}
          </Input.Table>
        </Haul.Provider>,
      );

    it("should move a row down and up with the arrow keys on its handle", () => {
      const onMove = vi.fn();
      renderTable({ value: ROWS, onMove });
      fireEvent.keyDown(handle("2"), { key: "ArrowDown" });
      fireEvent.keyDown(handle("2"), { key: "ArrowUp" });
      expect(onMove.mock.calls).toEqual([
        [1, 2],
        [1, 0],
      ]);
    });

    it("should keep each cell's input when the table leaves preview", () => {
      const value: Input.TableCell[][] = [
        [1, true],
        [2, false],
      ];
      const props = { value, onChange: vi.fn(), onMove: vi.fn(), rowKey: String };
      const columns = [
        <Input.TableColumn key="raw" name="Raw" />,
        <Input.TableColumn key="on" name="On" type="boolean" />,
      ];
      const { rerender } = render(
        <Haul.Provider>
          <Input.Table {...props} preview>
            {columns}
          </Input.Table>
        </Haul.Provider>,
      );
      const on = cell("On", "1");
      rerender(
        <Haul.Provider>
          <Input.Table {...props}>{columns}</Input.Table>
        </Haul.Provider>,
      );
      expect(cell("On", "1")).toBe(on);
    });

    it("should not move the first row up or the last row down", () => {
      const onMove = vi.fn();
      renderTable({ value: ROWS, onMove });
      fireEvent.keyDown(handle("1"), { key: "ArrowUp" });
      fireEvent.keyDown(handle("3"), { key: "ArrowDown" });
      expect(onMove).not.toHaveBeenCalled();
    });

    it("should keep focus on the handle of the row it moves", () => {
      render(<Movable initial={ROWS} />);
      handle("1").focus();
      fireEvent.keyDown(handle("1"), { key: "ArrowDown" });
      expect(cell("Raw", "2").value).toBe("1");
      expect(document.activeElement).toBe(handle("2"));
    });

    it("should drop a dragged row in the place of the row it lands on", () => {
      const onMove = vi.fn();
      renderDraggable(onMove);
      fireEvent.dragStart(handle("1"), { dataTransfer: new DataTransfer() });
      fireEvent.drop(rowOf("3"));
      expect(onMove).toHaveBeenCalledWith(0, 2);
    });

    it("should dim the whole row while it drags", () => {
      renderDraggable(vi.fn());
      fireEvent.dragStart(handle("1"), { dataTransfer: new DataTransfer() });
      expect(rowOf("1").hasAttribute("data-haul-source")).toBe(true);
      fireEvent.dragEnd(handle("1"), { dataTransfer: new DataTransfer() });
      expect(rowOf("1").hasAttribute("data-haul-source")).toBe(false);
    });

    it("should not drag a row from one of its cells", () => {
      const onMove = vi.fn();
      renderDraggable(onMove);
      fireEvent.dragStart(cell("Raw", "1"), { dataTransfer: new DataTransfer() });
      fireEvent.drop(rowOf("3"));
      expect(onMove).not.toHaveBeenCalled();
    });

    it("should not take a row from another table", () => {
      const onMove = vi.fn();
      render(
        <Haul.Provider>
          <Input.Table
            value={ROWS}
            onChange={vi.fn()}
            onMove={vi.fn()}
            rowKey={keyByFirstCell(ROWS)}
          >
            {COLUMNS}
          </Input.Table>
          <Input.Table
            value={ROWS}
            onChange={vi.fn()}
            onMove={onMove}
            rowKey={keyByFirstCell(ROWS)}
          >
            {COLUMNS}
          </Input.Table>
        </Haul.Provider>,
      );
      fireEvent.dragStart(handle("1", 0), { dataTransfer: new DataTransfer() });
      fireEvent.drop(rowOf("3", 1));
      expect(onMove).not.toHaveBeenCalled();
    });

    it("should move a mark with its row", () => {
      render(<Movable initial={ROWS} />);
      paste(cell("Raw", "1"), "x\t11");
      fireEvent.keyDown(handle("1"), { key: "ArrowDown" });
      expect(marked(cell("Raw", "2"))).toBe(true);
      expect(marked(cell("Raw", "1"))).toBe(false);
    });

    it("should move the mark of a row that a paste added with its row", () => {
      render(<Movable initial={ROWS} />);
      paste(cell("Raw", "3"), "3\t30\nx\t40");
      expect(marked(cell("Raw", "4"))).toBe(true);
      fireEvent.keyDown(handle("4"), { key: "ArrowUp" });
      expect(cell("Scaled", "3").value).toBe("40");
      expect(marked(cell("Raw", "3"))).toBe(true);
      expect(marked(cell("Raw", "4"))).toBe(false);
    });

    it("should show no gutter in preview", () => {
      renderTable({ value: ROWS, onMove: vi.fn(), preview: true });
      expect(screen.queryByRole("button", { name: "Move row 1" })).toBeNull();
      expect(screen.queryAllByRole("rowheader")).toHaveLength(0);
    });

    it("should name a handle after its row's label", () => {
      renderTable({ value: ROWS, onMove: vi.fn(), rowLabel: (i) => `c${i}` });
      expect(screen.getByRole("button", { name: "Move row c0" })).toBeTruthy();
    });

    it("should keep focus in the cell being edited after a refused move", () => {
      render(<Movable initial={ROWS} refused />);
      fireEvent.keyDown(handle("1"), { key: "ArrowDown" });
      const input = cell("Raw", "3");
      input.focus();
      paste(input, "4\t40");
      expect(cell("Scaled", "3").value).toBe("40");
      expect(document.activeElement).toBe(input);
    });

    it("should keep a mark on its row after a refused move", () => {
      render(<Movable initial={ROWS} refused />);
      paste(cell("Raw", "1"), "x\t11");
      fireEvent.keyDown(handle("1"), { key: "ArrowDown" });
      commit(cell("Scaled", "3"), "31");
      expect(marked(cell("Raw", "1"))).toBe(true);
      expect(marked(cell("Raw", "2"))).toBe(false);
    });

    it("should not take a remove after a refused move as the move", () => {
      render(<Movable initial={ROWS} refused />);
      paste(cell("Raw", "3"), "x\t31");
      fireEvent.keyDown(handle("2"), { key: "ArrowUp" });
      fireEvent.click(screen.getByRole("button", { name: "Remove row 1" }));
      expect(marked(cell("Raw", "2"))).toBe(true);
      expect(marked(cell("Raw", "1"))).toBe(false);
    });

    it("should move a mark with its row on a drop", () => {
      render(<Movable initial={ROWS} />);
      paste(cell("Raw", "1"), "x\t11");
      fireEvent.dragStart(handle("1"), { dataTransfer: new DataTransfer() });
      fireEvent.drop(rowOf("3"));
      expect(marked(cell("Raw", "3"))).toBe(true);
      expect(marked(cell("Raw", "1"))).toBe(false);
    });

    const renderKeyed = (
      rows: Input.TableCell[][],
      keys: string[],
      onMove = vi.fn(),
    ) => (
      <Haul.Provider>
        <Input.Table
          value={rows}
          onChange={vi.fn()}
          onMove={onMove}
          rowKey={(i) => keys[i]}
        >
          {COLUMNS}
        </Input.Table>
      </Haul.Provider>
    );

    it("should keep a mark on its row when the caller reorders the rows", () => {
      const { rerender } = render(renderKeyed(ROWS, ["a", "b", "c"]));
      paste(cell("Raw", "2"), "x\t21");
      rerender(renderKeyed([ROWS[1], ROWS[0], ROWS[2]], ["b", "a", "c"]));
      expect(marked(cell("Raw", "1"))).toBe(true);
      expect(marked(cell("Raw", "2"))).toBe(false);
    });

    it("should drop the dragged row by its key after the rows above it change", () => {
      const onMove = vi.fn();
      const { rerender } = render(renderKeyed(ROWS, ["a", "b", "c"], onMove));
      fireEvent.dragStart(handle("1"), { dataTransfer: new DataTransfer() });
      rerender(renderKeyed([[0, 0], ...ROWS], ["z", "a", "b", "c"], onMove));
      fireEvent.drop(rowOf("4"));
      expect(onMove).toHaveBeenCalledWith(1, 3);
    });

    it("should draw the drop line on the side the row lands on", () => {
      renderDraggable(vi.fn());
      fireEvent.dragStart(handle("1"), { dataTransfer: new DataTransfer() });
      fireEvent.dragOver(rowOf("3"));
      expect(rowOf("3").classList).toContain(CSS.BEM("input", "table-row", "below"));
      fireEvent.dragLeave(rowOf("3"));
      expect(rowOf("3").classList).not.toContain(
        CSS.BEM("input", "table-row", "below"),
      );
      fireEvent.dragEnd(handle("1"), { dataTransfer: new DataTransfer() });
      fireEvent.dragStart(handle("3"), { dataTransfer: new DataTransfer() });
      fireEvent.dragOver(rowOf("1"));
      expect(rowOf("1").classList).toContain(CSS.BEM("input", "table-row", "above"));
      fireEvent.drop(rowOf("1"));
      expect(rowOf("1").classList).not.toContain(
        CSS.BEM("input", "table-row", "above"),
      );
    });

    it("should picture a dragged row with its name by default", () => {
      renderDraggable(vi.fn());
      fireEvent.dragStart(handle("2"), { dataTransfer: new DataTransfer() });
      expect(screen.getByText("Row 2")).toBeTruthy();
    });

    it("should picture a dragged row with the caller's preview", () => {
      render(
        <Movable initial={ROWS} renderHaulPreview={(i) => <span>Pump {i}</span>} />,
      );
      fireEvent.dragStart(handle("2"), { dataTransfer: new DataTransfer() });
      expect(screen.getByText("Pump 1")).toBeTruthy();
    });

    it("should take no drop in preview", () => {
      const onMove = vi.fn();
      const { rerender } = render(
        <Haul.Provider>
          <Input.Table
            value={ROWS}
            onChange={vi.fn()}
            onMove={onMove}
            rowKey={keyByFirstCell(ROWS)}
          >
            {COLUMNS}
          </Input.Table>
        </Haul.Provider>,
      );
      fireEvent.dragStart(handle("1"), { dataTransfer: new DataTransfer() });
      const target = rowOf("3");
      rerender(
        <Haul.Provider>
          <Input.Table
            value={ROWS}
            onChange={vi.fn()}
            onMove={onMove}
            rowKey={keyByFirstCell(ROWS)}
            preview
          >
            {COLUMNS}
          </Input.Table>
        </Haul.Provider>,
      );
      fireEvent.drop(target);
      expect(onMove).not.toHaveBeenCalled();
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

    it("should give a custom cell the index of its row", () => {
      render(
        <Input.Table value={[["a"], ["b"]]} onChange={vi.fn()}>
          <Input.TableColumn<string> name="Name" type="custom" empty="">
            {({ value, row }) => <span>{`${value} at ${row}`}</span>}
          </Input.TableColumn>
        </Input.Table>,
      );
      expect(screen.getByText("b at 1")).toBeTruthy();
    });

    it("should leave the row off the default inputs", () => {
      renderPoints();
      expect(cell("Name", "1").hasAttribute("row")).toBe(false);
    });

    it("should leave the row off an input that a typed column's renderer spreads", () => {
      render(
        <Input.Table value={[["a"]]} onChange={vi.fn()}>
          <Input.TableColumn name="Name" type="string">
            {(p) => <Input.Text {...p} />}
          </Input.TableColumn>
        </Input.Table>,
      );
      expect(cell("Name", "1").hasAttribute("row")).toBe(false);
    });

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

    it("should leave one value pasted on a select whose column has no parse", () => {
      const { onChange, order } = renderOrders();
      paste(order, "foo");
      expect(onChange).not.toHaveBeenCalled();
    });

    it("should read a single value pasted on a select with its column's parse", () => {
      const { onChange, order } = renderOrders((text) =>
        ORDERS.includes(text) ? text : null,
      );
      paste(order, "little");
      expect(onChange).toHaveBeenCalledWith([["little"]]);
    });
  });
});
