// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { type CSSProperties, type ReactElement, type ReactNode, useState } from "react";
import { afterEach, assert, beforeEach, describe, expect, it, vi } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { PORTAL_ID_ATTR, PORTAL_OWNER_ATTR } from "@/dialog/useClickOutside";
import { Menu } from "@/menu";
import { Portal } from "@/portal";
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

/** A dropdown whose trigger grows to the left on a click in the dropdown. */
const Growing = (): ReactElement => {
  const [name, setName] = useState("Menu");
  return (
    <div style={{ display: "flex", justifyContent: "flex-end", width: "300px" }}>
      <Dialog.Frame>
        <Dialog.Trigger>{name}</Dialog.Trigger>
        <Dialog.Dialog>
          <button onClick={() => setName("A much longer menu")}>Grow</button>
        </Dialog.Dialog>
      </Dialog.Frame>
    </div>
  );
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
    const renderShadow = async (disabled = false): Promise<Element> => {
      const { getByText } = await renderThemed(
        <Dialog.Frame>
          <Dialog.Trigger variant="shadow" disabled={disabled}>
            Value
          </Dialog.Trigger>
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

    const caretOf = (trigger: Element): Element => {
      const caret = trigger.querySelector(`.${CSS.B("caret-animated")}`);
      assert(caret != null);
      return caret;
    };

    it("should hide its caret at rest", async () => {
      const trigger = await renderShadow();
      expect(computed(caretOf(trigger), "opacity")).toBe("0");
    });

    it("should show its caret on hover, in the color of every caret", async () => {
      const trigger = await renderShadow();
      await userEvent.hover(trigger);
      await expect.poll(() => computed(caretOf(trigger), "opacity")).toBe("1");
      expect(computed(caretOf(trigger), "color")).toBe(resolve("var(--pluto-gray-l9)"));
    });

    it("should show its caret on keyboard focus", async () => {
      const trigger = await renderShadow();
      assert(trigger instanceof HTMLElement);
      // Safari's Tab skips buttons, so a key press marks the focus as keyboard focus.
      await userEvent.keyboard("{Shift}");
      trigger.focus();
      expect(document.activeElement).toBe(trigger);
      await expect.poll(() => computed(caretOf(trigger), "opacity")).toBe("1");
    });

    it("should show its caret while open", async () => {
      const trigger = await renderShadow();
      await userEvent.click(trigger);
      await userEvent.unhover(trigger);
      await expect.poll(() => computed(caretOf(trigger), "opacity")).toBe("1");
    });

    it("should keep its caret hidden on hover while disabled", async () => {
      const trigger = await renderShadow(true);
      await userEvent.hover(trigger);
      expect(computed(caretOf(trigger), "opacity")).toBe("0");
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

    it("should follow its trigger as the trigger moves", async () => {
      await renderThemed(<Growing />);
      await userEvent.click(button("Menu"));
      const dialog = page.getByRole("dialog").element();
      await expect.poll(() => computed(dialog, "opacity")).toBe("1");
      const before = dialog.getBoundingClientRect().left;
      await userEvent.click(button("Grow"));
      const trigger = button("A much longer menu");
      await expect
        .poll(() => dialog.getBoundingClientRect().left)
        .toBeCloseTo(trigger.getBoundingClientRect().left, 0);
      expect(dialog.getBoundingClientRect().left).toBeLessThan(before);
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

describe("Dialog", () => {
  // These specs size their dialogs for a 1024 by 768 window.
  beforeEach(async () => {
    await page.viewport(1024, 768);
  });

  const rect = (left: number, top: number, width: number, height: number) =>
    ({
      left,
      top,
      width,
      height,
      right: left + width,
      bottom: top + height,
      x: left,
      y: top,
      toJSON: () => ({}),
    }) as DOMRect;

  it("should not display the dialog content by default", () => {
    const c = render(
      <Triggers.Provider>
        <Dialog.Frame>
          <Dialog.Trigger>Toggle</Dialog.Trigger>
          <Dialog.Dialog>
            <p>Content</p>
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    expect(c.getByText("Toggle")).toBeTruthy();
    expect(c.queryByText("Content")).toBeNull();
  });

  it("should display the dialog content when the trigger is clicked", () => {
    const c = render(
      <Triggers.Provider>
        <Dialog.Frame>
          <Dialog.Trigger>Toggle</Dialog.Trigger>
          <Dialog.Dialog>
            <p>Content</p>
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    fireEvent.click(c.getByText("Toggle"));
    expect(c.getByText("Content")).toBeTruthy();
  });

  it("should toggle the dialog when the trigger is clicked again", () => {
    const c = render(
      <Triggers.Provider>
        <Dialog.Frame>
          <Dialog.Trigger>Toggle</Dialog.Trigger>
          <Dialog.Dialog>
            <p>Content</p>
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    fireEvent.click(c.getByText("Toggle"));
    fireEvent.click(c.getByText("Toggle"));
    expect(c.queryByText("Content")).toBeNull();
  });

  it("should hide the dialog when the escape key is pressed", () => {
    const c = render(
      <Triggers.Provider>
        <Dialog.Frame>
          <Dialog.Trigger>Toggle</Dialog.Trigger>
          <Dialog.Dialog>
            <p>Content</p>
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    fireEvent.click(c.getByText("Toggle"));
    fireEvent.keyDown(c.container, { code: "Escape" });
    expect(c.queryByText("Content")).toBeNull();
  });

  describe("escape propagation", () => {
    const EscapeListener = ({ onEscape }: { onEscape: () => void }): null => {
      Triggers.use({
        triggers: [["Escape"]],
        callback: ({ stage }) => {
          if (stage === "start") onEscape();
        },
        loose: true,
      });
      return null;
    };

    it("should stop propagation to lower-priority Escape subscribers when a non-modal dialog closes", () => {
      const onEscape = vi.fn();
      const c = render(
        <Triggers.Provider>
          <EscapeListener onEscape={onEscape} />
          <Dialog.Frame variant="floating">
            <Dialog.Trigger>Toggle</Dialog.Trigger>
            <Dialog.Dialog>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      fireEvent.click(c.getByText("Toggle"));
      fireEvent.keyDown(c.container, { code: "Escape" });
      expect(c.queryByText("Content")).toBeNull();
      expect(onEscape).not.toHaveBeenCalled();
    });

    it("should stop propagation to lower-priority Escape subscribers when a modal dialog closes", () => {
      const onEscape = vi.fn();
      const c = render(
        <Triggers.Provider>
          <EscapeListener onEscape={onEscape} />
          <Dialog.Frame variant="modal">
            <Dialog.Trigger>Toggle</Dialog.Trigger>
            <Dialog.Dialog>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      fireEvent.click(c.getByText("Toggle"));
      fireEvent.keyDown(c.container, { code: "Escape" });
      expect(c.queryByText("Content")).toBeNull();
      expect(onEscape).not.toHaveBeenCalled();
    });

    it("should close nested floating dialogs one Escape at a time", () => {
      const c = render(
        <Triggers.Provider>
          <Dialog.Frame variant="floating">
            <Dialog.Trigger>Outer</Dialog.Trigger>
            <Dialog.Dialog>
              <p>Outer content</p>
              <Dialog.Frame variant="floating">
                <Dialog.Trigger>Inner</Dialog.Trigger>
                <Dialog.Dialog>
                  <p>Inner content</p>
                </Dialog.Dialog>
              </Dialog.Frame>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      fireEvent.click(c.getByText("Outer"));
      fireEvent.click(c.getByText("Inner"));
      fireEvent.keyDown(c.container, { code: "Escape" });
      expect(c.queryByText("Inner content")).toBeNull();
      expect(c.queryByText("Outer content")).not.toBeNull();
      fireEvent.keyUp(c.container, { code: "Escape" });
      fireEvent.keyDown(c.container, { code: "Escape" });
      expect(c.queryByText("Outer content")).toBeNull();
    });

    it("should let Escape propagate to lower-priority subscribers when no dialog is open", () => {
      const onEscape = vi.fn();
      const c = render(
        <Triggers.Provider>
          <EscapeListener onEscape={onEscape} />
          <Dialog.Frame variant="floating">
            <Dialog.Trigger>Toggle</Dialog.Trigger>
            <Dialog.Dialog>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      fireEvent.keyDown(c.container, { code: "Escape" });
      expect(onEscape).toHaveBeenCalledOnce();
    });
  });

  it("should link the portaled dialog back to its frame", () => {
    const c = render(
      <Triggers.Provider>
        <Dialog.Frame>
          <Dialog.Trigger>Toggle</Dialog.Trigger>
          <Dialog.Dialog>
            <p>Content</p>
          </Dialog.Dialog>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    fireEvent.click(c.getByText("Toggle"));
    const frame = c.container.querySelector(`[${PORTAL_ID_ATTR}]`);
    const id = frame?.getAttribute(PORTAL_ID_ATTR);
    expect(id).toBeTruthy();
    expect(c.getByRole("dialog").getAttribute(PORTAL_OWNER_ATTR)).toEqual(id);
  });

  describe("trigger accessibility", () => {
    it("should mark the trigger as the control of a collapsed dialog", () => {
      const c = render(
        <Triggers.Provider>
          <Dialog.Frame>
            <Dialog.Trigger>Toggle</Dialog.Trigger>
            <Dialog.Dialog>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      const trigger = c.getByRole("button", { expanded: false });
      expect(trigger.getAttribute("aria-haspopup")).toEqual("dialog");
    });

    it("should expand the trigger while the dialog is visible", () => {
      const c = render(
        <Triggers.Provider>
          <Dialog.Frame>
            <Dialog.Trigger>Toggle</Dialog.Trigger>
            <Dialog.Dialog>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      fireEvent.click(c.getByText("Toggle"));
      expect(c.getByRole("button", { expanded: true })).toBeTruthy();
    });

    it("should not announce a popup on a preview trigger", () => {
      const c = render(
        <Triggers.Provider>
          <Dialog.Frame>
            <Dialog.Trigger preview>Toggle</Dialog.Trigger>
            <Dialog.Dialog>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      const trigger = c.getByText("Toggle");
      expect(trigger.getAttribute("aria-haspopup")).toBeNull();
      expect(trigger.getAttribute("aria-expanded")).toBeNull();
    });
  });

  describe("variants", () => {
    const VARIANTS: Dialog.Variant[] = ["connected", "floating", "modal"];
    VARIANTS.forEach((variant) => {
      it(`should display a ${variant} dialog`, () => {
        const c = render(
          <Triggers.Provider>
            <Dialog.Frame variant={variant}>
              <Dialog.Trigger>Toggle</Dialog.Trigger>
              <Dialog.Dialog>
                <p>Content</p>
              </Dialog.Dialog>
            </Dialog.Frame>
          </Triggers.Provider>,
        );
        fireEvent.click(c.getByText("Toggle"));
        expect(c.getByText("Content")).toBeTruthy();
        expect(c.getByRole("dialog")).toBeTruthy();
        expect(c.getByRole("dialog").classList).toContain(`lyra--${variant}`);
      });
    });
  });
  describe("portal target", () => {
    const VARIANTS: Dialog.Variant[] = ["floating", "modal"];
    VARIANTS.forEach((variant) => {
      it(`should portal a ${variant} dialog into the root element by default`, () => {
        const root = document.createElement("div");
        root.id = "root";
        document.body.appendChild(root);
        const c = render(
          <Triggers.Provider>
            <Dialog.Frame variant={variant} initialVisible>
              <Dialog.Dialog>Content</Dialog.Dialog>
            </Dialog.Frame>
          </Triggers.Provider>,
        );
        expect(root.contains(c.getByRole("dialog"))).toBe(true);
        root.remove();
      });

      it(`should portal a ${variant} dialog into a Portal.Target`, () => {
        const target = document.createElement("div");
        document.body.appendChild(target);
        const c = render(
          <Triggers.Provider>
            <Portal.Target element={target}>
              <Dialog.Frame variant={variant} initialVisible>
                <Dialog.Dialog>Content</Dialog.Dialog>
              </Dialog.Frame>
            </Portal.Target>
          </Triggers.Provider>,
        );
        expect(target.contains(c.getByRole("dialog"))).toBe(true);
        target.remove();
      });
    });
  });
  describe("anchor", () => {
    afterEach(() => {
      vi.restoreAllMocks();
    });

    it("should place the dialog under the anchor instead of the frame", () => {
      const anchor = document.createElement("div");
      document.body.appendChild(anchor);
      vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (
        this: Element,
      ) {
        return this === anchor ? rect(400, 100, 100, 30) : rect(0, 0, 0, 0);
      });
      vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(200);
      vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(100);
      const c = render(
        <Triggers.Provider>
          <Dialog.Frame anchor={anchor} initialVisible>
            <Dialog.Dialog>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      act(() => {
        window.dispatchEvent(new Event("resize"));
      });
      const el = c.getByRole("dialog");
      expect(el.style.left).toBe("400px");
      expect(parseFloat(el.style.top)).toBeGreaterThanOrEqual(130);
      anchor.remove();
    });
  });

  describe("open", () => {
    it("should open the frame with the given id", () => {
      const c = render(
        <Triggers.Provider>
          <Dialog.Frame id="frame">
            <Dialog.Trigger>Toggle</Dialog.Trigger>
            <Dialog.Dialog>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      act(() => Dialog.open("frame"));
      expect(c.getByText("Content")).toBeTruthy();
    });

    it("should throw when the id names no frame", () => {
      render(<p id="paragraph">Text</p>);
      expect(() => Dialog.open("paragraph")).toThrow(
        "no dialog frame has the id paragraph",
      );
    });
  });

  describe("window fit", () => {
    afterEach(() => {
      vi.restoreAllMocks();
    });

    const open = (
      target: DOMRect,
      dialog: DOMRect,
      variant?: Dialog.Variant,
      style?: CSSProperties,
    ): HTMLElement => {
      vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (
        this: Element,
      ) {
        if (this.classList.contains("lyra-dialog__frame")) return target;
        return rect(0, 0, 0, 0);
      });
      const isDialog = (el: HTMLElement): boolean =>
        el.classList.contains("lyra-dialog__dialog");
      vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockImplementation(
        function (this: HTMLElement) {
          return isDialog(this) ? dialog.width : 0;
        },
      );
      vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockImplementation(
        function (this: HTMLElement) {
          return isDialog(this) ? dialog.height : 0;
        },
      );
      const c = render(
        <Triggers.Provider>
          <Dialog.Frame variant={variant}>
            <Dialog.Trigger>Toggle</Dialog.Trigger>
            <Dialog.Dialog style={style}>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      fireEvent.click(c.getByText("Toggle"));
      // A window resize places the dialog before the resize observers fire.
      act(() => {
        window.dispatchEvent(new Event("resize"));
      });
      return c.getByRole("dialog");
    };

    const variable = (el: HTMLElement, name: string): string =>
      el.style.getPropertyValue(`--pluto-dialog-${name}`);

    it("should keep a dialog wider than the window inside its edges", () => {
      const el = open(rect(400, 100, 100, 30), rect(0, 0, 1100, 100));
      expect(el.style.left).toBe("6px");
      expect(variable(el, "available-width")).toBe(`${window.innerWidth - 12}px`);
    });

    it("should limit a dialog below its trigger to the space under it", () => {
      const el = open(rect(400, 100, 100, 30), rect(0, 0, 200, 100));
      const top = parseFloat(el.style.top);
      expect(top).toBeGreaterThanOrEqual(130);
      expect(variable(el, "available-height")).toBe(
        `${window.innerHeight - top - 6}px`,
      );
    });

    it("should limit a dialog above its trigger to the space over it", () => {
      const bottomEdge = window.innerHeight - 40;
      const el = open(rect(400, bottomEdge, 100, 30), rect(0, 0, 200, 300));
      const bottom = parseFloat(el.style.bottom);
      expect(bottom).toBeGreaterThanOrEqual(window.innerHeight - bottomEdge);
      expect(variable(el, "available-height")).toBe(
        `${window.innerHeight - bottom - 6}px`,
      );
    });

    const resize = (): void => {
      act(() => {
        window.dispatchEvent(new Event("resize"));
      });
    };

    it("should clamp again when the dialog widens in place", () => {
      const el = open(rect(400, 100, 100, 30), rect(0, 0, 200, 100));
      expect(el.style.left).toBe("400px");
      vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockImplementation(
        function (this: HTMLElement) {
          return this.classList.contains("lyra-dialog__dialog") ? 620 : 0;
        },
      );
      resize();
      expect(el.style.left).toBe(`${window.innerWidth - 6 - 620}px`);
    });

    it("should shrink the available height when the window shrinks", () => {
      const el = open(rect(400, 100, 100, 30), rect(0, 0, 200, 100));
      const top = parseFloat(el.style.top);
      vi.spyOn(window, "innerHeight", "get").mockReturnValue(500);
      resize();
      expect(el.style.top).toBe(`${top}px`);
      expect(variable(el, "available-height")).toBe(`${500 - top - 6}px`);
    });

    describe("a connected dialog with a min-width", () => {
      const MIN_WIDTH: CSSProperties = { minWidth: "300px" };

      it("should widen to its min-width past a narrower trigger", () => {
        const el = open(
          rect(400, 100, 100, 30),
          rect(0, 0, 100, 100),
          "connected",
          MIN_WIDTH,
        );
        expect(el.style.width).toBe("300px");
      });

      it("should match a trigger wider than its min-width", () => {
        const el = open(
          rect(400, 100, 400, 30),
          rect(0, 0, 400, 100),
          "connected",
          MIN_WIDTH,
        );
        expect(el.style.width).toBe("400px");
      });

      it("should keep a widened dialog inside the window edge", () => {
        const el = open(
          rect(window.innerWidth - 120, 100, 100, 30),
          rect(0, 0, 100, 100),
          "connected",
          MIN_WIDTH,
        );
        expect(el.style.width).toBe("300px");
        expect(parseFloat(el.style.left)).toBeLessThanOrEqual(
          window.innerWidth - 6 - 300,
        );
      });
    });

    describe("a dialog that outgrows its space", () => {
      // Stands in for the CSS limits: the dialog never grows past its own height
      // limit or the written space.
      const MAX_HEIGHT = 360;
      const grow = (natural: number): void => {
        vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockImplementation(
          function (this: HTMLElement) {
            if (!this.classList.contains("lyra-dialog__dialog")) return 0;
            const available = parseFloat(variable(this, "available-height"));
            return Math.min(
              natural,
              MAX_HEIGHT,
              Number.isNaN(available) ? Infinity : available,
            );
          },
        );
        resize();
      };

      it("should move above the trigger when there is more room above", () => {
        const el = open(rect(400, 500, 100, 30), rect(0, 0, 200, 100));
        expect(el.style.top).not.toBe("");
        grow(300);
        expect(el.style.top).toBe("");
        expect(parseFloat(el.style.bottom)).toBeGreaterThanOrEqual(
          window.innerHeight - 500,
        );
      });

      it("should stay below the trigger when there is more room below", () => {
        // The dialog fits on both sides, so only the order of the sides decides.
        vi.spyOn(window, "innerHeight", "get").mockReturnValue(465);
        const el = open(rect(400, 215, 100, 30), rect(0, 0, 200, 100));
        grow(2000);
        expect(el.style.bottom).toBe("");
        expect(parseFloat(el.style.top)).toBeGreaterThanOrEqual(245);
      });

      it("should stay below the trigger when the dialog fits", () => {
        const el = open(rect(400, 500, 100, 30), rect(0, 0, 200, 100));
        grow(150);
        expect(el.style.bottom).toBe("");
        expect(parseFloat(el.style.top)).toBeGreaterThanOrEqual(530);
      });
    });
  });
});
