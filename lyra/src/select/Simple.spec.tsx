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
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Icon } from "@/icon";
import { Select } from "@/select";
import { mockBoundingClientRect } from "@/testutil/dom";

describe("Select.Simple", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });
  const onChange = vi.fn();
  const SelectSimple = () => {
    const [value, setValue] = useState("");
    const handleChange = (key: string) => {
      setValue(key);
      onChange(key);
    };
    return (
      <Select.Simple<string>
        value={value}
        onChange={handleChange}
        resourceName="Test Item"
      >
        <Select.Item itemKey="1">First Item</Select.Item>
        <Select.Item itemKey="2">
          <Icon.Add />
          Second Item
        </Select.Item>
        <Select.Item itemKey="3">Third Item</Select.Item>
        <Select.Item itemKey="4">Another Item</Select.Item>
      </Select.Simple>
    );
  };

  it("should render a selection trigger", () => {
    const c = render(<SelectSimple />);
    expect(c.getByText("Test Item")).toBeTruthy();
  });

  it("should open the selection dialog when the trigger is clicked", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    expect(c.getByText("First Item")).toBeTruthy();
    expect(c.getByText("Second Item")).toBeTruthy();
  });

  it("should call onChange when an item is selected and the dialog is closed", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    fireEvent.click(c.getByText("Second Item"));
    expect(onChange).toHaveBeenCalledWith("2");
  });

  it("should render a search box", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    expect(c.getByPlaceholderText("Search Test Items...")).toBeTruthy();
  });

  it("should filter the list when the search box is typed into", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    fireEvent.change(c.getByPlaceholderText("Search Test Items..."), {
      target: { value: "Second" },
    });
    const hidden = (text: string) => c.getByText(text).closest("[hidden]") != null;
    expect(hidden("Second Item")).toBe(false);
    expect(hidden("First Item")).toBe(true);
    expect(hidden("Third Item")).toBe(true);
  });

  it("should show the selected item's label in the trigger once closed", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    fireEvent.click(c.getByText("Third Item"));
    expect(c.queryByPlaceholderText("Search Test Items...")).toBeNull();
    expect(c.getByRole("button", { name: "Test Item" }).textContent).toBe("Third Item");
  });

  it("should show empty content when the search hides every item", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    fireEvent.change(c.getByPlaceholderText("Search Test Items..."), {
      target: { value: "zzz" },
    });
    expect(c.getByText("No Test Items found")).toBeTruthy();
  });

  it("should clear the search when the dialog closes", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    fireEvent.change(c.getByPlaceholderText("Search Test Items..."), {
      target: { value: "Second" },
    });
    fireEvent.click(c.getByText("Second Item"));
    fireEvent.click(c.getByRole("button", { name: "Test Item" }));
    expect(c.getByText("First Item").closest("[hidden]")).toBeNull();
  });
});
