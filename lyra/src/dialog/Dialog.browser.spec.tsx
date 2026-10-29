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

import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { computed, renderThemed, resolve } from "@/styletest";
import { Triggers } from "@/triggers";

const TALL = <div style={{ height: 2000, flexShrink: 0 }}>Tall</div>;

const openDialog = async (
  variant: Dialog.FrameProps["variant"],
  content: ReactNode = <p>Content</p>,
  size?: Dialog.DialogProps["size"],
): Promise<{ trigger: Element; dialog: Element; unmount: () => unknown }> => {
  const { getByText, unmount } = await renderThemed(
    <Triggers.Provider>
      <Dialog.Frame variant={variant}>
        <Dialog.Trigger>Toggle</Dialog.Trigger>
        <Dialog.Dialog size={size}>{content}</Dialog.Dialog>
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
  const { dialog, unmount } = await openDialog("floating", TALL, size);
  const { height } = dialog.getBoundingClientRect();
  await unmount();
  return height;
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
      expect(await capOf()).toBeLessThan(2000);
    });

    it("should cap a small dialog lower and a large one higher", async () => {
      const medium = await capOf();
      expect(await capOf("small")).toBeLessThan(medium);
      expect(await capOf("large")).toBeGreaterThan(medium);
    });
  });

  describe("modal", () => {
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
