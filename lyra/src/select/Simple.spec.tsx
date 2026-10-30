// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { createRoot, hydrateRoot } from "react-dom/client";
import { renderToString } from "react-dom/server";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { Icon } from "@/icon";
import { type Input } from "@/input";
import { List } from "@/list";
import { Select } from "@/select";
import { disableActEnvironment } from "@/testutil/act";
import { mockBoundingClientRect } from "@/testutil/dom";

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
    expect(c.getByRole("button", { name: "Test Item Third Item" }).textContent).toBe(
      "Third Item",
    );
  });

  it("should show a fixed item's label in a label that fades on overflow", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    fireEvent.click(c.getByText("Third Item"));
    const label = c
      .getByRole("button", { name: "Test Item Third Item" })
      .querySelector(".lyra-select__label");
    expect(label?.classList).toContain("lyra-text--overflow-fade");
    expect(label?.textContent).toBe("Third Item");
  });

  it("should name the trigger by its resource while nothing is selected", () => {
    const c = render(<SelectSimple />);
    expect(c.getByRole("button", { name: "Test Item" })).toBeTruthy();
  });

  it("should show empty content when the search hides every item", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    fireEvent.change(c.getByPlaceholderText("Search Test Items..."), {
      target: { value: "zzz" },
    });
    expect(c.getByText("No Test Items found")).toBeTruthy();
  });

  it("should name the search by searchName and the trigger by resourceName", () => {
    const c = render(
      <Select.Simple<string>
        value=""
        onChange={vi.fn()}
        resourceName="resistance configuration"
        searchName="configuration"
      >
        <Select.Item itemKey="1">First Item</Select.Item>
      </Select.Simple>,
    );
    fireEvent.click(c.getByLabelText("Resistance configuration"));
    expect(c.getByPlaceholderText("Search configurations...")).toBeTruthy();
  });

  it("should render string empty content as text", () => {
    const c = render(
      <Select.Simple<string>
        value=""
        onChange={vi.fn()}
        resourceName="page"
        emptyContent="No other pages"
      >
        {[]}
      </Select.Simple>,
    );
    fireEvent.click(c.getByLabelText("Page"));
    const empty = c.getByText("No other pages");
    expect(empty.tagName).toBe("P");
    expect(empty.querySelector("svg")).toBeNull();
  });

  it("should add a lowercase s to a resource name ending in an acronym", () => {
    const c = render(
      <Select.Simple<string> value="" onChange={vi.fn()} resourceName="PDO">
        <Select.Item itemKey="1">First Item</Select.Item>
      </Select.Simple>,
    );
    fireEvent.click(c.getByLabelText("PDO"));
    expect(c.getByPlaceholderText("Search PDOs...")).toBeTruthy();
  });

  it("should drop a trailing one-character identifier from the plural", () => {
    const c = render(
      <Select.Simple<string>
        value=""
        onChange={vi.fn()}
        resourceName="input terminal A"
      >
        <Select.Item itemKey="1">First Item</Select.Item>
      </Select.Simple>,
    );
    fireEvent.click(c.getByLabelText("Input terminal A"));
    expect(c.getByPlaceholderText("Search input terminals...")).toBeTruthy();
  });

  it("should clear the search when the dialog closes", () => {
    const c = render(<SelectSimple />);
    fireEvent.click(c.getByText("Test Item"));
    fireEvent.change(c.getByPlaceholderText("Search Test Items..."), {
      target: { value: "Second" },
    });
    fireEvent.click(c.getByText("Second Item"));
    fireEvent.click(c.getByRole("button", { name: "Test Item Second Item" }));
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
    expect(c.getByRole("button", { name: "Nested Bravo" }).textContent).toBe("Bravo");
  });

  describe("text value", () => {
    const Collapsible = () => {
      const [value, setValue] = useState("");
      return (
        <Select.Simple<string>
          value={value}
          onChange={setValue}
          resourceName="Level"
          triggerProps={{ collapsible: true }}
        >
          <Select.Item itemKey="high" textValue="High">
            <Icon.Add />
            <span>High</span>
          </Select.Item>
          <Select.Item itemKey="low">Low</Select.Item>
          <Select.Item itemKey="plus" textValue="Plus">
            <Icon.Add />
          </Select.Item>
        </Select.Simple>
      );
    };

    const collapse = (c: ReturnType<typeof render>, option: string): HTMLElement => {
      fireEvent.click(c.getByText("Level"));
      fireEvent.click(c.getByText(option));
      const trigger = c.getByRole("button", { name: `Level ${option}` });
      const label = trigger.querySelector<HTMLElement>(".lyra-select__label");
      if (label != null) label.style.display = "none";
      vi.useFakeTimers();
      fireEvent.pointerOver(trigger, { pointerType: "mouse" });
      act(() => {
        vi.advanceTimersByTime(2000);
      });
      vi.useRealTimers();
      return trigger;
    };

    const tooltip = (): string | null | undefined =>
      document.querySelector(".lyra-tooltip")?.textContent;

    it("should show an item's text value in a collapsed trigger's tooltip", () => {
      const c = render(<Collapsible />);
      collapse(c, "High");
      expect(tooltip()).toBe("High");
    });

    it("should default the text value to string children", () => {
      const c = render(<Collapsible />);
      collapse(c, "Low");
      expect(tooltip()).toBe("Low");
    });

    it("should match the search against the text value", () => {
      const c = render(<Collapsible />);
      fireEvent.click(c.getByText("Level"));
      fireEvent.change(c.getByPlaceholderText("Search Levels..."), {
        target: { value: "plus" },
      });
      const options = c.getAllByRole("option");
      expect(options.filter((o) => o.closest("[hidden]") == null)).toHaveLength(1);
      expect(c.getByText("High").closest("[hidden]")).not.toBeNull();
    });
  });

  describe("variant", () => {
    const trigger = (variant?: Input.Variant): HTMLElement => {
      const c = render(
        <Select.Simple<string>
          value="high"
          onChange={vi.fn()}
          resourceName="Level"
          triggerProps={{ variant }}
        >
          <Select.Item itemKey="high">High</Select.Item>
        </Select.Simple>,
      );
      return c.getByRole("button", { name: "Level High" });
    };

    it("should draw an outlined trigger by default", () => {
      const { classList } = trigger();
      expect(classList.contains("lyra-btn--outlined")).toBe(true);
      expect(classList.contains("lyra--shadow")).toBe(false);
    });

    it("should draw a shadow trigger on the outlined chassis", () => {
      const { classList } = trigger("shadow");
      expect(classList.contains("lyra-btn--outlined")).toBe(true);
      expect(classList.contains("lyra--shadow")).toBe(true);
    });

    it("should draw a text trigger without the shadow modifier", () => {
      const { classList } = trigger("text");
      expect(classList.contains("lyra-btn--text")).toBe(true);
      expect(classList.contains("lyra--shadow")).toBe(false);
    });
  });

  describe("server rendering", () => {
    it("should hydrate the server HTML of a selected fixed item", async () => {
      const container = document.createElement("div");
      container.innerHTML = renderToString(<Fixed />);
      document.body.appendChild(container);
      const onRecoverableError = vi.fn();
      const root = await act(async () =>
        hydrateRoot(container, <Fixed />, { onRecoverableError }),
      );
      expect(onRecoverableError).not.toHaveBeenCalled();
      const trigger = container.querySelector(".lyra-dialog__trigger");
      expect(trigger?.textContent).toBe("First Item");
      act(() => root.unmount());
      container.remove();
    });
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
    const tag = c.getByRole("button", { name: "Modes" }).querySelector(".lyra-tag");
    expect(tag?.textContent).toBe("Fast");
    expect(tag?.querySelector("[aria-label='trigger icon']")).toBeNull();
  });

  it("should clear a selected item with allowNone", () => {
    const onClear = vi.fn();
    const Clearable = (): ReactElement => {
      const [value, setValue] = useState<string | undefined>("1");
      const handleChange = (key: string | null) => {
        setValue(key ?? undefined);
        onClear(key);
      };
      return (
        <Select.Simple<string>
          value={value}
          onChange={handleChange}
          resourceName="Test Item"
          allowNone
        >
          <Select.Item itemKey="1">First Item</Select.Item>
        </Select.Simple>
      );
    };
    const c = render(<Clearable />);
    fireEvent.click(c.getByLabelText("Clear First Item"));
    expect(onClear).toHaveBeenLastCalledWith(null);
    expect(c.getByText("Test Item")).toBeTruthy();
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
        const trigger = container.querySelector(".lyra-dialog__trigger");
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

    it("should paint the selected fixed item in the trigger at once", async () => {
      expect(await paints(<Fixed />)).toEqual(["First Item"]);
    });

    it("should not paint the empty content when mounted open", async () => {
      expect(await paints(<Fixed initialVisible />)).toEqual(["First Item"]);
    });
  });
});
