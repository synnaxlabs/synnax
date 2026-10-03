// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { PORTAL_ID_ATTR, PORTAL_OWNER_ATTR } from "@/dialog/useClickOutside";
import { Triggers } from "@/triggers";

describe("Dialog", () => {
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
        expect(c.getByRole("dialog").classList).toContain(`pluto--${variant}`);
      });
    });
  });
  describe("window fit", () => {
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

    afterEach(() => {
      vi.restoreAllMocks();
    });

    const open = (target: DOMRect, dialog: DOMRect): HTMLElement => {
      vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (
        this: Element,
      ) {
        if (this.classList.contains("pluto-dialog__frame")) return target;
        return rect(0, 0, 0, 0);
      });
      const isDialog = (el: HTMLElement): boolean =>
        el.classList.contains("pluto-dialog__dialog");
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
          <Dialog.Frame>
            <Dialog.Trigger>Toggle</Dialog.Trigger>
            <Dialog.Dialog>
              <p>Content</p>
            </Dialog.Dialog>
          </Dialog.Frame>
        </Triggers.Provider>,
      );
      fireEvent.click(c.getByText("Toggle"));
      // jsdom never fires the resize observers, so a window resize places the dialog.
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
  });
});
