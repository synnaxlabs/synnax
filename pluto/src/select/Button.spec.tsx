// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, type RenderResult } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Component } from "@/component";
import { Select } from "@/select";

describe("Select.Button", () => {
  it("should render a collection of buttons", () => {
    const onChange = vi.fn();
    const c = render(
      <Select.Buttons keys={[1, 2, 3]} value={1} onChange={onChange}>
        <Select.Button itemKey={1}>Option 1</Select.Button>
        <Select.Button itemKey={2}>Option 2</Select.Button>
        <Select.Button itemKey={3}>Option 3</Select.Button>
      </Select.Buttons>,
    );
    expect(c.getByText("Option 1")).toBeTruthy();
    expect(c.getByText("Option 2")).toBeTruthy();
    expect(c.getByText("Option 3")).toBeTruthy();
  });

  it("should mark the selected button with the selected class", () => {
    const c = render(
      <Select.Buttons keys={[1, 2, 3]} value={1} onChange={vi.fn()}>
        <Select.Button itemKey={1}>Option 1</Select.Button>
        <Select.Button itemKey={2}>Option 2</Select.Button>
        <Select.Button itemKey={3}>Option 3</Select.Button>
      </Select.Buttons>,
    );
    expect(c.getByText("Option 1").closest("button")?.classList).toContain(
      "pluto--selected",
    );
    expect(c.getByText("Option 2").closest("button")?.classList).not.toContain(
      "pluto--selected",
    );
    expect(c.getByText("Option 3").closest("button")?.classList).not.toContain(
      "pluto--selected",
    );
  });

  it("should move the selection state when a button is clicked", () => {
    const C = () => {
      const [value, setValue] = useState(1);
      return (
        <Select.Buttons keys={[1, 2, 3]} value={value} onChange={setValue}>
          <Select.Button itemKey={1}>Option 1</Select.Button>
          <Select.Button itemKey={2}>Option 2</Select.Button>
          <Select.Button itemKey={3}>Option 3</Select.Button>
        </Select.Buttons>
      );
    };
    const c = render(<C />);
    fireEvent.click(c.getByText("Option 2"));
    expect(c.getByText("Option 2").closest("button")?.classList).toContain(
      "pluto--selected",
    );
    expect(c.getByText("Option 1").closest("button")?.classList).not.toContain(
      "pluto--selected",
    );
    expect(c.getByText("Option 3").closest("button")?.classList).not.toContain(
      "pluto--selected",
    );
  });

  describe("preview", () => {
    const renderButtons = (preview?: boolean, onChange = vi.fn()) =>
      render(
        <Select.Buttons keys={[1, 2]} value={1} onChange={onChange} preview={preview}>
          <Select.Button itemKey={1}>Option 1</Select.Button>
          <Select.Button itemKey={2}>Option 2</Select.Button>
        </Select.Buttons>,
      );

    it("should render only the selected option as a preview", () => {
      const c = renderButtons(true);
      expect(c.getByText("Option 1").closest("button")?.classList).toContain(
        "pluto-btn--preview",
      );
      expect(c.queryByText("Option 2")).toBeNull();
    });

    it("should render None when nothing is selected", () => {
      const c = render(
        <Select.Buttons keys={[1, 2]} value={undefined} onChange={vi.fn()} preview>
          <Select.Button itemKey={1}>Option 1</Select.Button>
          <Select.Button itemKey={2}>Option 2</Select.Button>
        </Select.Buttons>,
      );
      expect(c.getByText("None")).toBeTruthy();
      expect(c.queryByText("Option 1")).toBeNull();
    });

    it("should not mark any button when preview is unset", () => {
      const c = renderButtons();
      expect(c.getByText("Option 1").closest("button")?.classList).not.toContain(
        "pluto-btn--preview",
      );
    });

    it("should swallow clicks while previewing", () => {
      const onChange = vi.fn();
      const c = renderButtons(true, onChange);
      fireEvent.click(c.getByText("Option 1"));
      expect(onChange).not.toHaveBeenCalled();
    });
  });
});

// Arrow keys walk the `keys` array, so buttons that drift out of that order break
// navigation with nothing on screen to show it.
describe("picker button order", () => {
  const ids = (c: RenderResult): string[] =>
    Array.from(c.container.querySelectorAll("button")).map((b) => b.id);

  it("should render text levels from XS to XL", () =>
    expect(ids(render(<Select.Text.Level value="h4" onChange={vi.fn()} />))).toEqual([
      "small",
      "h5",
      "h4",
      "h3",
      "h2",
    ]));

  it("should render component sizes from XS to XL", () =>
    expect(
      ids(render(<Component.SelectSize value="medium" onChange={vi.fn()} />)),
    ).toEqual(["tiny", "small", "medium", "large", "huge"]));

  it("should render text weights from light to bold", () =>
    expect(ids(render(<Select.Text.Weight value={400} onChange={vi.fn()} />))).toEqual([
      "250",
      "400",
      "500",
      "600",
    ]));

  it("should render alignments from start to end", () =>
    expect(
      ids(render(<Select.Flex.Alignment value="center" onChange={vi.fn()} />)),
    ).toEqual(["start", "center", "end"]));
});
