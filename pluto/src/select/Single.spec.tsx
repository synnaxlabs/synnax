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

import { renderProp } from "@/component/renderProp";
import { List } from "@/list";
import { Select } from "@/select";
import { mockBoundingClientRect } from "@/testutil/dom";
import { Text } from "@/text";

describe("Select.Single", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  interface TestEntry {
    key: string;
    name: string;
  }

  const testData: TestEntry[] = [
    { key: "1", name: "First Item" },
    { key: "2", name: "Second Item" },
    { key: "3", name: "Third Item" },
  ];

  const listItemRenderProp = renderProp((props: List.ItemProps<string>) => {
    const { itemKey } = props;
    const item = testData.find((i) => i.key === itemKey);
    return (
      <Select.ListItem {...props}>
        <Text.Text>{item?.name} Option</Text.Text>
      </Select.ListItem>
    );
  });

  const createSelectSingle = () => {
    const onChange = vi.fn();
    const SelectSingle = (
      props: Omit<
        Select.SingleProps<string, TestEntry>,
        "data" | "value" | "onChange" | "children" | "resourceName"
      >,
    ) => {
      const { data, getItem, retrieve } = List.useStaticData<string, TestEntry>({
        data: testData,
      });
      const { search } = List.usePager({ retrieve });
      const [value, setValue] = useState<string | undefined>(undefined);
      const handleChange = (key: string | undefined) => {
        setValue(key);
        onChange(key);
      };
      return (
        <Select.Single<string, TestEntry>
          getItem={getItem}
          data={data}
          value={value}
          onChange={handleChange}
          resourceName="Test Item"
          onSearch={search}
          {...props}
        >
          {listItemRenderProp}
        </Select.Single>
      );
    };
    return {
      SelectSingle,
      onChange,
    };
  };

  it("should render a selection trigger", () => {
    const { SelectSingle } = createSelectSingle();
    const c = render(<SelectSingle />);
    expect(c.getByText("Select a Test Item")).toBeTruthy();
  });

  it("should open the selection dialog when the trigger is clicked", () => {
    const { SelectSingle } = createSelectSingle();
    const c = render(<SelectSingle />);
    fireEvent.click(c.getByText("Select a Test Item"));
    expect(c.getByText("First Item Option")).toBeTruthy();
    expect(c.getByText("Second Item Option")).toBeTruthy();
    expect(c.getByText("Third Item Option")).toBeTruthy();
  });

  it("should close the selection dialog when the user clicks on the trigger", () => {
    const { SelectSingle } = createSelectSingle();
    const c = render(<SelectSingle />);
    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.click(c.getByText("Select a Test Item"));
    expect(c.queryByText("First Item Option")).toBeNull();
  });

  it("should call onChange when an item is selected", () => {
    const { SelectSingle, onChange } = createSelectSingle();
    const c = render(<SelectSingle />);
    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith("1");
  });

  it("should close the dialog when an item is selected", () => {
    const { SelectSingle } = createSelectSingle();
    const c = render(<SelectSingle />);
    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(c.queryByText("First Item Option")).toBeNull();
    expect(c.queryByText("Second Item Option")).toBeNull();
    expect(c.queryByText("Third Item Option")).toBeNull();
  });

  it("should display the selected item name in the trigger", () => {
    const { SelectSingle } = createSelectSingle();
    const c = render(<SelectSingle />);
    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(c.getByText("First Item")).toBeTruthy();
    expect(c.queryByText("Select a Test Item")).toBeNull();
  });

  it("should allow the user to change selection", () => {
    const { SelectSingle, onChange } = createSelectSingle();
    const c = render(<SelectSingle />);

    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith("1");

    fireEvent.click(c.getByText("First Item"));
    fireEvent.click(c.getByText("Second Item Option"));
    expect(onChange).toHaveBeenCalledWith("2");
    expect(c.getByText("Second Item")).toBeTruthy();
    expect(c.queryByText("First Item")).toBeNull();
  });

  it("should allow the user to deselect an item when allowNone is true", () => {
    const { SelectSingle, onChange } = createSelectSingle();
    const c = render(<SelectSingle allowNone />);
    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.click(c.getByText("First Item Option"));

    fireEvent.click(c.getByText("First Item"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenLastCalledWith(null);
    expect(c.getByText("Select a Test Item")).toBeTruthy();
  });

  it("should not allow the user to deselect when allowNone is false", () => {
    const { SelectSingle, onChange } = createSelectSingle();
    const c = render(<SelectSingle allowNone={false} />);
    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.click(c.getByText("First Item Option"));

    fireEvent.click(c.getByText("First Item"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenLastCalledWith("1");
    expect(c.getByText("First Item")).toBeTruthy();
  });

  it("should allow the caller to search for an item", () => {
    const { SelectSingle } = createSelectSingle();
    const c = render(<SelectSingle />);
    fireEvent.click(c.getByText("Select a Test Item"));
    fireEvent.change(c.getByPlaceholderText("Search Test Items..."), {
      target: { value: "First" },
    });
    expect(c.getByText("First Item Option")).toBeTruthy();
    expect(c.queryByText("Second Item Option")).toBeNull();
    expect(c.queryByText("Third Item Option")).toBeNull();
  });

  it("should work with initial value", () => {
    const onChange = vi.fn();
    const SelectSingle = () => {
      const { data, getItem, retrieve } = List.useStaticData<string, TestEntry>({
        data: testData,
      });
      const { search } = List.usePager({ retrieve });
      return (
        <Select.Single<string, TestEntry>
          getItem={getItem}
          data={data}
          value="2"
          onChange={onChange}
          resourceName="Test Item"
          onSearch={search}
        >
          {listItemRenderProp}
        </Select.Single>
      );
    };
    const c = render(<SelectSingle />);
    expect(c.getByText("Second Item")).toBeTruthy();
    expect(c.queryByText("Select a Test Item")).toBeNull();
  });
});
