// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode, useState } from "react";
import { assert, describe, expect, it } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Menu } from "@/menu";
import { computed, renderThemed, resolve } from "@/styletest";
import { Triggers } from "@/triggers";

const TALL = <div style={{ height: 2000, flexShrink: 0 }}>Tall</div>;
const WIDE = <p>{"wide ".repeat(200)}</p>;

const openDialog = async (
  variant: Dialog.FrameProps["variant"],
  content: ReactNode = <p>Content</p>,
  dialogProps: Dialog.DialogProps = {},
): Promise<{ trigger: Element; dialog: Element; unmount: () => unknown }> => {
  const { getByText, unmount } = await renderThemed(
    <Triggers.Provider>
      <Dialog.Frame variant={variant}>
        <Dialog.Trigger>Toggle</Dialog.Trigger>
        <Dialog.Dialog {...dialogProps}>{content}</Dialog.Dialog>
      </Dialog.Frame>
    </Triggers.Provider>,
  );
  const trigger = getByText("Toggle").element();
  await userEvent.click(trigger);
  await expect.element(page.getByRole("dialog")).toBeVisible();
  const dialog = page.getByRole("dialog").element();
  // toBeVisible accepts zero opacity, and the frame shows a floating dialog only once
  // it has placed it.
  await expect.poll(() => computed(dialog, "opacity")).toBe("1");
  return { trigger, dialog, unmount };
};

/** @returns the height of a floating dialog of the size, holding tall content. */
const capOf = async (size?: Dialog.DialogProps["size"]): Promise<number> => {
  const { dialog, unmount } = await openDialog("floating", TALL, { size });
  const { height } = dialog.getBoundingClientRect();
  await unmount();
  return height;
};

/** @returns the width of a floating dialog holding a long line of text. */
const widthOf = async (maxWidth?: string): Promise<number> => {
  const style: CSS.VarProperties =
    maxWidth == null ? {} : { "--pluto-dialog-max-width": maxWidth };
  const { dialog, unmount } = await openDialog("floating", WIDE, { style });
  const { width } = dialog.getBoundingClientRect();
  await unmount();
  return width;
};

/** @returns true when the element is the topmost one at its own center. */
const onTop = (el: Element): boolean => {
  const rect = el.getBoundingClientRect();
  const hit = document.elementFromPoint(
    rect.left + rect.width / 2,
    rect.top + rect.height / 2,
  );
  return hit != null && el.contains(hit);
};

/** @returns the element the locator finds, which must be an HTML element. */
const html = (locator: Locator): HTMLElement => {
  const el = locator.element();
  assert(el instanceof HTMLElement);
  return el;
};

const button = (name: string): HTMLElement => html(page.getByRole("button", { name }));

const field = (name: string): HTMLElement => html(page.getByLabelText(name));

/** Opens a modal from the keyboard, so the trigger holds the focus in every engine. */
const openModal = async (
  content: ReactNode,
  behind?: ReactNode,
): Promise<{ trigger: HTMLElement; dialog: Element }> => {
  await renderThemed(
    <Triggers.Provider>
      {behind}
      <Dialog.Frame variant="modal">
        <Dialog.Trigger>Open</Dialog.Trigger>
        <Dialog.Dialog>{content}</Dialog.Dialog>
      </Dialog.Frame>
    </Triggers.Provider>,
  );
  const trigger = button("Open");
  trigger.focus();
  await userEvent.keyboard("{Enter}");
  await expect.element(page.getByRole("dialog")).toBeVisible();
  return { trigger, dialog: page.getByRole("dialog").element() };
};

/** Opens a dropdown from the keyboard, between a field before it and one after it. */
const openDropdown = async (content: ReactNode): Promise<HTMLElement> => {
  await renderThemed(
    <Triggers.Provider>
      <input aria-label="Before" />
      <Dialog.Frame>
        <Dialog.Trigger>Menu</Dialog.Trigger>
        <Dialog.Dialog>{content}</Dialog.Dialog>
      </Dialog.Frame>
      <input aria-label="After" />
    </Triggers.Provider>,
  );
  const trigger = button("Menu");
  trigger.focus();
  await userEvent.keyboard("{Enter}");
  await expect.element(page.getByLabelText("Search")).toHaveFocus();
  return trigger;
};

/** An item that closes its dropdown when clicked, and cannot take the focus. */
const Pick = (): ReactElement => {
  const { close } = Dialog.useContext();
  return <div onClick={close}>Pick</div>;
};

const SEARCH = <input aria-label="Search" autoFocus />;

/** A dropdown that opens on a button it does not wrap. */
const Anchored = (): ReactElement => {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  return (
    <Triggers.Provider>
      <button ref={setAnchor}>Anchor</button>
      <Dialog.Frame id="anchored" anchor={anchor}>
        <Dialog.Dialog>{SEARCH}</Dialog.Dialog>
      </Dialog.Frame>
    </Triggers.Provider>
  );
};

/** A field that opens a context menu, which takes the focus. */
const MenuTarget = (): ReactElement => {
  const ctx = Menu.useContextMenu();
  return (
    <Menu.ContextMenu menu={() => <Menu.Item itemKey="copy">Copy</Menu.Item>} {...ctx}>
      <input aria-label="Search" autoFocus onContextMenu={ctx.open} />
    </Menu.ContextMenu>
  );
};

/** A dropdown holding another, whose button closes both in one render. */
const Nested = (): ReactElement => {
  const [visible, setVisible] = useState(false);
  return (
    <Triggers.Provider>
      <Dialog.Frame visible={visible} onVisibleChange={setVisible}>
        <Dialog.Trigger>Menu</Dialog.Trigger>
        <Dialog.Dialog>
          <Dialog.Frame>
            <Dialog.Trigger>Inner</Dialog.Trigger>
            <Dialog.Dialog>
              <button onClick={() => setVisible(false)}>Done</button>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Dialog.Dialog>
      </Dialog.Frame>
    </Triggers.Provider>
  );
};

/** A modal holding a dropdown between two fields. */
const DROPDOWN_FIELDS = (
  <>
    <input aria-label="First" />
    <Dialog.Frame>
      <Dialog.Trigger>Menu</Dialog.Trigger>
      <Dialog.Dialog>{SEARCH}</Dialog.Dialog>
    </Dialog.Frame>
    <input aria-label="Last" />
  </>
);

const FIELDS = (
  <>
    <input aria-label="First" />
    <input aria-label="Last" />
  </>
);

/** Opens one modal from another, as a palette command opens a modal. */
const Chain = (): ReactElement => {
  const [step, setStep] = useState<"first" | "second" | null>(null);
  const close = () => setStep(null);
  return (
    <Triggers.Provider>
      <button onClick={() => setStep("first")}>Start</button>
      <Dialog.Frame variant="modal" visible={step === "first"} onVisibleChange={close}>
        <Dialog.Dialog>
          <button onClick={() => setStep("second")}>Next</button>
        </Dialog.Dialog>
      </Dialog.Frame>
      <Dialog.Frame variant="modal" visible={step === "second"} onVisibleChange={close}>
        <Dialog.Dialog>
          <input aria-label="Name" autoFocus />
        </Dialog.Dialog>
      </Dialog.Frame>
    </Triggers.Provider>
  );
};

describe("Dialog", () => {
  describe("shadow trigger", () => {
    const renderShadow = async (): Promise<Element> => {
      const { getByText } = await renderThemed(
        <Dialog.Frame>
          <Dialog.Trigger variant="shadow">Value</Dialog.Trigger>
          <Dialog.Dialog>Content</Dialog.Dialog>
        </Dialog.Frame>,
      );
      return getByText("Value").element();
    };

    it("should draw no box at rest", async () => {
      const trigger = await renderShadow();
      expect(computed(trigger, "background-color")).toBe(resolve("transparent"));
      expect(computed(trigger, "border-top-color")).toBe(resolve("transparent"));
    });

    it("should hide its caret at rest", async () => {
      const trigger = await renderShadow();
      const caret = trigger.querySelector(`.${CSS.B("caret-animated")}`);
      assert(caret != null);
      expect(computed(caret, "opacity")).toBe("0");
    });
  });

  describe("dropdown", () => {
    it("should float on the overlay shadow, over the page", async () => {
      const { dialog } = await openDialog("floating");
      expect(computed(dialog, "position")).toBe("fixed");
      expect(computed(dialog, "box-shadow")).toBe(
        resolve("var(--pluto-shadow-overlay)", "box-shadow"),
      );
      expect(onTop(dialog)).toBe(true);
    });

    it("should open beside its trigger without covering it", async () => {
      const { trigger, dialog } = await openDialog("floating");
      const t = trigger.getBoundingClientRect();
      const d = dialog.getBoundingClientRect();
      expect(d.top >= t.bottom || d.bottom <= t.top).toBe(true);
    });

    it("should cap the height of tall content", async () => {
      expect(await capOf()).toBeLessThan(2000);
    });

    it("should cap a small dialog lower and a large one higher", async () => {
      const medium = await capOf();
      expect(await capOf("small")).toBeLessThan(medium);
      expect(await capOf("large")).toBeGreaterThan(medium);
    });

    it("should wrap its content at the maximum width its caller sets", async () => {
      expect(await widthOf()).toBeGreaterThan(300);
      expect(await widthOf("200px")).toBeCloseTo(200, 0);
    });

    describe("focus", () => {
      it("should move Tab through its own fields", async () => {
        await openDropdown(
          <>
            {SEARCH}
            <button>Add</button>
          </>,
        );
        await userEvent.tab();
        expect(document.activeElement).toBe(button("Add"));
      });

      it("should close and pass the trigger on Tab from its last field", async () => {
        await openDropdown(SEARCH);
        await userEvent.tab();
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(field("After"));
      });

      it("should focus the trigger on Shift+Tab from its first field", async () => {
        const trigger = await openDropdown(SEARCH);
        await userEvent.tab({ shift: true });
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(trigger);
      });

      it("should focus the trigger on Escape", async () => {
        const trigger = await openDropdown(SEARCH);
        await userEvent.keyboard("{Escape}");
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(trigger);
      });

      it("should focus the trigger when a click inside closes it", async () => {
        const trigger = await openDropdown(
          <>
            {SEARCH}
            <Pick />
          </>,
        );
        await userEvent.click(page.getByText("Pick"));
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(trigger);
      });

      it("should go on in the outer dropdown when Tab leaves an inner one", async () => {
        await openDropdown(
          <>
            {SEARCH}
            <Dialog.Frame>
              <Dialog.Trigger>Inner</Dialog.Trigger>
              <Dialog.Dialog>
                <input aria-label="Inner search" autoFocus />
              </Dialog.Dialog>
            </Dialog.Frame>
            <button>Add</button>
          </>,
        );
        button("Inner").focus();
        await userEvent.keyboard("{Enter}");
        await expect.element(page.getByLabelText("Inner search")).toHaveFocus();
        await userEvent.tab();
        await expect
          .element(page.getByLabelText("Inner search"))
          .not.toBeInTheDocument();
        expect(document.activeElement).toBe(button("Add"));
      });

      it("should focus the outer trigger when both dropdowns close at once", async () => {
        await renderThemed(<Nested />);
        const trigger = button("Menu");
        trigger.focus();
        await userEvent.keyboard("{Enter}");
        button("Inner").focus();
        await userEvent.keyboard("{Enter}");
        button("Done").focus();
        await userEvent.keyboard("{Enter}");
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(trigger);
      });

      it("should leave Tab in a modal opened from it to the modal", async () => {
        await openDropdown(
          <>
            {SEARCH}
            <Dialog.Frame variant="modal">
              <Dialog.Trigger>Details</Dialog.Trigger>
              <Dialog.Dialog>{FIELDS}</Dialog.Dialog>
            </Dialog.Frame>
          </>,
        );
        button("Details").focus();
        await userEvent.keyboard("{Enter}");
        field("Last").focus();
        await userEvent.tab();
        expect(document.activeElement).toBe(field("First"));
        expect(page.getByLabelText("Search").query()).not.toBeNull();
      });

      it("should close when Tab moves the focus on from its trigger", async () => {
        const trigger = await openDropdown(SEARCH);
        trigger.focus();
        await userEvent.tab();
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(field("After"));
      });

      it("should close when a click on its trigger toggles it", async () => {
        const trigger = await openDropdown(SEARCH);
        await userEvent.click(trigger);
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
      });

      it("should stay open while its context menu holds the focus", async () => {
        await openDropdown(<MenuTarget />);
        await userEvent.click(page.getByLabelText("Search"), { button: "right" });
        await expect.element(page.getByRole("menu")).toHaveFocus();
        await userEvent.keyboard("{Escape}");
        await expect.element(page.getByLabelText("Search")).toHaveFocus();
        expect(page.getByRole("dialog").query()).not.toBeNull();
      });

      it("should stay open when the window gives the focus back", async () => {
        await renderThemed(
          <Triggers.Provider>
            <input aria-label="Before" />
            <Dialog.Frame id="menu">
              <Dialog.Trigger>Menu</Dialog.Trigger>
              <Dialog.Dialog>Content</Dialog.Dialog>
            </Dialog.Frame>
          </Triggers.Provider>,
        );
        const before = field("Before");
        before.focus();
        Dialog.open("menu");
        await expect.element(page.getByRole("dialog")).toBeVisible();
        const dialog = page.getByRole("dialog").element();
        await expect.poll(() => computed(dialog, "opacity")).toBe("1");
        before.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
        await userEvent.keyboard("{Shift}");
        expect(button("Menu").getAttribute("aria-expanded")).toBe("true");
      });

      it("should stay open while its anchor holds the focus", async () => {
        await renderThemed(<Anchored />);
        Dialog.open("anchored");
        await expect.element(page.getByLabelText("Search")).toHaveFocus();
        const dialog = page.getByRole("dialog").element();
        await expect.poll(() => computed(dialog, "opacity")).toBe("1");
        button("Anchor").focus();
        await userEvent.keyboard("{Shift}");
        const frame = document.getElementById("anchored");
        expect(frame?.classList.contains(CSS.visible(true))).toBe(true);
      });

      it("should leave the focus where a click outside puts it", async () => {
        await openDropdown(SEARCH);
        await userEvent.click(page.getByLabelText("Before"));
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(field("Before"));
      });
    });
  });

  describe("modal", () => {
    describe("focus", () => {
      it("should take the focus on open", async () => {
        const { dialog } = await openModal(FIELDS);
        expect(document.activeElement).toBe(dialog);
      });

      it("should leave the focus on a field that takes it on open", async () => {
        await openModal(<input aria-label="Name" autoFocus />);
        expect(document.activeElement).toBe(field("Name"));
      });

      it("should move Tab from the modal to its first field", async () => {
        await openModal(FIELDS);
        await userEvent.tab();
        expect(document.activeElement).toBe(field("First"));
      });

      it("should move Shift+Tab from the modal to its last field", async () => {
        await openModal(FIELDS);
        await userEvent.tab({ shift: true });
        expect(document.activeElement).toBe(field("Last"));
      });

      it("should wrap Tab from the last field to the first", async () => {
        await openModal(FIELDS, <button>Behind</button>);
        field("Last").focus();
        await userEvent.tab();
        expect(document.activeElement).toBe(field("First"));
      });

      it("should wrap Shift+Tab from the first field to the last", async () => {
        await openModal(FIELDS, <button>Behind</button>);
        field("First").focus();
        await userEvent.tab({ shift: true });
        expect(document.activeElement).toBe(field("Last"));
      });

      it("should stop on buttons and skip disabled and hidden controls", async () => {
        await openModal(
          <>
            <input aria-label="First" />
            <button disabled>Off</button>
            <input aria-label="Hidden" hidden />
            <button>Save</button>
          </>,
        );
        field("First").focus();
        await userEvent.tab();
        expect(document.activeElement).toBe(button("Save"));
      });

      it("should leave a Tab that a field handles to that field", async () => {
        await openModal(
          <>
            <input
              aria-label="Code"
              onKeyDown={(e) => {
                if (e.key === "Tab") e.preventDefault();
              }}
            />
            <input aria-label="Other" />
          </>,
        );
        field("Code").focus();
        await userEvent.tab();
        expect(document.activeElement).toBe(field("Code"));
      });

      it("should bring back focus that the page behind takes", async () => {
        await openModal(FIELDS, <button>Behind</button>);
        field("Last").focus();
        button("Behind").focus();
        expect(document.activeElement).toBe(field("Last"));
      });

      it("should leave the focus in a dropdown opened over the modal", async () => {
        await openModal(
          <Dialog.Frame>
            <Dialog.Trigger>Menu</Dialog.Trigger>
            <Dialog.Dialog>
              <input aria-label="Search" />
            </Dialog.Dialog>
          </Dialog.Frame>,
        );
        await userEvent.click(page.getByRole("button", { name: "Menu" }));
        field("Search").focus();
        expect(document.activeElement).toBe(field("Search"));
      });

      it("should go on in the modal when Tab leaves a dropdown", async () => {
        await openModal(DROPDOWN_FIELDS);
        button("Menu").focus();
        await userEvent.keyboard("{Enter}");
        await expect.element(page.getByLabelText("Search")).toHaveFocus();
        await userEvent.tab();
        await expect.element(page.getByLabelText("Search")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(field("Last"));
      });

      it("should focus the trigger when Shift+Tab leaves a dropdown", async () => {
        await openModal(DROPDOWN_FIELDS);
        const trigger = button("Menu");
        trigger.focus();
        await userEvent.keyboard("{Enter}");
        await expect.element(page.getByLabelText("Search")).toHaveFocus();
        await userEvent.tab({ shift: true });
        await expect.element(page.getByLabelText("Search")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(trigger);
      });

      it("should give the focus back to the opener on close", async () => {
        const { trigger } = await openModal(FIELDS);
        await userEvent.keyboard("{Escape}");
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(trigger);
      });

      it("should give the focus back to the opener after a field took it", async () => {
        const { trigger } = await openModal(<input aria-label="Name" autoFocus />);
        await userEvent.keyboard("{Escape}");
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(trigger);
      });

      it("should hold the focus only in the top modal", async () => {
        await openModal(
          <Dialog.Frame variant="modal">
            <Dialog.Trigger>Inner</Dialog.Trigger>
            <Dialog.Dialog>
              <input aria-label="Inner field" />
            </Dialog.Dialog>
          </Dialog.Frame>,
        );
        const inner = button("Inner");
        inner.focus();
        await userEvent.keyboard("{Enter}");
        await userEvent.tab();
        expect(document.activeElement).toBe(field("Inner field"));
        await userEvent.tab();
        expect(document.activeElement).toBe(field("Inner field"));
        await userEvent.keyboard("{Escape}");
        await expect
          .element(page.getByLabelText("Inner field"))
          .not.toBeInTheDocument();
        expect(document.activeElement).toBe(inner);
      });

      it("should give the focus to the first opener when a modal opens the next", async () => {
        await renderThemed(<Chain />);
        const start = button("Start");
        start.focus();
        await userEvent.keyboard("{Enter}");
        await userEvent.tab();
        await userEvent.keyboard("{Enter}");
        await expect.element(page.getByLabelText("Name")).toHaveFocus();
        await userEvent.keyboard("{Escape}");
        await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
        expect(document.activeElement).toBe(start);
      });
    });

    it("should tint the whole window behind it", async () => {
      const { dialog } = await openDialog("modal");
      const bg = dialog.closest(`.${CSS.BE("dialog", "bg")}`)!;
      const rect = bg.getBoundingClientRect();
      expect(rect.width).toBe(window.innerWidth);
      expect(rect.height).toBe(window.innerHeight);
      expect(computed(bg, "background-color")).toBe(resolve("var(--pluto-gray-l8-40)"));
    });

    it("should sit on top of its background on the modal shadow", async () => {
      const { dialog } = await openDialog("modal");
      expect(onTop(dialog)).toBe(true);
      expect(computed(dialog, "box-shadow")).toContain(
        resolve("var(--pluto-shadow-modal)", "box-shadow"),
      );
      expect(computed(dialog, "border-top-left-radius")).toBe(
        resolve("var(--pluto-border-radius-large)", "border-top-left-radius"),
      );
    });
  });
});
