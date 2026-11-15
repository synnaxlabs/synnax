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
import { Menu } from "@/menu";
import { Select } from "@/select";
import { computed, height, renderThemed, resolve } from "@/styletest";

const openMenu = async (): Promise<HTMLElement> => {
  await renderThemed(
    <Dialog.Frame initialVisible>
      <Dialog.Trigger>Filter</Dialog.Trigger>
      <Menu.Dialog>
        <Menu.Menu>
          <Menu.Item itemKey="labels">Labels</Menu.Item>
        </Menu.Menu>
      </Menu.Dialog>
    </Dialog.Frame>,
  );
  const dialog = page.getByRole("dialog").element() as HTMLElement;
  await expect.poll(() => computed(dialog, "opacity")).toBe("1");
  return dialog;
};

const openSelect = async (): Promise<HTMLElement> => {
  await renderThemed(
    <Dialog.Frame initialVisible>
      <Select.Frame<string, undefined>
        multiple
        data={["low", "high"]}
        value={["low", "high"]}
        onChange={() => {}}
      >
        <Dialog.Trigger>Labels</Dialog.Trigger>
        <Select.Dialog surface="menu">
          <Select.Body<string> resourceName="label" onSearch={() => {}}>
            {({ key, ...p }) => (
              <Select.Item key={key} {...p}>
                {key}
              </Select.Item>
            )}
          </Select.Body>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>,
  );
  const dialog = page.getByRole("dialog").element() as HTMLElement;
  await expect.poll(() => computed(dialog, "opacity")).toBe("1");
  return dialog;
};

describe("Menu.Dialog", () => {
  it("should draw the surface of a context menu", async () => {
    const dialog = await openMenu();
    expect(computed(dialog, "background-color")).toBe(
      resolve("var(--pluto-gray-l1-l2)"),
    );
    expect(computed(dialog, "border-top-color")).toBe(resolve("var(--pluto-gray-l6)"));
    expect(computed(dialog, "border-top-width")).toBe("1px");
    expect(computed(dialog, "border-top-left-radius")).toBe(
      resolve("var(--pluto-border-radius-large)", "border-top-left-radius"),
    );
    expect(computed(dialog, "padding-top")).toBe("6px");
    expect(computed(dialog, "box-shadow")).toBe(
      resolve("var(--pluto-shadow-overlay)", "box-shadow"),
    );
  });

  it("should draw its items at the menu size", async () => {
    await openMenu();
    const item = page.getByRole("menuitem", { name: "Labels" }).element();
    expect(height(item)).toBe(28);
    expect(computed(item, "font-size")).toBe("13px");
  });

  it("should light an item on hover with the context menu fill", async () => {
    await openMenu();
    const item = page.getByRole("menuitem", { name: "Labels" }).element();
    await userEvent.hover(item);
    expect(computed(item, "background-color")).toBe(resolve("var(--pluto-gray-l4)"));
  });

  describe("selection", () => {
    it("should draw its search as a line of text over a hairline", async () => {
      const dialog = await openSelect();
      const search = dialog.querySelector(".lyra-input") as HTMLElement;
      expect(height(search)).toBe(36);
      expect(computed(search, "background-color")).toBe(resolve("transparent"));
      expect(computed(search, "border-top-width")).toBe("0px");
      expect(computed(search, "border-bottom-width")).toBe("1px");
      expect(computed(search, "border-bottom-color")).toBe(
        resolve("var(--pluto-gray-l6)"),
      );
      expect(computed(search, "font-size")).toBe("13px");
    });

    it("should span the search across the panel", async () => {
      const dialog = await openSelect();
      const search = dialog.querySelector(".lyra-input") as HTMLElement;
      const panel = dialog.getBoundingClientRect();
      const line = search.getBoundingClientRect();
      expect(line.left).toBe(panel.left + 1);
      expect(line.right).toBe(panel.right - 1);
      expect(line.top).toBe(panel.top + 1);
    });

    it("should list its options with no box, at the menu size", async () => {
      const dialog = await openSelect();
      const scroll = dialog.querySelector(".lyra-list__scroll") as HTMLElement;
      expect(computed(scroll, "border-top-width")).toBe("0px");
      expect(computed(scroll, "padding-top")).toBe("0px");
      const [low, high] = page.getByRole("option").elements();
      expect(height(low)).toBe(28);
      expect(computed(low, "font-size")).toBe("13px");
      expect(computed(low, "border-top-left-radius")).toBe(
        resolve("var(--pluto-border-radius-tiny)", "border-top-left-radius"),
      );
      expect(computed(low, "border-bottom-left-radius")).toBe("0px");
      expect(computed(high, "border-top-left-radius")).toBe("0px");
    });
  });
});
