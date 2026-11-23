// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { assert, describe, expect, it, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Button } from "@/button";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { computed, height, renderThemed, resolve } from "@/styletest";
import { Triggers } from "@/triggers";

const renderInput = async (
  props: Partial<Input.TextProps> = {},
): Promise<{ input: Element; box: Element }> => {
  const { getByRole } = await renderThemed(
    <Input.Text value="" onChange={() => {}} placeholder="Name" {...props} />,
  );
  const input = getByRole("textbox").element();
  return { input, box: input.closest(`.${CSS.B("input")}`)! };
};

const background = (el: Element): string => computed(el, "background-color");

// The left edge of the text, past the input's own padding.
const textStart = (input: Element): number =>
  input.getBoundingClientRect().left + parseFloat(computed(input, "padding-left"));

describe("Input", () => {
  it("should paint from the outlined rest tokens", async () => {
    const { box } = await renderInput();
    expect(background(box)).toBe(resolve("var(--pluto-fill-rest)"));
    expect(computed(box, "border-top-color")).toBe(resolve("var(--pluto-gray-l7)"));
  });

  it("should be as tall as the medium height token", async () => {
    const { box } = await renderInput();
    expect(height(box)).toBe(
      parseFloat(resolve("var(--pluto-height-medium)", "height")),
    );
  });

  it("should ring itself in the primary color while focused", async () => {
    const { input, box } = await renderInput();
    await userEvent.click(input);
    const primary = resolve("var(--pluto-primary-z)");
    expect(computed(box, "border-top-color")).toBe(primary);
    expect(computed(box, "box-shadow")).toContain(primary);
  });

  it("should keep its rest fill under the pointer while focused", async () => {
    const { input, box } = await renderInput();
    await userEvent.click(input);
    expect(background(box)).toBe(resolve("var(--pluto-fill-rest)"));
  });

  it("should take the hover fill when hovered without focus", async () => {
    const { box } = await renderInput();
    await userEvent.hover(box);
    expect(background(box)).toBe(resolve("var(--pluto-fill-hover)"));
  });

  it("should color its placeholder with the placeholder token", async () => {
    const { input } = await renderInput();
    expect(getComputedStyle(input, "::placeholder").color).toBe(
      resolve("var(--pluto-text-placeholder)"),
    );
  });

  it("should dim its placeholder when disabled", async () => {
    const { input } = await renderInput({ disabled: true });
    expect(getComputedStyle(input, "::placeholder").color).toBe(
      resolve("var(--pluto-text-disabled)"),
    );
  });

  describe("start icon", () => {
    const renderIconInputs = async () => {
      const { getByPlaceholder } = await renderThemed(
        <>
          <Input.Text value="" onChange={() => {}} placeholder="text" variant="text" />
          <Input.Text
            value=""
            onChange={() => {}}
            placeholder="text icon"
            variant="text"
            startContent={<Icon.Search />}
          />
          <Input.Text
            value=""
            onChange={() => {}}
            placeholder="outlined icon"
            startContent={<Icon.Search />}
          />
        </>,
      );
      const parts = (placeholder: string) => {
        const input = getByPlaceholder(placeholder, { exact: true }).element();
        const box = input.closest(`.${CSS.B("input")}`)!;
        const icon = box.querySelector("svg")?.getBoundingClientRect();
        return { input, left: box.getBoundingClientRect().left, icon };
      };
      return {
        text: parts("text"),
        textIcon: parts("text icon"),
        outlinedIcon: parts("outlined icon"),
      };
    };

    it("should sit in a text variant where its text sits without one", async () => {
      const { text, textIcon } = await renderIconInputs();
      expect(textIcon.icon!.left - textIcon.left).toBe(
        textStart(text.input) - text.left,
      );
    });

    it("should keep a text variant's text as clear as an outlined one", async () => {
      const { textIcon, outlinedIcon } = await renderIconInputs();
      expect(textStart(textIcon.input) - textIcon.icon!.right).toBe(
        textStart(outlinedIcon.input) - outlinedIcon.icon!.right,
      );
    });
  });

  describe("slot button", () => {
    const renderSlotButton = async (props: Partial<Input.TextProps> = {}) => {
      let pressed = 0;
      const { getByRole } = await renderThemed(
        <Input.Text
          value=""
          onChange={() => {}}
          placeholder="Name"
          {...props}
          startContent={
            <Button.Button aria-label="Pick color" onClick={() => pressed++}>
              <Icon.Add />
            </Button.Button>
          }
        />,
      );
      return {
        button: getByRole("button", { name: "Pick color" }),
        input: () => getByRole("textbox").element(),
        pressed: () => pressed,
      };
    };

    it("should take a click on a button in the start content", async () => {
      const { button, pressed } = await renderSlotButton();
      await userEvent.click(button);
      expect(pressed()).toBe(1);
    });

    it("should leave a slot button of a preview input inert", async () => {
      const { button, pressed } = await renderSlotButton({ preview: true });
      await userEvent.click(button, { force: true });
      expect(pressed()).toBe(0);
    });

    it("should leave a slot button of a disabled input inert", async () => {
      const { button, pressed } = await renderSlotButton({ disabled: true });
      await userEvent.click(button, { force: true });
      expect(pressed()).toBe(0);
    });

    it("should not focus the input on a press of a slot button", async () => {
      const { button, input } = await renderSlotButton();
      await userEvent.click(button);
      expect(document.activeElement).not.toBe(input());
    });

    it("should leave a press in a dialog that a slot button opens to the dialog", async () => {
      const { getByRole } = await renderThemed(
        <Triggers.Provider>
          <Input.Text
            value=""
            onChange={() => {}}
            placeholder="Name"
            startContent={
              <Color.Swatch
                value="#ff0000"
                onChange={() => {}}
                aria-label="Pick color"
              />
            }
          />
        </Triggers.Provider>,
      );
      await userEvent.click(getByRole("button", { name: "Pick color" }));
      const hex = page.getByRole("textbox", { name: "Hex" });
      await userEvent.click(hex);
      expect(document.activeElement).toBe(hex.element());
    });
  });

  it("should stack above the packed button beside it", async () => {
    const { getByRole } = await renderThemed(
      <Input.Text value="" onChange={() => {}} placeholder="Name">
        <Button.Button>Go</Button.Button>
      </Input.Text>,
    );
    const box = getByRole("textbox")
      .element()
      .closest(`.${CSS.B("input")}`)!;
    const button = getByRole("button", { name: "Go" }).element();
    expect(Number(computed(box, "z-index"))).toBeGreaterThan(
      Number(computed(button, "z-index")),
    );
  });

  describe("time editor", () => {
    it("should close on Enter and give the focus to the trigger", async () => {
      await renderThemed(
        <Triggers.Provider>
          <Input.DateTime
            value={TimeStamp.now().nanoseconds}
            onChange={() => {}}
            aria-label="Start"
          />
        </Triggers.Provider>,
      );
      const trigger = page.getByRole("button", { name: "Start" }).element();
      assert(trigger instanceof HTMLElement);
      trigger.focus();
      await userEvent.keyboard("{Enter}");
      await expect.element(page.getByRole("textbox")).toHaveFocus();
      await expect.element(page.getByRole("menuitem").first()).toBeVisible();
      await userEvent.keyboard("{Enter}");
      await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
      expect(document.activeElement).toBe(trigger);
    });

    it("should stay closed after a click on a reading of an inline editor", async () => {
      await renderThemed(
        <Triggers.Provider>
          <Input.TimeSpan
            inline
            value={TimeSpan.HOUR.nanoseconds}
            onChange={() => {}}
            aria-label="Period"
          />
        </Triggers.Provider>,
      );
      await userEvent.click(page.getByRole("textbox", { name: "Period" }));
      const reading = page.getByRole("menuitem").first();
      await expect.element(reading).toBeVisible();
      await userEvent.click(reading);
      await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
      expect(page.getByRole("textbox", { name: "Period" }).element()).not.toBe(
        document.activeElement,
      );
    });

    it("should keep the edit when the field takes the focus back", async () => {
      await renderThemed(
        <Triggers.Provider>
          <Input.TimeSpan
            inline
            value={TimeSpan.HOUR.nanoseconds}
            onChange={() => {}}
            aria-label="Period"
          />
        </Triggers.Provider>,
      );
      const field = page.getByRole("textbox", { name: "Period" });
      await userEvent.click(field);
      await userEvent.fill(field, "90s");
      await userEvent.click(page.getByRole("dialog"), { position: { x: 1, y: 1 } });
      await userEvent.click(field);
      await expect.element(field).toHaveValue("90s");
    });
  });
});

describe("Input", () => {
  describe("Text", () => {
    describe("keyboard handlers", () => {
      it("should close a modal on one Escape from an input in it", () => {
        const c = render(
          <Triggers.Provider>
            <Dialog.Frame variant="modal">
              <Dialog.Trigger>Open</Dialog.Trigger>
              <Dialog.Dialog>
                <Input.Text value="" onChange={vi.fn()} aria-label="Name" />
              </Dialog.Dialog>
            </Dialog.Frame>
          </Triggers.Provider>,
        );
        fireEvent.click(c.getByText("Open"));
        const input = c.getByLabelText("Name");
        act(() => input.focus());
        fireEvent.keyDown(input, { key: "Escape", code: "Escape" });
        expect(c.queryByLabelText("Name")).toBeNull();
      });

      it("should close only a dropdown in a modal on Escape from its search", () => {
        const c = render(
          <Triggers.Provider>
            <Dialog.Frame variant="modal">
              <Dialog.Trigger>Open</Dialog.Trigger>
              <Dialog.Dialog>
                <p>Modal content</p>
                <Dialog.Frame variant="connected">
                  <Dialog.Trigger>Pick</Dialog.Trigger>
                  <Dialog.Dialog>
                    <Input.Search value="" onChange={vi.fn()} aria-label="Search" />
                  </Dialog.Dialog>
                </Dialog.Frame>
              </Dialog.Dialog>
            </Dialog.Frame>
          </Triggers.Provider>,
        );
        fireEvent.click(c.getByText("Open"));
        fireEvent.click(c.getByText("Pick"));
        const search = c.getByLabelText("Search");
        act(() => search.focus());
        fireEvent.keyDown(search, { key: "Escape", code: "Escape" });
        fireEvent.keyUp(document.body, { key: "Escape", code: "Escape" });
        expect(c.queryByLabelText("Search")).toBeNull();
        expect(c.getByText("Modal content")).toBeTruthy();
        fireEvent.keyDown(c.container, { key: "Escape", code: "Escape" });
        expect(c.queryByText("Modal content")).toBeNull();
      });
    });
  });
});
