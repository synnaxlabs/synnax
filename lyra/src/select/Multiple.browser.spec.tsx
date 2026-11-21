// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, type RenderResult } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { renderProp } from "@/component/renderProp";
import { Haul } from "@/haul";
import { Input } from "@/input";
import { List } from "@/list";
import type { Menu } from "@/menu";
import { Select } from "@/select";
import { mockGeometry } from "@/testutil/dom";
import { Text } from "@/text";
import { Triggers } from "@/triggers";

describe("Select.Multiple", () => {
  beforeAll(() => mockGeometry(100, 100));
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
      <Select.Item {...props}>
        <Text.Text>{item?.name} Option</Text.Text>
      </Select.Item>
    );
  });

  const createSelectMultiple = (
    initialValue: string[] = [],
    /** Replaces the keys of the data, as a list that has not loaded every page. */
    dataKeys?: string[],
  ) => {
    const onChange = vi.fn();
    const SelectMultiple = (
      props: Omit<
        Select.MultipleProps<string, TestEntry>,
        "data" | "value" | "onChange" | "children" | "resourceName"
      >,
    ) => {
      const { data, getItem, search } = List.useStaticData<string, TestEntry>({
        data: testData,
      });
      const [value, setValue] = useState<string[]>(initialValue);
      const handleChange = (keys: string[]) => {
        setValue(keys);
        onChange(keys);
      };
      return (
        <Triggers.Provider>
          <Select.Multiple<string, TestEntry>
            getItem={getItem}
            data={dataKeys ?? data}
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
    expect(c.getByText("Test Items")).toBeTruthy();
  });

  it("should select the first entry when autoSelectOnNone is set", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    render(<SelectMultiple autoSelectOnNone virtual={false} />);
    expect(onChange).toHaveBeenCalledWith(["1"]);
  });

  it("should open the selection dialog when the trigger is clicked", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    expect(c.getByText("First Item Option")).toBeTruthy();
    expect(c.getByText("Second Item Option")).toBeTruthy();
    expect(c.getByText("Third Item Option")).toBeTruthy();
  });

  it("should close the selection dialog when the user clicks on the trigger", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("Test Items"));
    expect(c.queryByText("First Item Option")).toBeNull();
  });

  it("should call onChange when an item is selected", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith(["1"]);
  });

  it("should not close the dialog when an item is selected", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(c.getByText("First Item Option")).toBeTruthy();
    expect(c.getByText("Second Item Option")).toBeTruthy();
    expect(c.getByText("Third Item Option")).toBeTruthy();
  });

  it("should allow the user to select multiple items", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("Second Item Option"));
    expect(onChange).toHaveBeenCalledWith(["1", "2"]);
  });

  it("should allow the user to deselect an item", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("should not allow the user to deselect to empty when allowNone is false", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple allowNone={false} />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(onChange).toHaveBeenCalledWith(["1"]);
  });

  it("should render a tag for each selected item", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("Second Item Option"));
    expect(c.getAllByText("First Item")).toHaveLength(1);
    expect(c.getAllByText("Second Item")).toHaveLength(1);
    expect(c.queryByText("Third Item")).toBeNull();
  });

  it("should allow the caller to deselect an item via its tag", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByLabelText("Remove First Item"));
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("should not remove the last tag when allowNone is false", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple allowNone={false} />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByLabelText("Remove First Item"));
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenLastCalledWith(["1"]);
  });

  it("should remove only the tag's item when replaceOnSingle is set", () => {
    const { SelectMultiple, onChange } = createSelectMultiple();
    const c = render(<SelectMultiple replaceOnSingle />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.keyDown(c.container, { code: "Control" });
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("Second Item Option"));
    fireEvent.keyUp(c.container, { code: "Control" });
    expect(onChange).toHaveBeenLastCalledWith(["1", "2"]);
    fireEvent.click(c.getByLabelText("Remove First Item"));
    expect(onChange).toHaveBeenLastCalledWith(["2"]);
  });

  it("should toggle the selection dialog when the user clicks a tag", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.click(c.getByText("First Item"));
    expect(c.queryByText("Second Item Option")).toBeNull();
    fireEvent.click(c.getByText("First Item"));
    expect(c.getByText("Second Item Option")).toBeTruthy();
  });

  it("should allow the caller to search for an item", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
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
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    fireEvent.keyDown(c.container, { code: "Shift" });
    fireEvent.click(c.getByText("Third Item Option"));
    expect(onChange).toHaveBeenLastCalledWith(["1", "2", "3"]);
  });

  it("should select a shift range that spans unmounted rows", () => {
    const onChange = vi.fn();
    const data: TestEntry[] = Array.from({ length: 300 }, (_, i) => ({
      key: `${i}`,
      name: `Item ${i}`,
    }));
    const renderItem = renderProp((props: List.ItemProps<string>) => (
      <Select.Item {...props}>
        <Text.Text>Row {props.itemKey}</Text.Text>
      </Select.Item>
    ));
    const Component = () => {
      const { data: keys, getItem } = List.useStaticData<string, TestEntry>({ data });
      const [value, setValue] = useState<string[]>([]);
      return (
        <Triggers.Provider>
          <Select.Multiple<string, TestEntry>
            getItem={getItem}
            data={keys}
            value={value}
            onChange={(next) => {
              setValue(next);
              onChange(next);
            }}
            resourceName="Row"
          >
            {renderItem}
          </Select.Multiple>
        </Triggers.Provider>
      );
    };
    const c = render(<Component />);
    fireEvent.click(c.getByText("Rows"));
    fireEvent.click(c.getByText("Row 0"));
    // Far enough down that the anchor row has unmounted by the time the range closes.
    const scroller = document.querySelector<HTMLElement>(".lyra-list__scroll");
    scroller!.scrollTop = 200 * 33;
    fireEvent.scroll(scroller!);
    fireEvent.keyDown(c.container, { code: "Shift" });
    expect(c.queryByText("Row 0")).toBeNull();
    fireEvent.click(c.getByText("Row 205"));
    expect(onChange).toHaveBeenLastCalledWith(
      Array.from({ length: 206 }, (_, i) => `${i}`),
    );
  });

  it("should name the trigger by its resource", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    expect(c.getByRole("button", { name: "Test Items" })).toBeTruthy();
  });

  it("should give the trigger the class its caller passes", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple triggerProps={{ className: "custom" }} />);
    expect(c.getByRole("button", { name: "Test Items" }).classList).toContain("custom");
  });

  it("should give a tagless trigger the class its caller passes", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(
      <SelectMultiple triggerProps={{ className: "custom", hideTags: true }} />,
    );
    expect(c.getByRole("button", { name: "Test Items" }).classList).toContain("custom");
  });

  it("should open the selection dialog from the keyboard", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    const trigger = c.getByRole("button", { name: "Test Items" });
    expect(trigger.tabIndex).toBe(0);
    fireEvent.keyDown(trigger, { key: "Enter" });
    expect(c.getByText("First Item Option")).toBeTruthy();
  });

  it("should prompt a selection when a visible label names the field", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(
      <Input.Item label="Items">
        <SelectMultiple />
      </Input.Item>,
    );
    expect(c.getByText("Select Test Items")).toBeTruthy();
  });

  it("should make the sole selected item inert when allowNone is false", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple allowNone={false} />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(c.getByRole("option", { name: "First Item Option" }).classList).toContain(
      "lyra-btn--prevent-click",
    );
  });

  it("should keep the sole selected item clickable when allowNone is true", () => {
    const { SelectMultiple } = createSelectMultiple();
    const c = render(<SelectMultiple />);
    fireEvent.click(c.getByText("Test Items"));
    fireEvent.click(c.getByText("First Item Option"));
    expect(
      c.getByRole("option", { name: "First Item Option" }).classList,
    ).not.toContain("lyra-btn--prevent-click");
  });

  describe("drag preview", () => {
    const dragTag = (props: Partial<Select.MultipleProps<string, TestEntry>> = {}) => {
      const { SelectMultiple } = createSelectMultiple(["1"]);
      const c = render(
        <Haul.Provider>
          <SelectMultiple haulType="test-item" {...props} />
        </Haul.Provider>,
      );
      fireEvent.dragStart(c.getByText("First Item"), {
        dataTransfer: new DataTransfer(),
      });
      return document.querySelector(".lyra-haul__layer")?.textContent;
    };

    it("should picture a dragged tag by its name", () => {
      expect(dragTag()).toBe("First Item");
    });

    it("should picture a dragged tag with the preview that its trigger renders", () => {
      const renderHaulPreview = (entry: unknown) =>
        `Custom ${(entry as TestEntry).name}`;
      expect(dragTag({ triggerProps: { renderHaulPreview } })).toBe(
        "Custom First Item",
      );
    });
  });

  describe("selected first", () => {
    const options = (c: RenderResult): string[] =>
      c.getAllByRole("option").map((option) => option.textContent ?? "");
    const toggle = (c: RenderResult): void => {
      fireEvent.click(c.getByRole("button", { name: "Test Items" }));
    };

    it("should list the selected items first when the dialog opens", () => {
      const { SelectMultiple } = createSelectMultiple(["3"]);
      const c = render(<SelectMultiple />);
      toggle(c);
      expect(options(c)).toEqual([
        "Third Item Option",
        "First Item Option",
        "Second Item Option",
      ]);
    });

    it("should keep the order while the dialog stays open", () => {
      const { SelectMultiple, onChange } = createSelectMultiple(["3"]);
      const c = render(<SelectMultiple />);
      toggle(c);
      fireEvent.click(c.getByText("Second Item Option"));
      fireEvent.click(c.getByText("Third Item Option"));
      expect(onChange).toHaveBeenLastCalledWith(["2"]);
      expect(options(c)).toEqual([
        "Third Item Option",
        "First Item Option",
        "Second Item Option",
      ]);
    });

    it("should list the new selection first when the dialog opens again", () => {
      const { SelectMultiple } = createSelectMultiple();
      const c = render(<SelectMultiple />);
      toggle(c);
      fireEvent.click(c.getByText("Second Item Option"));
      toggle(c);
      toggle(c);
      expect(options(c)).toEqual([
        "Second Item Option",
        "First Item Option",
        "Third Item Option",
      ]);
    });

    it("should list the matches of a search in their own order", () => {
      const { SelectMultiple } = createSelectMultiple(["3"]);
      const c = render(<SelectMultiple />);
      toggle(c);
      const search = c.getByPlaceholderText("Search Test Items...");
      fireEvent.change(search, { target: { value: "Item" } });
      expect(options(c)).toEqual([
        "First Item Option",
        "Second Item Option",
        "Third Item Option",
      ]);
      fireEvent.change(search, { target: { value: "" } });
      expect(options(c)).toEqual([
        "Third Item Option",
        "First Item Option",
        "Second Item Option",
      ]);
    });

    it("should leave out a selected key that names no item", () => {
      const { SelectMultiple } = createSelectMultiple(["deleted"]);
      const c = render(<SelectMultiple />);
      toggle(c);
      expect(options(c)).toEqual([
        "First Item Option",
        "Second Item Option",
        "Third Item Option",
      ]);
    });

    it("should treat a search of only spaces as a search", () => {
      const { SelectMultiple } = createSelectMultiple(["3"]);
      const c = render(<SelectMultiple />);
      toggle(c);
      fireEvent.change(c.getByPlaceholderText("Search Test Items..."), {
        target: { value: " " },
      });
      expect(c.queryAllByRole("option")).toHaveLength(0);
    });

    it("should list a selected item that the data has not loaded", () => {
      const { SelectMultiple } = createSelectMultiple(["3"], ["1", "2"]);
      const c = render(<SelectMultiple />);
      toggle(c);
      expect(options(c)).toEqual([
        "Third Item Option",
        "First Item Option",
        "Second Item Option",
      ]);
    });
  });

  describe("replaceOnSingle", () => {
    it("should replace the selection when the user selects a single item", () => {
      const { SelectMultiple, onChange } = createSelectMultiple();
      const c = render(<SelectMultiple replaceOnSingle />);
      fireEvent.click(c.getByText("Test Items"));
      fireEvent.click(c.getByText("First Item Option"));
      fireEvent.click(c.getByText("Second Item Option"));
      expect(onChange).toHaveBeenLastCalledWith(["2"]);
    });

    it("should add to the selection when the user has the control key pressed", () => {
      const { SelectMultiple, onChange } = createSelectMultiple();
      const c = render(<SelectMultiple replaceOnSingle />);
      fireEvent.click(c.getByText("Test Items"));
      fireEvent.click(c.getByText("First Item Option"));
      fireEvent.keyDown(c.container, { code: "Control" });
      fireEvent.click(c.getByText("Second Item Option"));
      expect(onChange).toHaveBeenLastCalledWith(["1", "2"]);
    });
  });

  describe("option menu", () => {
    it("should give the menu only the option it opened on among selected ones", () => {
      const menu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => (
        <span>{`menu for ${keys.join(",")}`}</span>
      );
      const { SelectMultiple } = createSelectMultiple(["1", "2"]);
      const c = render(<SelectMultiple menu={menu} virtual={false} />);
      fireEvent.click(c.getByRole("button", { name: "Test Items" }));
      fireEvent.contextMenu(c.getByRole("option", { name: "First Item Option" }));
      expect(c.getByText("menu for 1")).toBeTruthy();
    });
  });
});
