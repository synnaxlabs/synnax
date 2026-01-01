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
import { Triggers } from "@/triggers";

describe("Select.Multiple", () => {
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

  const createSelectMultiple = () => {
    const onChange = vi.fn();
    const SelectMultiple = (
      props: Omit<
        Select.MultipleProps<string, TestEntry>,
        "data" | "value" | "onChange" | "children" | "resourceName"
      >,
    ) => {
      const { data, getItem, retrieve } = List.useStaticData<string, TestEntry>({
        data: testData,
      });
      const { search } = List.usePager({ retrieve });
      const [value, setValue] = useState<string[]>([]);
      const handleChange = (keys: string[]) => {
        setValue(keys);
        onChange(keys);
      };
      return (
        <Triggers.Provider>
          <Select.Multiple<string, TestEntry>
            getItem={getItem}
            data={data}
            value={value}
            onChange={handleChange}
            resourceName="Test Item"
            onSearch={search}
            {...props}
          >
            {listItemRenderProp}
          </Select.Multiple>
        </Triggers.Provider>
      );
    };
    return {
      SelectMultiple,
      onChange,
    };
  };

  it("should render a selection trigger", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    expect(c.getByText("Select Test Items")).toBeTruthy();
  });

  it("should open the selection dialog when the trigger is clicked", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    expect(c.getByText("First Item Option")).toBeTruthy();
    expect(c.getByText("Second Item Option")).toBeTruthy();
    expect(c.getByText("Third Item Option")).toBeTruthy();
  });

  it("should close the selection dialog when the user clicks on the trigger", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("Select Test Items"));
    expect(c.queryByText("First Item Option")).toBeNull();
  });

  it("should call onChange when an item is selected", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith(["1"]);
  });

  it("should not close the dialog when an item is selected", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(c.getByText("First Item Option")).toBeTruthy();
    expect(c.getByText("Second Item Option")).toBeTruthy();
    expect(c.getByText("Third Item Option")).toBeTruthy();
  });

  it("should allow the user to select multiple items", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("Second Item Option"));
    expect(onChange).toHaveBeenCalledWith(["1", "2"]);
  });

  it("should allow the user to deselect an item", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("should not allow the user to deselect to empty when allowNone is false", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple allowNone={false} />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith(["1"]);
  });

  it("should render a tag for each selected item", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("Second Item Option"));
    expect(c.getAllByText("First Item")).toHaveLength(1);
    expect(c.getAllByText("Second Item")).toHaveLength(1);
    expect(c.queryByText("Third Item")).toBeNull();
  });

  it("should allow the caller to deselect an item via its tag", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByLabelText("close"));
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("should allow the caller to search for an item", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.change(c.getByPlaceholderText("Search Test Items..."), {
      target: { value: "First" },
    });
    expect(c.getByText("First Item Option")).toBeTruthy();
    expect(c.queryByText("Second Item Option")).toBeNull();
    expect(c.queryByText("Third Item Option")).toBeNull();
  });

  it("should allow the caller to select multiple items with the shift key", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.keyDown(c.container, { code: "Shift" });
    fireEvent.click(c.getByText("Third Item Option"));
    expect(onChange).toHaveBeenLastCalledWith(["1", "2", "3"]);
  });

  describe("replaceOnSingle", () => {
    it("should replace the selection when the user selects a single item", () => {
      const { SelectMultiple, onChange } = createSelectMultiple();
      const c = render(<SelectMultiple replaceOnSingle />);
      fireEvent.click(c.getByText("Select Test Items"));
      fireEvent.click(c.getByText("First Item Option"));
      fireEvent.click(c.getByText("Second Item Option"));
      expect(onChange).toHaveBeenLastCalledWith(["2"]);
    });

    it("should add to the selection when the user has the control key pressed", () => {
      const { SelectMultiple, onChange } = createSelectMultiple();
      const c = render(<SelectMultiple replaceOnSingle />);
      fireEvent.click(c.getByText("Select Test Items"));
      fireEvent.click(c.getByText("First Item Option"));
      fireEvent.keyDown(c.container, { code: "Control" });
      fireEvent.click(c.getByText("Second Item Option"));
      expect(onChange).toHaveBeenLastCalledWith(["1", "2"]);
    });
  });
});
