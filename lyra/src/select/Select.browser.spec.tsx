// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { Select } from "@/select";
import { computed, renderThemed, resolve } from "@/styletest";
import { Triggers } from "@/triggers";

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

describe("Select", () => {
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

    it("should rest an unchosen card on the page color", async () => {
      const button = await renderButtons("card");
      const card = button("A much longer second option");
      expect(computed(card, "background-color")).toBe(resolve("var(--pluto-gray-l0)"));
      expect(computed(card, "border-top-color")).toBe(resolve("var(--pluto-gray-l6)"));
    });
  });
});
