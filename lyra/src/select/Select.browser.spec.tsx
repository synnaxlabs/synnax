// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";
import { assert, describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { Select } from "@/select";
import { computed, renderThemed, resolve } from "@/styletest";
import { Text } from "@/text";
import { Triggers } from "@/triggers";

const TWO_LINE_KEYS = ["1", "2", "3", "4", "5", "6"];

const renderTwoLineOptions = async (): Promise<Element[]> => {
  const { getByRole } = await renderThemed(
    <Select.Simple<string>
      resourceName="endpoint"
      value="1"
      onChange={() => {}}
      initialVisible
    >
      {TWO_LINE_KEYS.map((key) => (
        <Select.Item key={key} itemKey={key} textValue={`Endpoint ${key}`} y>
          <Text.Text>Endpoint {key}</Text.Text>
          <Text.Text level="small">Basic256Sha256, sign and encrypt</Text.Text>
        </Select.Item>
      ))}
    </Select.Simple>,
  );
  // The dialog caps its height once it is placed, and it shows only then.
  const dialog = getByRole("dialog").element();
  await expect.poll(() => computed(dialog, "opacity")).toBe("1");
  return getByRole("option").elements();
};

const renderButtons = async (
  variant: Select.ButtonsProps["variant"],
): Promise<(name: string) => Element> => {
  const { getByRole } = await renderThemed(
    <Select.Buttons value={1} onChange={() => {}} variant={variant}>
      <Select.Item itemKey={1}>One</Select.Item>
      <Select.Item itemKey={2}>A much longer second option</Select.Item>
    </Select.Buttons>,
  );
  return (name) => getByRole("button", { name }).element();
};

const renderFixedLabel = async (): Promise<Record<string, Element>> => {
  const { container } = await renderThemed(
    <Dialog.Frame>
      <Select.Frame<string, undefined> value="time" onChange={() => {}}>
        <Select.SingleTrigger placeholder="Pick" />
        <Select.Dialog>
          <Select.Item itemKey="time">Time</Select.Item>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>,
  );
  const get = (selector: string): Element =>
    container.querySelector(selector) as Element;
  return {
    label: get(".lyra-select__label"),
    fixed: get(".lyra-select__fixed-label"),
    caret: get(".lyra-caret-animated"),
  };
};

const renderEndContent = async (width: string): Promise<Record<string, Element>> => {
  const { container } = await renderThemed(
    <div style={{ width }}>
      <Dialog.Frame>
        <Select.Frame<string, undefined> value="time" onChange={() => {}}>
          <Select.SingleTrigger
            placeholder="Pick"
            collapsible
            renderEndContent={() => "Connected"}
          />
          <Select.Dialog>
            <Select.Item itemKey="time">Time</Select.Item>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </div>,
  );
  const get = (selector: string): Element =>
    container.querySelector(selector) as Element;
  return {
    trigger: get(".lyra-dialog__trigger"),
    label: get(".lyra-select__label"),
    end: get(".lyra-select-single-trigger__end"),
    caret: get(".lyra-caret-animated"),
  };
};

const CHECKABLE_COLUMNS: List.Column[] = [
  { name: "Type", width: "20rem" },
  { name: "Channel", width: "20rem" },
];

const renderCheckable = async (
  pin?: ReactElement,
): Promise<(text: string) => Element> => {
  const { getByText } = await renderThemed(
    <Triggers.Provider>
      <List.Columns columns={CHECKABLE_COLUMNS}>
        <List.Headings checkable />
        <Select.Frame<string, undefined>
          multiple
          data={["1"]}
          value={["1"]}
          onChange={() => {}}
        >
          <List.Scroll>
            <List.Items<string>>
              {({ key, ...p }) => (
                <Select.Item key={key} {...p} checkable>
                  <List.Cells>
                    <List.Cell>type-{key}</List.Cell>
                    <List.Cell>channel-{key}</List.Cell>
                  </List.Cells>
                  {pin}
                </Select.Item>
              )}
            </List.Items>
          </List.Scroll>
        </Select.Frame>
      </List.Columns>
    </Triggers.Provider>,
  );
  return (text) => getByText(text, { exact: true }).element();
};

const FOOTER_HEIGHT = 100;
const KEYS = Array.from({ length: 30 }, (_, i) => `key-${i}`);

const renderFooter = async (): Promise<Element> => {
  const { getByText } = await renderThemed(
    <Triggers.Provider>
      <Dialog.Frame>
        <Select.Frame<string, undefined> data={KEYS} onChange={() => {}}>
          <Select.SingleTrigger placeholder="Pick" />
          <Select.Dialog>
            <Select.Body<string>
              resourceName="key"
              footer={<div id="footer" style={{ height: FOOTER_HEIGHT }} />}
            >
              {({ key, ...p }) => (
                <Select.Item key={key} {...p}>
                  {p.itemKey}
                </Select.Item>
              )}
            </Select.Body>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Triggers.Provider>,
  );
  await userEvent.click(getByText("Pick").element());
  const dialog = page.getByRole("dialog");
  await expect.element(dialog).toBeVisible();
  await expect.poll(() => computed(dialog.element(), "opacity")).toBe("1");
  return document.getElementById("footer") as Element;
};

const DESCRIPTION = "Wider than the minimum width of a select dialog";
const TRIGGER_WIDTH = 100;

interface DescribedOption {
  option: Element;
  icon: DOMRect;
  name: DOMRect;
  description: Element;
}

const rangeOf = (node: Node): Range => {
  const range = document.createRange();
  range.selectNodeContents(node);
  return range;
};

const renderDescribedOptions = async (
  searchHidden = true,
): Promise<DescribedOption[]> => {
  const { getByRole } = await renderThemed(
    <div style={{ width: TRIGGER_WIDTH, marginLeft: "auto" }}>
      <Select.Simple<string>
        resourceName="level"
        value="low"
        onChange={() => {}}
        variant="floating"
        searchHidden={searchHidden}
        initialVisible
      >
        {["low", "high"].map((key) => (
          <Select.Item key={key} itemKey={key} label={key} textValue={key}>
            <Icon.Auto />
            {key}
            <Select.ItemDescription>{DESCRIPTION}</Select.ItemDescription>
          </Select.Item>
        ))}
      </Select.Simple>
    </div>,
  );
  const dialog = getByRole("dialog").element();
  await expect.poll(() => computed(dialog, "opacity")).toBe("1");
  return getByRole("option")
    .elements()
    .map((option) => {
      const name = Array.from(option.childNodes).find(
        (node) => node.nodeType === Node.TEXT_NODE,
      );
      if (name == null) throw new Error("option name not found");
      return {
        option,
        icon: option.querySelector(".lyra-icon")!.getBoundingClientRect(),
        name: rangeOf(name).getBoundingClientRect(),
        description: option.querySelector(".lyra-select__item-description")!,
      };
    });
};

const LEVELS = [
  <Select.Item key="low" itemKey="low">
    Low
  </Select.Item>,
  <Select.Item key="high" itemKey="high">
    High
  </Select.Item>,
];

/** Opens a select from the keyboard, inside a modal. */
const openInModal = async (multiple = false): Promise<HTMLElement> => {
  await renderThemed(
    <Triggers.Provider>
      <Dialog.Frame variant="modal" initialVisible>
        <Dialog.Dialog>
          {multiple ? (
            <Select.Simple<string>
              resourceName="level"
              multiple
              value={["low"]}
              onChange={() => {}}
            >
              {LEVELS}
            </Select.Simple>
          ) : (
            <Select.Simple<string> resourceName="level" value="low" onChange={() => {}}>
              {LEVELS}
            </Select.Simple>
          )}
        </Dialog.Dialog>
      </Dialog.Frame>
    </Triggers.Provider>,
  );
  const trigger = page.getByRole("button", { name: /^Level/ }).element();
  assert(trigger instanceof HTMLElement);
  trigger.focus();
  await userEvent.keyboard("{Enter}");
  await expect.element(page.getByPlaceholder("Search levels...")).toHaveFocus();
  return trigger;
};

const TextTags = (): ReactElement => {
  const [value, setValue] = useState(["low"]);
  return (
    <Dialog.Frame>
      <Select.Frame<string, undefined> multiple value={value} onChange={setValue}>
        <Select.MultipleTrigger variant="text" aria-label="Levels" />
        <Select.Dialog>{LEVELS}</Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

const ShadowTags = ({ value }: { value: string[] }): ReactElement => (
  <List.Item itemKey="row">
    <Dialog.Frame>
      <Select.Frame<string, undefined> multiple value={value} onChange={() => {}}>
        <Select.MultipleTrigger variant="shadow" aria-label="Levels" />
        <Select.Dialog>{LEVELS}</Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  </List.Item>
);

const renderSearchless = async (
  multiple: boolean,
): Promise<ReturnType<typeof vi.fn>> => {
  const onChange = vi.fn();
  const items = ["low", "medium", "high"].map((key) => (
    <Select.Item key={key} itemKey={key}>
      {key}
    </Select.Item>
  ));
  const { getByRole } = await renderThemed(
    <Triggers.Provider>
      {multiple ? (
        <Select.Simple<string>
          resourceName="level"
          multiple
          value={["low"]}
          onChange={onChange}
          searchHidden
        >
          {items}
        </Select.Simple>
      ) : (
        <Select.Simple<string>
          resourceName="level"
          value="low"
          onChange={onChange}
          allowNone={false}
          searchHidden
        >
          {items}
        </Select.Simple>
      )}
    </Triggers.Provider>,
  );
  await userEvent.click(getByRole("button", { name: multiple ? "Levels" : /^Level/ }));
  const dialog = getByRole("dialog").element();
  await expect.poll(() => computed(dialog, "opacity")).toBe("1");
  await userEvent.keyboard("{ArrowDown}{ArrowDown}{Enter}");
  await expect.poll(() => onChange).toHaveBeenCalledOnce();
  return onChange;
};

describe("Select", () => {
  describe("described options", () => {
    it("should put the icon beside the name and the description under it", async () => {
      const [{ icon, name, description }] = await renderDescribedOptions();
      const line = description.getBoundingClientRect();
      expect(icon.right).toBeLessThan(name.left);
      expect(line.left).toBeCloseTo(name.left, 0);
      expect(line.top).toBeGreaterThanOrEqual(name.bottom);
    });

    it("should fit each description on one line", async () => {
      const options = await renderDescribedOptions();
      options.forEach(({ description }) =>
        expect(rangeOf(description).getClientRects()).toHaveLength(1),
      );
    });

    it("should space the options apart", async () => {
      const [first, second] = await renderDescribedOptions();
      expect(second.option.getBoundingClientRect().top).toBeGreaterThan(
        first.option.getBoundingClientRect().bottom,
      );
    });

    it("should hide the options a search leaves out, with no space above the rest", async () => {
      const [low, high] = await renderDescribedOptions(false);
      await userEvent.fill(page.getByPlaceholder("Search levels..."), "high");
      await expect.poll(() => computed(low.option, "display")).toBe("none");
      expect(computed(high.option, "margin-top")).toBe("0px");
    });
  });

  describe("searchless options", () => {
    it("should close and stay closed when Enter picks an option", async () => {
      const onChange = await renderSearchless(false);
      expect(onChange.mock.calls[0][0]).toBe("medium");
      await expect
        .element(page.getByRole("button", { name: /^Level/ }))
        .toHaveAttribute("aria-expanded", "false");
    });

    it("should close when Tab moves the focus on from its trigger", async () => {
      const onChange = vi.fn();
      await renderThemed(
        <Triggers.Provider>
          <Select.Simple<string>
            resourceName="level"
            value="low"
            onChange={onChange}
            allowNone={false}
            searchHidden
          >
            {LEVELS}
          </Select.Simple>
          <input aria-label="After" />
        </Triggers.Provider>,
      );
      const trigger = page.getByRole("button", { name: /^Level/ });
      trigger.element().focus();
      await userEvent.keyboard("{Enter}");
      await expect.element(page.getByRole("dialog")).toBeVisible();
      await userEvent.tab();
      await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
      await userEvent.keyboard("{ArrowDown}{Enter}");
      expect(onChange).not.toHaveBeenCalled();
      await expect.element(trigger).toHaveAttribute("aria-expanded", "false");
    });

    it("should stay open when Enter adds one of many options", async () => {
      const onChange = await renderSearchless(true);
      expect(onChange.mock.calls[0][0]).toEqual(["low", "medium"]);
      await expect.element(page.getByRole("dialog")).toBeVisible();
    });
  });

  describe("in a modal", () => {
    it("should focus the trigger when Enter picks an option", async () => {
      const trigger = await openInModal();
      await userEvent.keyboard("{ArrowDown}{Enter}");
      await expect
        .element(page.getByPlaceholder("Search levels..."))
        .not.toBeInTheDocument();
      expect(document.activeElement).toBe(trigger);
    });

    it("should focus the trigger of a multiple select on Escape", async () => {
      const trigger = await openInModal(true);
      await userEvent.keyboard("{Escape}");
      await expect
        .element(page.getByPlaceholder("Search levels..."))
        .not.toBeInTheDocument();
      expect(document.activeElement).toBe(trigger);
    });

    it("should focus the trigger when a click picks an option", async () => {
      const trigger = await openInModal();
      await userEvent.click(page.getByRole("option", { name: "High" }));
      await expect
        .element(page.getByPlaceholder("Search levels..."))
        .not.toBeInTheDocument();
      expect(document.activeElement).toBe(trigger);
    });
  });

  describe("text multiple trigger", () => {
    it("should get the focus back once its last tag goes", async () => {
      const { container } = await renderThemed(
        <Triggers.Provider>
          <TextTags />
        </Triggers.Provider>,
      );
      const add = Array.from(container.querySelectorAll("button")).at(-1);
      assert(add instanceof HTMLElement);
      await userEvent.click(add);
      await userEvent.click(page.getByRole("option", { name: "Low" }));
      await userEvent.keyboard("{Escape}");
      await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
      expect(document.activeElement).toBe(
        page.getByRole("button", { name: "Levels" }).element(),
      );
    });
  });

  describe("shadow multiple trigger", () => {
    it("should drop the remove buttons of its tags", async () => {
      const { container } = await renderThemed(<ShadowTags value={["low"]} />);
      const remove = container.querySelector(".lyra-tag__close");
      assert(remove != null);
      expect(computed(remove, "display")).toBe("none");
    });

    it("should show its placeholder in a list item on hover only", async () => {
      const { container } = await renderThemed(<ShadowTags value={[]} />);
      const item = container.querySelector(".lyra-list__item");
      const placeholder = container.querySelector(
        ".lyra-select-multiple-trigger-placeholder",
      );
      assert(item != null && placeholder != null);
      expect(computed(placeholder, "opacity")).toBe("0");
      await userEvent.hover(item);
      expect(computed(placeholder, "opacity")).toBe("1");
    });
  });

  describe("footer", () => {
    it("should keep its height when the list overflows the dialog", async () => {
      const footer = await renderFooter();
      expect(footer.getBoundingClientRect().height).toBe(FOOTER_HEIGHT);
    });
  });

  describe("fixed item label", () => {
    it("should fill the trigger up to the caret", async () => {
      const { label, fixed, caret } = await renderFixedLabel();
      expect(fixed.getBoundingClientRect().width).toBe(
        label.getBoundingClientRect().width,
      );
      expect(computed(caret, "margin-left")).toBe("0px");
    });
  });

  describe("end content", () => {
    it("should sit right before the caret", async () => {
      const { trigger, label, end, caret } = await renderEndContent("400px");
      const gap = parseFloat(computed(trigger, "column-gap"));
      expect(
        end.getBoundingClientRect().left - label.getBoundingClientRect().right,
      ).toBeCloseTo(gap, 0);
      expect(computed(caret, "margin-left")).toBe("0px");
    });

    it("should hide in a collapsed trigger", async () => {
      const { end } = await renderEndContent("100px");
      expect(computed(end, "display")).toBe("none");
    });
  });

  describe("text buttons", () => {
    it("should mark the chosen button in the subtle selection tier", async () => {
      const button = await renderButtons("text");
      const chosen = button("One");
      expect(computed(chosen, "background-color")).toBe(
        resolve("var(--pluto-selected-subtle-bg)"),
      );
      expect(computed(chosen, "color")).toBe(
        resolve("var(--pluto-selected-subtle-color)"),
      );
      expect(computed(chosen, "box-shadow")).toContain(
        resolve("var(--pluto-selected-subtle-border-color)"),
      );
    });

    it("should shade the other buttons", async () => {
      const button = await renderButtons("text");
      expect(computed(button("A much longer second option"), "color")).toBe(
        resolve("var(--pluto-gray-l9)"),
      );
    });
  });

  describe("cards", () => {
    it("should share the width equally", async () => {
      const button = await renderButtons("card");
      expect(button("One").getBoundingClientRect().width).toBe(
        button("A much longer second option").getBoundingClientRect().width,
      );
    });

    it("should rest an unchosen card half a gray step above the page", async () => {
      const button = await renderButtons("card");
      const card = button("A much longer second option");
      expect(computed(card, "background-color")).toBe(
        resolve("var(--pluto-gray-l1-l2)"),
      );
      expect(computed(card, "border-top-color")).toBe(resolve("var(--pluto-gray-l6)"));
    });
  });

  describe("fixed options", () => {
    it("should keep each option as tall as its content", async () => {
      const options = await renderTwoLineOptions();
      expect(options).toHaveLength(TWO_LINE_KEYS.length);
      options.forEach((option) =>
        expect(option.clientHeight).toBe(option.scrollHeight),
      );
    });

    it("should scroll the options that do not fit the dialog", async () => {
      const options = await renderTwoLineOptions();
      const last = options[options.length - 1];
      const scroll = last.closest(".lyra-list__scroll")!;
      const bottom = (el: Element): number => el.getBoundingClientRect().bottom;
      expect(bottom(last)).toBeGreaterThan(bottom(scroll));
      await userEvent.wheel(scroll, { delta: { y: 1000 } });
      await expect.poll(() => bottom(last)).toBeLessThanOrEqual(bottom(scroll));
    });

    it("should show the new options when they shrink under a scrolled list", async () => {
      const keys = (prefix: string, count: number): string[] =>
        Array.from({ length: count }, (_, i) => `${prefix}-${i}`);
      const select = (data: string[]): ReactElement => (
        <Select.Single<string, undefined>
          resourceName="channel"
          value="old-0"
          onChange={() => {}}
          data={data}
          initialVisible
          allowNone
        >
          {({ key, ...rest }) => (
            <Select.Item key={key} {...rest} textValue={rest.itemKey}>
              {rest.itemKey}
            </Select.Item>
          )}
        </Select.Single>
      );
      const { getByRole, getByText, rerender } = await renderThemed(
        select(keys("old", 200)),
      );
      const scroll = getByRole("listbox").element();
      await expect.poll(() => scroll.scrollHeight).toBeGreaterThan(5000);
      scroll.scrollTop = scroll.scrollHeight;
      await expect.poll(() => getByText("old-199").elements().length).toBe(1);

      await rerender(select(keys("new", 3)));

      await expect.poll(() => getByText("new-0").elements().length).toBe(1);
      expect(getByRole("dialog").element().isConnected).toBe(true);
      expect(getByRole("listbox").element()).toBe(scroll);
      expect(scroll.scrollTop).toBeLessThanOrEqual(
        Math.max(0, scroll.scrollHeight - scroll.clientHeight),
      );
    });
  });

  describe("checkable items", () => {
    const rect = (el: Element): DOMRect => el.getBoundingClientRect();

    it("should inset the box as far from the left edge as from the top", async () => {
      const item = (await renderCheckable())("type-1").closest(".lyra-list__item")!;
      const box = rect(item.querySelector(".lyra-input__checkbox-indicator")!);
      expect(box.left - rect(item).left).toBeCloseTo(box.top - rect(item).top, 0);
    });

    it("should keep a checkbox in the content in its place", async () => {
      const find = await renderCheckable(
        <Input.Checkbox value={false} onChange={() => {}} aria-label="Pin" />,
      );
      const pin = find("type-1")
        .closest(".lyra-list__item")!
        .querySelector('.lyra-input__checkbox:has(> [aria-label="Pin"])')!;
      expect(computed(pin, "margin-left")).toBe("0px");
      expect(computed(pin, "margin-right")).toBe("0px");
    });

    it("should keep each heading over its column", async () => {
      const find = await renderCheckable();
      expect(rect(find("Type")).left).toBeCloseTo(rect(find("type-1")).left, 0);
      expect(rect(find("Channel")).left).toBeCloseTo(rect(find("channel-1")).left, 0);
    });
  });
});
