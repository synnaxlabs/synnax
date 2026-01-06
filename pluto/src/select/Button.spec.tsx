// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

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

  it("should give the selected button a filled variant", () => {
    const c = render(
      <Select.Buttons keys={[1, 2, 3]} value={1} onChange={vi.fn()}>
        <Select.Button itemKey={1}>Option 1</Select.Button>
        <Select.Button itemKey={2}>Option 2</Select.Button>
        <Select.Button itemKey={3}>Option 3</Select.Button>
      </Select.Buttons>,
    );
    expect(c.getByText("Option 1").closest("button")?.classList).toContain(
      "pluto-btn--filled",
    );
    expect(c.getByText("Option 2").closest("button")?.classList).not.toContain(
      "pluto-btn--filled",
    );
    expect(c.getByText("Option 3").closest("button")?.classList).not.toContain(
      "pluto-btn--filled",
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
      "pluto-btn--filled",
    );
    expect(c.getByText("Option 1").closest("button")?.classList).not.toContain(
      "pluto-btn--filled",
    );
    expect(c.getByText("Option 3").closest("button")?.classList).not.toContain(
      "pluto-btn--filled",
    );
  });
});
