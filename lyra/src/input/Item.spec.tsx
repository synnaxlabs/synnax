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

import { Input } from "@/input";
import { type Status } from "@/status";

const textbox = (): HTMLInputElement => screen.getByRole<HTMLInputElement>("textbox");

const frame = (): HTMLElement => {
  const el = textbox().closest<HTMLElement>(".lyra-input");
  if (el == null) throw new Error("The textbox has no input frame");
  return el;
};

/** @returns the tooltip text a hover on the input's frame shows. */
const hoverTip = (): string | null => {
  vi.useFakeTimers();
  try {
    const el = frame();
    el.getBoundingClientRect = () => new DOMRect(0, 0, 100, 20);
    fireEvent.pointerOver(el, { pointerType: "mouse" });
    act(() => {
      vi.advanceTimersByTime(2000);
    });
    return screen.queryByRole("tooltip")?.textContent ?? null;
  } finally {
    vi.useRealTimers();
  }
};

const input = (): ReactElement => <Input.Text value="9" onChange={vi.fn()} />;

const bare = (status?: Status.Variant, helpText?: string): ReactElement => (
  <Input.Item
    label="Line"
    showLabel={false}
    showHelpText={false}
    status={status}
    helpText={helpText}
  >
    {input()}
  </Input.Item>
);

const renderBare = (status?: Status.Variant, helpText?: string) =>
  render(bare(status, helpText));

describe("Input.Item", () => {
  describe("with no label and no help text", () => {
    it("should mark its input with an error", () => {
      renderBare("error", "port1 has 8 lines");
      expect(textbox().getAttribute("aria-invalid")).toBe("true");
      expect(frame().className).toContain("lyra--status-error");
    });

    it("should show the status message in the input's tooltip", () => {
      renderBare("error", "port1 has 8 lines");
      expect(hoverTip()).toBe("port1 has 8 lines");
    });

    it("should tint its input with a warning without marking it invalid", () => {
      renderBare("warning", "Slow");
      expect(frame().className).toContain("lyra--status-warning");
      expect(textbox().getAttribute("aria-invalid")).toBeNull();
    });

    it("should leave its input unmarked without a status", () => {
      renderBare("success", "");
      expect(frame().className).not.toContain("lyra--status");
      expect(textbox().getAttribute("aria-invalid")).toBeNull();
      expect(hoverTip()).toBeNull();
    });

    it("should keep the focused input as an error shows and clears", () => {
      const { rerender } = renderBare("success", "");
      const el = textbox();
      el.focus();
      rerender(bare("error", "port1 has 8 lines"));
      expect(textbox()).toBe(el);
      expect(document.activeElement).toBe(el);
      rerender(bare("success", ""));
      expect(textbox()).toBe(el);
      expect(document.activeElement).toBe(el);
    });

    it("should keep its status from a labeled item inside it", () => {
      render(
        <Input.Item
          showLabel={false}
          showHelpText={false}
          status="error"
          helpText="Bad"
        >
          <Input.Item label="Inner">{input()}</Input.Item>
        </Input.Item>,
      );
      expect(textbox().getAttribute("aria-invalid")).toBeNull();
    });
  });

  it("should show the status as help text, not on the input, when a label shows", () => {
    render(
      <Input.Item label="Line" status="error" helpText="port1 has 8 lines">
        {input()}
      </Input.Item>,
    );
    expect(screen.getByText("port1 has 8 lines")).toBeTruthy();
    expect(textbox().getAttribute("aria-invalid")).toBeNull();
  });
});
