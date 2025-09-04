// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
    { key: "4", name: "Fourth Item" },
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

  it("should open the selection dialog when the trigger is clicked", async () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    await userEvent.click(c.getByText("Select Test Items"));
    expect(c.getByText("First Item Option")).toBeTruthy();
    expect(c.getByText("Second Item Option")).toBeTruthy();
    expect(c.getByText("Third Item Option")).toBeTruthy();
  });

  it("should close the selection dialog when the user clicks on the trigger", async () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    await userEvent.click(c.getByText("Select Test Items"));
    await userEvent.click(c.getByText("Select Test Items"));
    expect(c.queryByText("First Item Option")).toBeNull();
  });

  it("should call onChange when an item is selected", async () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Select Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith(["1"]);
  });

  it("should not close the dialog when an item is selected", async () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    await userEvent.click(c.getByText("Select Test Items"));
    await userEvent.click(c.getByText("First Item Option"));
    expect(c.getByText("First Item Option")).toBeTruthy();
    expect(c.getByText("Second Item Option")).toBeTruthy();
    expect(c.getByText("Third Item Option")).toBeTruthy();
  });

  it("should allow the user to select multiple items", async () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    await userEvent.click(c.getByText("Select Test Items"));
    await userEvent.click(c.getByText("First Item Option"));
    await userEvent.click(c.getByText("Second Item Option"));
    expect(onChange).toHaveBeenCalledWith(["1", "2"]);
  });

  it("should allow the user to deselect an item", async () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    await userEvent.click(c.getByText("Select Test Items"));
    await userEvent.click(c.getByText("First Item Option"));
    await userEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("should not allow the user to deselect to empty when allowNone is false", async () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple allowNone={false} />);
    await userEvent.click(c.getByText("Select Test Items"));
    await userEvent.click(c.getByText("First Item Option"));
    await userEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith(["1"]);
  });

  it("should render a tag for each selected item", async () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    await userEvent.click(c.getByText("Select Test Items"));
    await userEvent.click(c.getByText("First Item Option"));
    await userEvent.click(c.getByText("Second Item Option"));
    expect(c.getAllByText("First Item")).toHaveLength(1);
    expect(c.getAllByText("Second Item")).toHaveLength(1);
    expect(c.queryByText("Third Item")).toBeNull();
  });

  it("should allow the caller to deselect an item via its tag", async () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    await userEvent.click(c.getByText("Select Test Items"));
    await userEvent.click(c.getByText("First Item Option"));
    await userEvent.click(c.getByLabelText("close"));
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("should allow the caller to search for an item", async () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    await userEvent.click(c.getByText("Select Test Items"));
    await userEvent.type(c.getByPlaceholderText("Search Test Items..."), "First");
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
    it("should replace the selection when the user selects a single item", async () => {
      const { SelectMultiple, onChange } = createSelectMultiple();
      const c = render(<SelectMultiple replaceOnSingle />);
      await userEvent.click(c.getByText("Select Test Items"));
      await userEvent.click(c.getByText("First Item Option"));
      await userEvent.click(c.getByText("Second Item Option"));
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

  describe("right click", () => {
    describe("multiple selected before", () => {
      it("should extend the selection when you right click even if replaceOnSingle is true", async () => {
        const { SelectMultiple, onChange } = createSelectMultiple();
        const c = render(<SelectMultiple replaceOnSingle />);
        await userEvent.click(c.getByText("Select Test Items"));
        await userEvent.click(c.getByText("First Item Option"));
        await userEvent.click(c.getByText("Second Item Option"));
        await userEvent.pointer([
          {
            target: c.getByText("Third Item Option"),
            keys: "[MouseRight]",
          },
        ]);
        expect(onChange).toHaveBeenLastCalledWith(["1", "2", "3"]);
      });

      it("should replace the previous right click selection when you right click again", async () => {
        const { SelectMultiple, onChange } = createSelectMultiple();
        const c = render(<SelectMultiple replaceOnSingle />);
        await userEvent.click(c.getByText("Select Test Items"));
        await userEvent.click(c.getByText("First Item Option"));
        await userEvent.pointer([
          { target: c.getByText("Second Item Option"), keys: "[MouseRight]" },
        ]);
        await userEvent.pointer([
          { target: c.getByText("Third Item Option"), keys: "[MouseRight]" },
        ]);
        expect(onChange).toHaveBeenLastCalledWith(["1", "3"]);
      });

      it("should keep existing selection when you right click on the same item", async () => {
        const { SelectMultiple, onChange } = createSelectMultiple();
        const c = render(<SelectMultiple replaceOnSingle />);
        await userEvent.click(c.getByText("Select Test Items"));
        await userEvent.click(c.getByText("First Item Option"));
        await userEvent.pointer([
          { target: c.getByText("Second Item Option"), keys: "[MouseRight]" },
        ]);
        expect(onChange).toHaveBeenLastCalledWith(["1", "2"]);
        await userEvent.pointer([
          { target: c.getByText("Second Item Option"), keys: "[MouseRight]" },
        ]);
        expect(onChange).toHaveBeenLastCalledWith(["1", "2"]);
      });
    });
  });
});
