// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Dialog } from "@/dialog";
import { computed, renderThemed, resolve } from "@/testutil/style";
import { Triggers } from "@/triggers";

const openDialog = async (
  variant: Dialog.FrameProps["variant"],
  content: ReactNode = <p>Content</p>,
): Promise<{ trigger: Element; dialog: Element }> => {
  const { getByText } = await renderThemed(
    <Triggers.Provider>
      <Dialog.Frame variant={variant}>
        <Dialog.Trigger>Toggle</Dialog.Trigger>
        <Dialog.Dialog>{content}</Dialog.Dialog>
      </Dialog.Frame>
    </Triggers.Provider>,
  );
  const trigger = getByText("Toggle").element();
  await userEvent.click(trigger);
  await expect.element(page.getByRole("dialog")).toBeVisible();
  return { trigger, dialog: page.getByRole("dialog").element() };
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

describe("Dialog", () => {
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
      const { dialog } = await openDialog(
        "floating",
        <div style={{ height: 2000, flexShrink: 0 }}>Tall</div>,
      );
      expect(dialog.getBoundingClientRect().height).toBeLessThan(2000);
    });
  });

  describe("modal", () => {
    it("should tint the whole window behind it", async () => {
      const { dialog } = await openDialog("modal");
      const bg = dialog.parentElement!;
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
