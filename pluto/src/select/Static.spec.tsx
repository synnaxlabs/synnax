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

interface TestEntry {
  key: string;
  name: string;
  icon?: Icon.ReactElement;
}

describe("Select.Static", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });
  const testData: TestEntry[] = [
    { key: "1", name: "First Item" },
    { key: "2", name: "Second Item", icon: <Icon.Add /> },
    { key: "3", name: "Third Item" },
    { key: "4", name: "Another Item" },
  ];

  const onChange = vi.fn();
  const SelectSimple = () => {
    const [value, setValue] = useState("");
    const handleChange = (key: string) => {
      setValue(key);
      onChange(key);
    };
    return (
      <Select.Static<string, TestEntry>
        value={value}
        data={testData}
        onChange={handleChange}
        resourceName="Test Item"
      />
    );
  };

  it("should render a selection trigger", () => {
    const c = render(<SelectSimple />);
    expect(c.getByText("Select a Test Item")).toBeTruthy();
  });

  it("should open the selection dialog when the trigger is clicked", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Select a Test Item"));
    expect(c.getByText("First Item")).toBeTruthy();
    expect(c.getByText("Second Item")).toBeTruthy();
  });

  it("should call onChange when an item is selected and the dialog is closed", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.click(c.getByText("Second Item"));
    expect(onChange).toHaveBeenCalledWith("2");
  });

  it("should render a search box", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Select a Test Item"));
    expect(c.getByPlaceholderText("Search Test Items...")).toBeTruthy();
  });

  it("should filter the list when the search box is typed into", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.change(c.getByPlaceholderText("Search Test Items..."), {
      target: { value: "Second" },
    });
    expect(c.getByText("Second Item")).toBeTruthy();
    expect(c.queryByText("First Item")).toBeNull();
    expect(c.queryByText("Third Item")).toBeNull();
  });
});
