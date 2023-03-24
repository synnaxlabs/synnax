// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { Dropdown } from ".";

import { Button } from "@/core/Button";

const TestDropdown = (): JSX.Element => {
  const { toggle, visible } = Dropdown.use();

  return (
    <Dropdown visible={visible}>
      <Button onClick={() => toggle()}>Toggle</Button>
      <p>Content</p>
    </Dropdown>
  );
};

describe("Dropdown", () => {
  it("should render a dropdown", () => {
    const c = render(<TestDropdown />);
    expect(c.getByText("Toggle")).toBeTruthy();
    const dialog = c.getByRole("dialog");
    expect(dialog).toBeTruthy();
    expect(dialog.className).toContain("hidden");
  });
  it("should open the dropdown when the toggle button is clicked", async () => {
    const c = render(<TestDropdown />);
    const toggle = c.getByText("Toggle");
    const dialog = c.getByRole("dialog");
    expect(dialog.className).toContain("hidden");
    await userEvent.click(toggle);
    expect(dialog.className).not.toContain("hidden");
  });
});
