// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Color } from "@/color";
import { Input } from "@/input";
import { Select } from "@/select";
import { type Status } from "@/status";
import { Theming } from "@/theming";

const textbox = (name?: string): HTMLInputElement =>
  screen.getByRole<HTMLInputElement>("textbox", { name });

const frame = (el: HTMLElement = textbox()): HTMLElement => {
  const f = el.closest<HTMLElement>(".lyra-input, .lyra-input__switch");
  if (f == null) throw new Error("The control has no frame");
  return f;
};

/** @returns the tooltip text a hover on the anchor shows. */
const hoverTip = (anchor: HTMLElement = frame()): string | null => {
  vi.useFakeTimers();
  try {
    anchor.getBoundingClientRect = () => new DOMRect(0, 0, 100, 20);
    fireEvent.pointerOver(anchor, { pointerType: "mouse" });
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    return screen.queryByRole("tooltip")?.textContent ?? null;
  } finally {
    vi.useRealTimers();
  }
};

const byClass = (className: string): HTMLElement => {
  const el = document.querySelector<HTMLElement>(`.${className}`);
  if (el == null) throw new Error(`Nothing has the class ${className}`);
  return el;
};

const PORTS = Array.from({ length: 12 }, (_, i) => `ai${i}`);

const ports = (): ReactElement => (
  <Select.Simple<string> value="ai1" onChange={vi.fn()} resourceName="port">
    {PORTS.map((port) => (
      <Select.Item key={port} itemKey={port}>
        {port}
      </Select.Item>
    ))}
  </Select.Simple>
);

const input = (props: Partial<Input.TextProps> = {}): ReactElement => (
  <Input.Text value="9" onChange={vi.fn()} {...props} />
);

interface ItemOptions {
  status?: Status.Variant;
  helpText?: string;
  showLabel?: boolean;
  showHelpText?: boolean;
  control?: ReactElement;
}

const item = ({
  status,
  helpText,
  showLabel = false,
  showHelpText = false,
  control = input(),
}: ItemOptions = {}): ReactElement => (
  <Input.Item
    label="Line"
    showLabel={showLabel}
    showHelpText={showHelpText}
    status={status}
    helpText={helpText}
  >
    {control}
  </Input.Item>
);

/** Asserts that the focused textbox stays mounted as an error comes and goes. */
const expectFocusKept = (options: ItemOptions): void => {
  const { rerender } = render(item({ ...options, status: "success", helpText: "" }));
  const el = textbox();
  el.focus();
  rerender(item({ ...options, status: "error", helpText: "port1 has 8 lines" }));
  expect(textbox()).toBe(el);
  expect(document.activeElement).toBe(el);
  rerender(item({ ...options, status: "success", helpText: "" }));
  expect(textbox()).toBe(el);
  expect(document.activeElement).toBe(el);
};

describe("Input.Item", () => {
  describe("with no help text", () => {
    it("should mark its input with an error", () => {
      render(item({ status: "error", helpText: "port1 has 8 lines" }));
      expect(textbox().getAttribute("aria-invalid")).toBe("true");
      expect(frame().className).toContain("lyra--status-error");
    });

    it("should show the status message in the input's tooltip", () => {
      render(item({ status: "error", helpText: "port1 has 8 lines" }));
      expect(hoverTip()).toBe("port1 has 8 lines");
    });

    it("should tint its input with a warning without marking it invalid", () => {
      render(item({ status: "warning", helpText: "Slow" }));
      expect(frame().className).toContain("lyra--status-warning");
      expect(textbox().getAttribute("aria-invalid")).toBeNull();
    });

    it("should leave its input unmarked without a status", () => {
      render(item({ status: "success", helpText: "" }));
      expect(frame().className).not.toContain("lyra--status");
      expect(textbox().getAttribute("aria-invalid")).toBeNull();
      expect(hoverTip()).toBeNull();
    });

    it("should keep the focused input as an error shows and clears", () => {
      expectFocusKept({});
    });

    it("should show the input's own tooltip while it has no error", () => {
      const control = input({ tooltip: "Zero-based" });
      const { rerender } = render(item({ control }));
      expect(hoverTip()).toBe("Zero-based");
      rerender(item({ control, status: "error", helpText: "Out of range" }));
      expect(hoverTip()).toBe("Out of range");
    });

    it("should let the input's own status win", () => {
      render(
        item({
          status: "error",
          helpText: "Bad",
          control: input({ status: "warning" }),
        }),
      );
      expect(frame().className).toContain("lyra--status-warning");
      expect(frame().className).not.toContain("lyra--status-error");
    });

    it("should show the status on a labeled input", () => {
      render(item({ showLabel: true, status: "error", helpText: "Bad" }));
      expect(textbox("Line").getAttribute("aria-invalid")).toBe("true");
      expect(hoverTip()).toBe("Bad");
    });

    it("should mark a switch with an error", () => {
      render(
        item({
          status: "error",
          helpText: "Must be on",
          control: <Input.Switch value={false} onChange={vi.fn()} aria-label="On" />,
        }),
      );
      const el = screen.getByRole("checkbox", { name: "On" });
      expect(el.getAttribute("aria-invalid")).toBe("true");
      expect(hoverTip(frame(el))).toBe("Must be on");
    });

    it("should mark a select with an error", () => {
      render(
        item({
          status: "error",
          helpText: "Port in use",
          control: (
            <Select.Simple<string> value="1" onChange={vi.fn()} resourceName="port">
              <Select.Item itemKey="1">ai0</Select.Item>
            </Select.Simple>
          ),
        }),
      );
      const trigger = screen.getByRole("button", { name: /ai0/ });
      expect(trigger.getAttribute("aria-invalid")).toBe("true");
      expect(trigger.className).toContain("lyra--status-error");
      expect(hoverTip(trigger)).toBe("Port in use");
    });

    it("should mark a multi select with an error", () => {
      render(
        item({
          status: "error",
          helpText: "Pick a label",
          control: (
            <Select.Simple<string>
              multiple
              value={["1"]}
              onChange={vi.fn()}
              resourceName="label"
            >
              <Select.Item itemKey="1">Cold</Select.Item>
            </Select.Simple>
          ),
        }),
      );
      const trigger = byClass("lyra-dialog__trigger");
      expect(trigger.getAttribute("aria-invalid")).toBe("true");
      expect(trigger.className).toContain("lyra--status-error");
      expect(hoverTip(trigger)).toBe("Pick a label");
    });

    it("should mark each button of a button group with an error", () => {
      render(
        item({
          status: "error",
          helpText: "Pick a side",
          control: (
            <Select.Buttons<string> value="top" onChange={vi.fn()}>
              <Select.Item itemKey="top">Top</Select.Item>
              <Select.Item itemKey="left">Left</Select.Item>
            </Select.Buttons>
          ),
        }),
      );
      for (const name of ["Top", "Left"]) {
        const button = screen.getByRole("button", { name });
        expect(button.className).toContain("lyra--status-error");
        expect(button.getAttribute("aria-invalid")).toBe("true");
      }
      expect(hoverTip(byClass("lyra-select-btns"))).toBe("Pick a side");
    });

    it("should mark a color swatch with an error", () => {
      render(
        <Theming.Provider>
          {item({
            status: "error",
            helpText: "Too dark",
            control: <Color.Swatch value="#000000" onChange={vi.fn()} />,
          })}
        </Theming.Provider>,
      );
      const swatch = byClass("lyra-color-swatch");
      expect(swatch.getAttribute("aria-invalid")).toBe("true");
      expect(swatch.className).toContain("lyra--status-error");
      expect(hoverTip(swatch)).toBe("Too dark");
    });

    it("should mark a time input with an error", () => {
      render(
        item({
          status: "error",
          helpText: "Too long",
          control: <Input.TimeSpan value={1000} onChange={vi.fn()} />,
        }),
      );
      const trigger = byClass("lyra-time-editor__trigger");
      expect(trigger.getAttribute("aria-invalid")).toBe("true");
      expect(trigger.className).toContain("lyra--status-error");
      expect(hoverTip(trigger)).toBe("Too long");
    });

    it("should leave a control inside its input unmarked", () => {
      render(
        item({
          status: "error",
          helpText: "Bad",
          control: input({
            "aria-label": "Outer",
            children: input({ "aria-label": "Inner" }),
          }),
        }),
      );
      expect(textbox("Outer").getAttribute("aria-invalid")).toBe("true");
      expect(textbox("Inner").getAttribute("aria-invalid")).toBeNull();
    });

    it("should leave an input in a labeled item inside it unmarked", () => {
      render(
        item({
          status: "error",
          helpText: "Bad",
          control: <Input.Item label="Inner">{input()}</Input.Item>,
        }),
      );
      expect(textbox().getAttribute("aria-invalid")).toBeNull();
    });
  });

  describe("with help text", () => {
    it("should show the status as help text, not on the input", () => {
      render(
        item({ showLabel: true, showHelpText: true, status: "error", helpText: "Bad" }),
      );
      expect(screen.getByText("Bad")).toBeTruthy();
      expect(frame().className).not.toContain("lyra--status");
      expect(hoverTip()).toBeNull();
    });

    it("should mark an input with an error and describe it by its help text", () => {
      render(
        item({ showLabel: true, showHelpText: true, status: "error", helpText: "Bad" }),
      );
      const el = textbox("Line");
      expect(el.getAttribute("aria-invalid")).toBe("true");
      const id = el.getAttribute("aria-describedby");
      expect(id).not.toBeNull();
      expect(document.getElementById(id ?? "")?.textContent).toBe("Bad");
    });

    it("should describe each button of a button group by its help text", () => {
      render(
        item({
          showLabel: true,
          showHelpText: true,
          status: "error",
          helpText: "Pick a side",
          control: (
            <Select.Buttons<string> value="top" onChange={vi.fn()}>
              <Select.Item itemKey="top">Top</Select.Item>
              <Select.Item itemKey="left">Left</Select.Item>
            </Select.Buttons>
          ),
        }),
      );
      for (const name of ["Top", "Left"]) {
        const button = screen.getByRole("button", { name });
        expect(button.getAttribute("aria-invalid")).toBe("true");
        const id = button.getAttribute("aria-describedby");
        expect(document.getElementById(id ?? "")?.textContent).toBe("Pick a side");
      }
    });

    it("should leave an input undescribed while it has no message", () => {
      render(item({ showLabel: true, showHelpText: true }));
      expect(textbox("Line").getAttribute("aria-describedby")).toBeNull();
    });

    it("should keep the focused input of a labeled item", () => {
      expectFocusKept({ showLabel: true, showHelpText: true });
    });

    it("should keep the focused input of an item with no label", () => {
      expectFocusKept({ showHelpText: true });
    });
  });

  describe("in a control's popup", () => {
    const errored = (control: ReactElement): ReactElement =>
      item({
        showLabel: true,
        showHelpText: true,
        status: "error",
        helpText: "Bad",
        control,
      });

    it("should leave the search field of a select unmarked", () => {
      render(errored(ports()));
      fireEvent.click(screen.getByRole("button", { name: /ai1/ }));
      const search = screen.getByPlaceholderText("Search ports...");
      expect(search.getAttribute("aria-invalid")).toBeNull();
      expect(frame(search).className).not.toContain("lyra--status");
    });

    it("should leave the field of a time editor unmarked", () => {
      render(errored(<Input.TimeSpan value={1000} onChange={vi.fn()} />));
      fireEvent.click(byClass("lyra-time-editor__trigger"));
      expect(textbox().getAttribute("aria-invalid")).toBeNull();
    });

    it("should leave the fields of a color picker unmarked", () => {
      render(
        <Theming.Provider>
          {errored(<Color.Swatch value="#000000" onChange={vi.fn()} />)}
        </Theming.Provider>,
      );
      fireEvent.click(byClass("lyra-color-swatch"));
      expect(screen.getByLabelText("Hex").getAttribute("aria-invalid")).toBeNull();
    });
  });
});
