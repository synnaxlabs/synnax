// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

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
});
