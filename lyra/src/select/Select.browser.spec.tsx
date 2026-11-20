// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scheduler } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import {
  afterEach,
  assert,
  beforeEach,
  describe,
  expect,
  it,
  onTestFinished,
  vi,
} from "vitest";
import { page, userEvent } from "vitest/browser";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Haul } from "@/haul";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { Menu } from "@/menu";
import { type position } from "@/position";
import { Select } from "@/select";
import { computed, renderThemed, resolve } from "@/styletest";
import { Tag } from "@/tag";
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
  favorite?: ReactElement,
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
                  {favorite}
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

const MANY_KEYS = Array.from({ length: 60 }, (_, i) => `key-${i}`);
const CHOICE = "key-50";

const fixedOptions = (): ReactElement[] =>
  MANY_KEYS.map((key) => (
    <Select.Item key={key} itemKey={key}>
      {key}
    </Select.Item>
  ));

const listSelect = (
  virtual: boolean,
  data: string[] = MANY_KEYS,
  onSearch?: (term: string) => void,
): ReactElement => (
  <Dialog.Frame>
    <Select.Frame<string, undefined>
      data={data}
      value={CHOICE}
      onChange={() => {}}
      virtual={virtual}
    >
      <Select.SingleTrigger placeholder="Pick" />
      <Select.Dialog>
        <Select.Body<string> resourceName="key" onSearch={onSearch}>
          {({ key, ...p }) => (
            <Select.Item key={key} {...p}>
              {p.itemKey}
            </Select.Item>
          )}
        </Select.Body>
      </Select.Dialog>
    </Select.Frame>
  </Dialog.Frame>
);

const renderChoice = async (select: ReactElement): Promise<Element> => {
  const { container } = await renderThemed(
    <Triggers.Provider>{select}</Triggers.Provider>,
  );
  return container.querySelector(".lyra-dialog__trigger") as Element;
};

const openDialog = async (trigger: Element): Promise<Element> => {
  await userEvent.click(trigger);
  const dialog = page.getByRole("dialog").element();
  await expect.poll(() => computed(dialog, "opacity")).toBe("1");
  return dialog;
};

const isInView = (dialog: Element, text: string): boolean => {
  const option = Array.from(dialog.querySelectorAll("[role='option']")).find(
    (el) => el.textContent === text,
  );
  const scroll = option?.closest(".lyra-list__scroll");
  if (option == null || scroll == null) return false;
  const { top, bottom } = option.getBoundingClientRect();
  const view = scroll.getBoundingClientRect();
  return top >= view.top && bottom <= view.bottom;
};

interface PinnedOptions {
  body: DOMRect;
  border: number;
  listed: DOMRect;
  divider: DOMRect;
  pinned: DOMRect;
}

const renderPinnedOptions = async (
  surface: Select.DialogProps["surface"],
): Promise<PinnedOptions> => {
  await renderThemed(
    <Triggers.Provider>
      <Dialog.Frame variant="floating">
        <Dialog.Trigger>Pick</Dialog.Trigger>
        <Select.Frame<string, undefined> data={["alpha"]} onChange={() => {}}>
          <Select.Dialog surface={surface} aria-label="Key">
            <Select.Body<string>
              resourceName="key"
              footer={
                <>
                  <Menu.Divider />
                  <Select.Item<string> itemKey="none" textValue="No key">
                    No key
                  </Select.Item>
                </>
              }
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
  await userEvent.click(page.getByText("Pick"));
  const dialog = page.getByRole("dialog", { name: "Key" });
  await expect.poll(() => computed(dialog.element(), "opacity")).toBe("1");
  const body = document.querySelector(".lyra-select__body") as HTMLElement;
  const rect = (selector: string): DOMRect =>
    (document.querySelector(selector) as Element).getBoundingClientRect();
  return {
    body: body.getBoundingClientRect(),
    border: body.clientLeft,
    listed: page
      .getByRole("option", { name: "alpha" })
      .element()
      .getBoundingClientRect(),
    divider: rect(".lyra-select__body > .lyra-menu__divider"),
    pinned: page
      .getByRole("option", { name: "No key" })
      .element()
      .getBoundingClientRect(),
  };
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

const NAMES = ["Hotfire", "Stand 2", "Pass", "Calibration", "Cold flow", "Anomaly"];

const RIGHT_EDGE: position.LocationPreference = {
  targetCorner: { x: "right" },
  dialogCorner: { x: "right" },
};

interface ChipsOptions {
  initial?: string[];
  // Steps the tags and opens the dialog at the right edge, as the range list does.
  stepped?: boolean;
  // Moves the tags down with each tag, so each change places the dialog again.
  sinks?: boolean;
  // Gives each tag an icon, as labels have their dots.
  icon?: Icon.ReactElement;
  // The size of the trigger's own buttons.
  size?: Component.Size;
  preview?: boolean;
  haulType?: string;
  // A drag that runs while the trigger shows.
  dragging?: Haul.State;
}

interface ChipsProps extends ChipsOptions {
  variant: "outlined" | "shadow" | "text";
  width: number;
}

// A trigger that reads as its tags, past a gutter so its dialog never meets the left
// edge. The gutter leaves room on the right for a dialog that opens at the count.
const Chips = ({
  variant,
  width,
  initial = NAMES,
  stepped = false,
  sinks = false,
  icon,
  size,
  preview,
  haulType,
  dragging,
}: ChipsProps): ReactElement => {
  const [value, setValue] = useState(initial);
  const frame = (
    <Dialog.Frame location={stepped ? RIGHT_EDGE : undefined}>
      {sinks && <div style={{ height: `${value.length * 10}px` }} />}
      <Select.Frame<string, undefined> multiple value={value} onChange={setValue}>
        <Select.MultipleTrigger
          variant={variant}
          icon={icon}
          size={size}
          preview={preview}
          haulType={haulType}
          aria-label="Labels"
        />
        <Select.Dialog>
          {NAMES.map((name) => (
            <Select.Item key={name} itemKey={name}>
              {name}
            </Select.Item>
          ))}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
  return (
    <Haul.Provider value={dragging}>
      <div style={{ width: `${width}px`, marginLeft: "180px" }}>
        {stepped ? <Tag.StepProvider>{frame}</Tag.StepProvider> : frame}
      </div>
    </Haul.Provider>
  );
};

const renderChips = async (
  variant: ChipsProps["variant"],
  width: number,
  options: ChipsOptions = {},
): Promise<HTMLElement> => {
  const { container } = await renderThemed(
    <Chips variant={variant} width={width} {...options} />,
  );
  return container.querySelector<HTMLElement>(".lyra-dialog__trigger")!;
};

const chip = (trigger: HTMLElement, name: string): HTMLElement =>
  Array.from(trigger.querySelectorAll<HTMLElement>(":scope > .lyra-tag")).find(
    (tag) => tag.textContent === name,
  )!;

const dialogBox = (): DOMRect =>
  document.querySelector(".lyra-dialog__dialog")!.getBoundingClientRect();

// Whether the open dialog sits just above or below the element.
const meets = (el: Element): boolean => {
  const dialog = dialogBox();
  const box = el.getBoundingClientRect();
  const y = Math.min(
    Math.abs(dialog.top - box.bottom),
    Math.abs(dialog.bottom - box.top),
  );
  return y < 6;
};

// Whether the open dialog lines up with an edge of the element and sits just above or
// below it.
const besides = (el: Element): boolean => {
  const dialog = dialogBox();
  const box = el.getBoundingClientRect();
  const x = Math.min(
    Math.abs(dialog.left - box.left),
    Math.abs(dialog.right - box.right),
  );
  return x < 2 && meets(el);
};

const inWindow = (): boolean => {
  const dialog = dialogBox();
  return dialog.left >= 0 && dialog.right <= window.innerWidth;
};

// Opens the dialog at the tag and returns its x once it shows.
const openAt = async (tag: HTMLElement): Promise<number> => {
  await userEvent.click(tag);
  const dialog = page.getByRole("dialog").element();
  await expect.poll(() => computed(dialog, "opacity")).toBe("1");
  return dialogBox().left;
};

const placeholderOf = (trigger: HTMLElement): HTMLElement =>
  trigger.querySelector<HTMLElement>(
    ":scope > .lyra-select-multiple-trigger-placeholder",
  )!;

const pick = (name: string): Promise<void> =>
  userEvent.click(page.getByRole("option", { name, exact: true }));

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
  // The first arrow key moves on from the choice.
  await userEvent.keyboard("{ArrowDown}{Enter}");
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

  describe("menu surface", () => {
    it("should pick with the arrow keys and Enter, and not filter as the user types", async () => {
      const onChange = vi.fn();
      await renderThemed(
        <Triggers.Provider>
          <Dialog.Frame variant="floating">
            <Dialog.Trigger>Stage</Dialog.Trigger>
            <Select.Frame<string, undefined> multiple value={[]} onChange={onChange}>
              <Select.Dialog surface="menu" aria-label="Stage">
                <Select.List>
                  {["alpha", "beta"].map((key) => (
                    <Select.Item<string> key={key} itemKey={key}>
                      {key}
                    </Select.Item>
                  ))}
                </Select.List>
              </Select.Dialog>
            </Select.Frame>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      await userEvent.click(page.getByText("Stage"));
      const dialog = page.getByRole("dialog", { name: "Stage" });
      await expect.poll(() => computed(dialog.element(), "opacity")).toBe("1");
      await userEvent.keyboard("b");
      expect(document.querySelector(".lyra-menu-filter")).toBeNull();
      await userEvent.keyboard("{ArrowDown}{Enter}");
      await expect.poll(() => onChange).toHaveBeenCalledOnce();
      expect(onChange.mock.calls[0][0]).toEqual(["alpha"]);
    });
  });

  describe("surface", () => {
    it("should fill the dialog like a menu", async () => {
      await renderThemed(
        <Select.Simple<string>
          resourceName="level"
          value="low"
          onChange={vi.fn()}
          allowNone={false}
          searchHidden
        >
          {LEVELS}
        </Select.Simple>,
      );
      await userEvent.click(page.getByRole("button", { name: /^Level/ }));
      const dialog = page.getByRole("dialog").element();
      expect(computed(dialog, "background-color")).toBe(
        resolve("var(--pluto-gray-l1-l2)"),
      );
    });

    it("should keep the surface of a modal in a modal", async () => {
      await renderThemed(
        <Select.Simple<string>
          resourceName="level"
          value="low"
          onChange={vi.fn()}
          allowNone={false}
          variant="modal"
        >
          {LEVELS}
        </Select.Simple>,
      );
      await userEvent.click(page.getByRole("button", { name: /^Level/ }));
      const dialog = page.getByRole("dialog").element();
      expect(computed(dialog, "background-color")).toBe(
        resolve("color-mix(in srgb, var(--pluto-gray-l0) 94%, transparent)"),
      );
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

    it("should show no caret, even on hover", async () => {
      const { container } = await renderThemed(<ShadowTags value={["low"]} />);
      const item = container.querySelector(".lyra-list__item");
      assert(item != null);
      await userEvent.hover(item);
      expect(container.querySelector(".lyra-caret-animated")).toBeNull();
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

  describe("trigger that reads as its tags", () => {
    it("should open beside the tag that was clicked", async () => {
      const trigger = await renderChips("shadow", 800);
      await userEvent.click(chip(trigger, "Pass"));
      await expect.poll(() => besides(chip(trigger, "Pass"))).toBe(true);
      await userEvent.click(chip(trigger, "Hotfire"));
      await expect.poll(() => besides(chip(trigger, "Hotfire"))).toBe(true);
      await userEvent.click(chip(trigger, "Hotfire"));
      await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
    });

    it("should stay open when the tag it opened at goes", async () => {
      const trigger = await renderChips("shadow", 800);
      await userEvent.click(chip(trigger, "Pass"));
      await userEvent.click(page.getByRole("option", { name: "Pass" }));
      await expect.poll(() => chip(trigger, "Pass")).toBeUndefined();
      await expect.element(page.getByRole("dialog")).toBeVisible();
    });

    it("should open beside its count", async () => {
      const trigger = await renderChips("shadow", 150, { icon: <Icon.Label /> });
      const more = (): HTMLElement | null =>
        trigger.querySelector<HTMLElement>(":scope > .lyra-tags__more[data-count]");
      await expect.poll(more).not.toBeNull();
      const count = more();
      assert(count != null);
      await userEvent.click(count);
      await expect.poll(() => besides(count)).toBe(true);
    });

    it("should open beside its add button", async () => {
      const trigger = await renderChips("text", 800);
      const add = trigger.querySelector<HTMLElement>(
        ":scope > .lyra-btn:not(.lyra-tag)",
      );
      assert(add != null);
      await userEvent.click(add);
      await expect.poll(() => besides(add)).toBe(true);
    });

    it("should light its placeholder as a text button on hover and press", async () => {
      for (const variant of ["shadow", "text"] as const) {
        const trigger = await renderChips(variant, 800, { initial: [] });
        const placeholder = placeholderOf(trigger);
        expect(placeholder.hasAttribute("tabindex")).toBe(false);
        expect(computed(placeholder, "font-weight")).toBe(
          computed(trigger, "font-weight"),
        );
        await userEvent.hover(placeholder);
        expect(computed(placeholder, "background-color")).toBe(
          resolve("var(--pluto-fill-hover)"),
        );
        placeholder.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
        await expect
          .poll(() => computed(placeholder, "background-color"))
          .toBe(resolve("var(--pluto-fill-press)"));
        document.dispatchEvent(new MouseEvent("mouseup"));
        await userEvent.unhover(placeholder);
      }
    });

    it.each([
      ["shadow", undefined, "medium"],
      ["text", undefined, "medium"],
      ["shadow", "small", "small"],
      ["text", "small", "small"],
    ] as const)(
      "should size the placeholder of a %s trigger of size %s as a %s text button",
      async (variant, size, expected) => {
        await renderThemed(
          <Button.Button variant="text" size={expected}>
            Reference
          </Button.Button>,
        );
        const reference = page.getByRole("button", { name: "Reference" }).element();
        const trigger = await renderChips(variant, 800, { initial: [], size });
        const placeholder = placeholderOf(trigger);
        for (const property of ["height", "font-size", "padding-left", "padding-right"])
          expect(computed(placeholder, property)).toBe(computed(reference, property));
      },
    );

    it("should keep a small trigger as tall as its tags when the first tag comes", async () => {
      const trigger = (): HTMLElement =>
        document.querySelector<HTMLElement>(".lyra-dialog__trigger")!;
      await renderChips("shadow", 800, { initial: [], size: "small" });
      const placeholder = placeholderOf(trigger());
      const placeholderHeight = computed(placeholder, "height");
      const { height } = trigger().getBoundingClientRect();
      await userEvent.click(placeholder);
      await pick("Pass");
      await expect.poll(() => chip(trigger(), "Pass")).toBeDefined();
      expect(computed(chip(trigger(), "Pass"), "height")).toBe(placeholderHeight);
      expect(trigger().getBoundingClientRect().height).toBe(height);
    });

    it("should keep the placeholder of an outlined trigger as text", async () => {
      const trigger = await renderChips("outlined", 800, { initial: [] });
      expect(placeholderOf(trigger).classList.contains("lyra-btn")).toBe(false);
    });

    it("should take no clicks in the room around its tags", async () => {
      for (const variant of ["shadow", "text"] as const) {
        const trigger = await renderChips(variant, 200, { initial: ["Pass"] });
        const room = trigger.getBoundingClientRect();
        const hit = document.elementFromPoint(
          room.right - 5,
          room.top + room.height / 2,
        );
        expect(hit).not.toBeNull();
        expect(trigger.contains(hit)).toBe(false);
      }
    });

    it("should take a drop anywhere on it while a drag it can take runs", async () => {
      const trigger = await renderChips("shadow", 200, {
        initial: ["Pass"],
        haulType: "label",
        dragging: {
          key: "drag",
          source: { key: "list", type: "label" },
          items: [{ key: "New", type: "label" }],
        },
      });
      const room = trigger.getBoundingClientRect();
      const hit = document.elementFromPoint(room.right - 5, room.top + room.height / 2);
      expect(hit).toBe(trigger);
    });

    it("should open from no tag of a preview", async () => {
      const trigger = await renderChips("text", 200, { preview: true });
      // A preview reads as disabled, which a user can still click.
      chip(trigger, "Pass").click();
      await scheduler.nextFrame();
      expect(document.querySelector(".lyra-dialog__dialog")).toBeNull();
    });

    describe("dialog opened at a tag", () => {
      const VIEWPORT = { width: window.innerWidth, height: window.innerHeight };
      beforeEach(async () => {
        await page.viewport(1200, 800);
      });
      afterEach(async () => {
        await page.viewport(VIEWPORT.width, VIEWPORT.height);
      });

      it("should open beside the placeholder of an empty trigger", async () => {
        const trigger = await renderChips("shadow", 260, {
          initial: [],
          stepped: true,
        });
        const placeholder = placeholderOf(trigger);
        await userEvent.click(placeholder);
        await expect.poll(() => besides(placeholder)).toBe(true);
      });

      it("should keep its x but follow its tag down as tags are added", async () => {
        const trigger = await renderChips("shadow", 260, {
          initial: ["Pass", "Calibration"],
          stepped: true,
          sinks: true,
        });
        const left = await openAt(chip(trigger, "Pass"));
        for (const name of ["Hotfire", "Stand 2", "Cold flow"]) {
          await pick(name);
          await expect.poll(() => meets(chip(trigger, "Pass"))).toBe(true);
          expect(dialogBox().left).toBe(left);
        }
      });

      it("should keep its x as tags go", async () => {
        const trigger = await renderChips("shadow", 260, {
          stepped: true,
          sinks: true,
        });
        const left = await openAt(chip(trigger, "Calibration"));
        for (const name of ["Hotfire", "Stand 2"]) {
          await pick(name);
          await expect.poll(() => meets(chip(trigger, "Calibration"))).toBe(true);
          expect(dialogBox().left).toBe(left);
        }
      });

      it("should keep its x when its tag folds into the count", async () => {
        const trigger = await renderChips("shadow", 120, {
          initial: ["Pass"],
          stepped: true,
          sinks: true,
        });
        const left = await openAt(chip(trigger, "Pass"));
        await pick("Hotfire");
        await expect.poll(() => meets(chip(trigger, "Pass"))).toBe(true);
        expect(dialogBox().left).toBe(left);
        await pick("Stand 2");
        await pick("Cold flow");
        await expect
          .poll(() => chip(trigger, "Pass").hasAttribute("data-hidden"))
          .toBe(true);
        expect(dialogBox().left).toBe(left);
      });

      it("should stay in the window when its tag goes and the window narrows", async () => {
        const trigger = await renderChips("shadow", 260, {
          initial: ["Pass", "Calibration"],
          stepped: true,
        });
        const left = await openAt(chip(trigger, "Pass"));
        await pick("Pass");
        await expect.poll(() => chip(trigger, "Pass")).toBeUndefined();
        expect(dialogBox().left).toBe(left);
        await page.viewport(300, 800);
        await expect.poll(inWindow).toBe(true);
      });

      it("should go back beside its tag when the window resizes", async () => {
        const trigger = await renderChips("shadow", 260, {
          initial: ["Pass", "Calibration"],
          stepped: true,
          sinks: true,
        });
        await openAt(chip(trigger, "Pass"));
        await pick("Stand 2");
        await expect.poll(() => meets(chip(trigger, "Pass"))).toBe(true);
        await page.viewport(1100, 800);
        await expect.poll(() => besides(chip(trigger, "Pass"))).toBe(true);
      });
    });
  });

  describe("opening", () => {
    it("should scroll a static list to the choice", async () => {
      const dialog = await openDialog(await renderChoice(listSelect(false)));
      await expect.poll(() => isInView(dialog, CHOICE)).toBe(true);
    });

    it("should scroll a virtual list to the choice", async () => {
      const dialog = await openDialog(await renderChoice(listSelect(true)));
      await expect.poll(() => isInView(dialog, CHOICE)).toBe(true);
    });

    it("should scroll a list of fixed options to the choice", async () => {
      const trigger = await renderChoice(
        <Select.Simple<string> resourceName="key" value={CHOICE} onChange={() => {}}>
          {fixedOptions()}
        </Select.Simple>,
      );
      const dialog = await openDialog(trigger);
      await expect.poll(() => isInView(dialog, CHOICE)).toBe(true);
    });

    it("should scroll a multiple select to its first choice in the list", async () => {
      const trigger = await renderChoice(
        <Select.Simple<string>
          resourceName="key"
          multiple
          value={["key-59", "key-30"]}
          onChange={() => {}}
        >
          {fixedOptions()}
        </Select.Simple>,
      );
      const dialog = await openDialog(trigger);
      await expect.poll(() => isInView(dialog, "key-30")).toBe(true);
    });

    it("should scroll to a choice that loads after the dialog opens", async () => {
      const { container, rerender } = await renderThemed(
        <Triggers.Provider>{listSelect(true, [])}</Triggers.Provider>,
      );
      const dialog = await openDialog(
        container.querySelector(".lyra-dialog__trigger") as Element,
      );
      // The first try at the reveal finds no options.
      await scheduler.nextFrame();
      await rerender(<Triggers.Provider>{listSelect(true)}</Triggers.Provider>);
      await expect.poll(() => isInView(dialog, CHOICE)).toBe(true);
    });

    it.each([
      [
        "scrolls",
        (scroll: Element) =>
          scroll.dispatchEvent(new WheelEvent("wheel", { bubbles: true })),
      ],
      ["moves the cursor", async () => await userEvent.keyboard("{ArrowDown}")],
    ])("should not scroll to a late choice once the user %s", async (_, act) => {
      const early = MANY_KEYS.filter((key) => key !== CHOICE);
      const { container, rerender } = await renderThemed(
        <Triggers.Provider>{listSelect(true, early)}</Triggers.Provider>,
      );
      const dialog = await openDialog(
        container.querySelector(".lyra-dialog__trigger") as Element,
      );
      const scroll = dialog.querySelector(".lyra-list__scroll") as Element;
      // The reveal waits for the choice once the dialog shows.
      await scheduler.nextFrame();
      await act(scroll);
      await rerender(<Triggers.Provider>{listSelect(true)}</Triggers.Provider>);
      await scheduler.nextFrame();
      await scheduler.nextFrame();
      expect(scroll.scrollTop).toBe(0);
    });

    it("should not scroll to a late choice once the user searches", async () => {
      const early = MANY_KEYS.filter((key) => key !== CHOICE);
      const select = (data: string[]): ReactElement => (
        <Triggers.Provider>{listSelect(true, data, () => {})}</Triggers.Provider>
      );
      const { container, rerender } = await renderThemed(select(early));
      const dialog = await openDialog(
        container.querySelector(".lyra-dialog__trigger") as Element,
      );
      await scheduler.nextFrame();
      (dialog.querySelector("input") as HTMLInputElement).focus();
      await userEvent.keyboard("k");
      await rerender(select(MANY_KEYS));
      await scheduler.nextFrame();
      await scheduler.nextFrame();
      expect(dialog.querySelector(".lyra-list__scroll")?.scrollTop).toBe(0);
    });

    it("should move the hover on from the choice", async () => {
      const onChange = vi.fn();
      const trigger = await renderChoice(
        <Select.Simple<string> resourceName="key" value={CHOICE} onChange={onChange}>
          {fixedOptions()}
        </Select.Simple>,
      );
      await openDialog(trigger);
      await userEvent.keyboard("{ArrowDown}{Enter}");
      await expect
        .poll(() => onChange)
        .toHaveBeenCalledWith("key-51", expect.anything());
    });

    it("should not scroll to the choice of a list outside a select dialog", async () => {
      const scroll = vi.spyOn(Element.prototype, "scrollIntoView");
      onTestFinished(() => scroll.mockRestore());
      await renderThemed(
        <Triggers.Provider>
          <Dialog.Frame variant="modal" visible>
            <Dialog.Dialog>
              <Select.Frame<string, undefined>
                data={MANY_KEYS}
                value={CHOICE}
                onChange={() => {}}
              >
                <Select.List>
                  <Select.Items<string>>
                    {({ key, ...p }) => (
                      <Select.Item key={key} {...p}>
                        {p.itemKey}
                      </Select.Item>
                    )}
                  </Select.Items>
                </Select.List>
              </Select.Frame>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      await scheduler.nextFrame();
      await scheduler.nextFrame();
      expect(scroll).not.toHaveBeenCalled();
    });

    it("should show the dialog already scrolled to the choice", async () => {
      const trigger = await renderChoice(listSelect(false));
      const shown: boolean[] = [];
      const observer = new MutationObserver(() => {
        const dialog = document.querySelector("[role='dialog']");
        if (shown.length > 0 || !dialog?.classList.contains(CSS.visible(true))) return;
        shown.push(isInView(dialog, CHOICE));
      });
      observer.observe(document.body, {
        subtree: true,
        childList: true,
        attributeFilter: ["class"],
      });
      onTestFinished(() => observer.disconnect());
      await openDialog(trigger);
      expect(shown).toEqual([true]);
    });

    it("should scroll a virtual list to the choice when it opens again", async () => {
      const trigger = await renderChoice(listSelect(true));
      const first = await openDialog(trigger);
      await expect.poll(() => isInView(first, CHOICE)).toBe(true);
      first.querySelector(".lyra-list__scroll")?.scrollTo({ top: 0 });
      await userEvent.keyboard("{Escape}");
      await expect.poll(() => document.querySelector("[role='dialog']")).toBeNull();
      const second = await openDialog(trigger);
      await expect.poll(() => isInView(second, CHOICE)).toBe(true);
    });
  });

  describe("footer", () => {
    it("should keep its height when the list overflows the dialog", async () => {
      const footer = await renderFooter();
      expect(footer.getBoundingClientRect().height).toBe(FOOTER_HEIGHT);
    });

    it("should line up its options and divider with the options of the list", async () => {
      const { body, listed, divider, pinned } = await renderPinnedOptions("dialog");
      expect(pinned.left).toBe(listed.left);
      expect(pinned.right).toBe(listed.right);
      expect(divider.left).toBe(listed.left);
      expect(divider.right).toBe(listed.right);
      expect(body.bottom - pinned.bottom).toBe(listed.left - body.left);
    });

    it("should leave its options in a menu where the panel puts them", async () => {
      const { body, border, listed, divider, pinned } =
        await renderPinnedOptions("menu");
      expect(pinned.left).toBe(listed.left);
      expect(pinned.right).toBe(listed.right);
      expect(divider.left).toBe(listed.left);
      expect(body.bottom - pinned.bottom).toBe(border);
    });

    it("should not inset a footer that pins no option", async () => {
      const footer = await renderFooter();
      const body = footer.parentElement as HTMLElement;
      expect(footer.getBoundingClientRect().left).toBe(
        body.getBoundingClientRect().left + body.clientLeft,
      );
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

  describe("clearable shadow single trigger", () => {
    const renderClearable = async (
      onChange: (key: string | null) => void,
    ): Promise<HTMLElement> => {
      const { container } = await renderThemed(
        <Dialog.Frame>
          <Select.Frame<string, undefined> value="time" onChange={onChange} allowNone>
            <Select.SingleTrigger variant="shadow" placeholder="Pick" clearable />
            <Select.Dialog>
              <Select.Item itemKey="time">Time</Select.Item>
            </Select.Dialog>
          </Select.Frame>
        </Dialog.Frame>,
      );
      const trigger = container.querySelector<HTMLElement>(".lyra-dialog__trigger");
      assert(trigger != null);
      return trigger;
    };

    const clearOf = (trigger: HTMLElement): HTMLElement => {
      const clear = trigger.querySelector<HTMLElement>(
        ".lyra-select-single-trigger__clear",
      );
      assert(clear != null);
      return clear;
    };

    it("should show its clear button in place of its caret on hover", async () => {
      const trigger = await renderClearable(vi.fn());
      expect(computed(clearOf(trigger), "opacity")).toBe("0");
      await userEvent.hover(trigger);
      await expect.poll(() => computed(clearOf(trigger), "opacity")).toBe("1");
      const caret = trigger.querySelector(".lyra-caret-animated");
      assert(caret != null);
      expect(computed(caret, "visibility")).toBe("hidden");
    });

    it("should clear the selection from the button on its caret", async () => {
      const onChange = vi.fn();
      const trigger = await renderClearable(onChange);
      await userEvent.hover(trigger);
      await userEvent.click(clearOf(trigger));
      expect(onChange).toHaveBeenLastCalledWith(null, expect.anything());
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
      // The dialog scrolls to its choice a frame after it shows.
      const dialog = getByRole("dialog").element();
      await expect.poll(() => computed(dialog, "opacity")).toBe("1");
      await scheduler.nextFrame();
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
        <Input.Checkbox value={false} onChange={() => {}} aria-label="Favorite" />,
      );
      const favorite = find("type-1")
        .closest(".lyra-list__item")!
        .querySelector('.lyra-input__checkbox:has(> [aria-label="Favorite"])')!;
      expect(computed(favorite, "margin-left")).toBe("0px");
      expect(computed(favorite, "margin-right")).toBe("0px");
    });

    it("should keep each heading over its column", async () => {
      const find = await renderCheckable();
      expect(rect(find("Type")).left).toBeCloseTo(rect(find("type-1")).left, 0);
      expect(rect(find("Channel")).left).toBeCloseTo(rect(find("channel-1")).left, 0);
    });
  });
});
