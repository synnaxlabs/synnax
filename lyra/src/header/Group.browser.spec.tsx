// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";
import { userEvent } from "vitest/browser";

import { Button } from "@/button";
import { Header } from "@/header";
import { Icon } from "@/icon";
import { computed, renderThemed, resolve, withMotion } from "@/styletest";

const renderGroup = async (onOpenChange: (open: boolean) => void = () => {}) => {
  const r = await renderThemed(
    <Header.Group
      open
      onOpenChange={onOpenChange}
      actions={
        <Button.Button variant="text" aria-label="Create">
          <Icon.Add />
        </Button.Button>
      }
    >
      Plant
    </Header.Group>,
  );
  const toggle = r.getByRole("button", { name: "Plant" }).element();
  const bar = toggle.parentElement as HTMLElement;
  const caret = bar.lastElementChild as Element;
  const action = r.getByRole("button", { name: "Create" }).element();
  return { bar, toggle, caret, action };
};

const background = (el: Element): string => computed(el, "background-color");

describe("Header.Group", () => {
  it("should fill with gray l2 and round like a list item", async () => {
    const { bar } = await renderGroup();
    expect(background(bar)).toBe(resolve("var(--pluto-gray-l2)"));
    expect(computed(bar, "border-top-left-radius")).toBe(
      resolve("var(--pluto-border-radius-small)", "border-top-left-radius"),
    );
  });

  it("should fill the whole bar while its toggle is hovered", async () => {
    const { bar, toggle } = await renderGroup();
    await userEvent.hover(toggle);
    expect(background(bar)).toBe(resolve("var(--pluto-fill-hover)"));
    expect(background(toggle)).toBe(resolve("transparent"));
  });

  it("should change its fill with the transition of a button", async () => {
    const { bar, action } = await renderGroup();
    withMotion(() => {
      for (const property of [
        "transition-property",
        "transition-duration",
        "transition-timing-function",
      ])
        expect(computed(bar, property)).toBe(computed(action, property));
    });
  });

  it("should keep its rest fill while an action is hovered", async () => {
    const { bar, action } = await renderGroup();
    await userEvent.hover(action);
    expect(background(bar)).toBe(resolve("var(--pluto-gray-l2)"));
  });

  it("should close the group from a click on the caret", async () => {
    const onOpenChange = vi.fn();
    const { bar, caret } = await renderGroup(onOpenChange);
    const b = bar.getBoundingClientRect();
    const c = caret.getBoundingClientRect();
    await userEvent.click(bar, {
      position: { x: c.left - b.left + c.width / 2, y: c.top - b.top + c.height / 2 },
    });
    expect(onOpenChange).toHaveBeenCalledExactlyOnceWith(false);
  });
});
