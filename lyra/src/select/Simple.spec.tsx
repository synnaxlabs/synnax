// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { createRoot } from "react-dom/client";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { Icon } from "@/icon";
import { List } from "@/list";
import { Select } from "@/select";
import { disableActEnvironment } from "@/testutil/act";
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

  it("should show a fixed label when nested in another list's row", () => {
    const c = render(
      <List.Frame data={["row"]}>
        <List.Scroll>
          <List.Items>
            {({ key }) => (
              <Select.Simple<string>
                key={key}
                value="b"
                onChange={vi.fn()}
                resourceName="Nested"
              >
                <Select.Item itemKey="a">Alpha</Select.Item>
                <Select.Item itemKey="b">Bravo</Select.Item>
              </Select.Simple>
            )}
          </List.Items>
        </List.Scroll>
      </List.Frame>,
    );
    expect(c.getByRole("button", { name: "Nested" }).textContent).toBe("Bravo");
  });
});

describe("Select.Simple multiple", () => {
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });
  const SelectMultiple = ({ onChange }: { onChange: (keys: string[]) => void }) => {
    const [value, setValue] = useState<string[]>([]);
    const handleChange = (keys: string[]) => {
      setValue(keys);
      onChange(keys);
    };
    return (
      <Select.Simple<string>
        multiple
        value={value}
        onChange={handleChange}
        icon={<Icon.Add aria-label="trigger icon" />}
        resourceName="mode"
      >
        <Select.Item itemKey="fast">Fast</Select.Item>
        <Select.Item itemKey="slow">Slow</Select.Item>
      </Select.Simple>
    );
  };

  it("should show the plural resource name as the placeholder", () => {
    const c = render(<SelectMultiple onChange={vi.fn()} />);
    expect(c.getByText("Modes")).toBeTruthy();
  });

  it("should select several items without closing the dialog", () => {
    const onChange = vi.fn();
    const c = render(<SelectMultiple onChange={onChange} />);
    fireEvent.click(c.getByText("Modes"));
    fireEvent.click(c.getByText("Fast"));
    fireEvent.click(c.getByText("Slow"));
    expect(onChange).toHaveBeenLastCalledWith(["fast", "slow"]);
  });

  it("should show each selected item as a tag without the trigger icon", () => {
    const c = render(<SelectMultiple onChange={vi.fn()} />);
    fireEvent.click(c.getByText("Modes"));
    fireEvent.click(c.getByText("Fast"));
    const tag = c.getByRole("button", { name: "Modes" }).querySelector(".pluto-tag");
    expect(tag?.textContent).toBe("Fast");
    expect(tag?.querySelector("[aria-label='trigger icon']")).toBeNull();
  });

  describe("first paint", () => {
    // act runs every effect before returning, which hides what a browser would paint.
    // These specs mount outside it and read the page after each task, where a browser
    // may paint.
    beforeEach(disableActEnvironment);

    const paints = async (element: ReactElement): Promise<string[]> => {
      const container = document.createElement("div");
      document.body.appendChild(container);
      const seen: string[] = [];
      const read = (): string => {
        const trigger = container.querySelector("[aria-label='Test Item']");
        const empty = document.body.textContent?.includes("No Test Items found");
        return `${trigger?.textContent}${empty === true ? " + empty" : ""}`;
      };
      const observer = new MutationObserver(() => seen.push(read()));
      observer.observe(document.body, {
        subtree: true,
        childList: true,
        characterData: true,
      });
      const root = createRoot(container);
      root.render(element);
      await new Promise((resolve) => setTimeout(resolve, 50));
      observer.disconnect();
      root.unmount();
      container.remove();
      return seen;
    };

    const Fixed = ({ initialVisible }: { initialVisible?: boolean }) => (
      <Select.Simple<string>
        value="1"
        onChange={vi.fn()}
        resourceName="Test Item"
        initialVisible={initialVisible}
      >
        <Select.Item itemKey="1">First Item</Select.Item>
        <Select.Item itemKey="2">Second Item</Select.Item>
      </Select.Simple>
    );

    it("should paint the selected fixed item in the trigger at once", async () => {
      expect(await paints(<Fixed />)).toEqual(["First Item"]);
    });

    it("should not paint the empty content when mounted open", async () => {
      expect(await paints(<Fixed initialVisible />)).toEqual(["First Item"]);
    });
  });
});
