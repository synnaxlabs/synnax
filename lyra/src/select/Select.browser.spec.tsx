// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
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
  describe("searchless options", () => {
    it("should close and stay closed when Enter picks an option", async () => {
      const onChange = await renderSearchless(false);
      expect(onChange.mock.calls[0][0]).toBe("medium");
      await expect
        .element(page.getByRole("button", { name: /^Level/ }))
        .toHaveAttribute("aria-expanded", "false");
    });

    it("should stay open when Enter adds one of many options", async () => {
      const onChange = await renderSearchless(true);
      expect(onChange.mock.calls[0][0]).toEqual(["low", "medium"]);
      await expect.element(page.getByRole("dialog")).toBeVisible();
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
  });
});
