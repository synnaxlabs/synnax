// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { computed, height, renderThemed, resolve } from "@/styletest";

const renderButton = async (
  props: Partial<Button.ButtonProps> = {},
): Promise<HTMLElement | SVGElement> =>
  (await renderThemed(<Button.Button {...props}>Save</Button.Button>))
    .getByRole("button")
    .element();

const background = (el: Element): string => computed(el, "background-color");
const border = (el: Element): string => computed(el, "border-top-color");

describe("Button", () => {
  describe("variants", () => {
    it.each([
      ["outlined", "--pluto-fill-rest", "--pluto-gray-l7"],
      ["filled", "--pluto-primary-z", "--pluto-primary-z"],
    ] as const)(
      "should paint an %s button from its rest tokens",
      async (variant, bg, bc) => {
        const el = await renderButton({ variant });
        expect(background(el)).toBe(resolve(`var(${bg})`));
        expect(border(el)).toBe(resolve(`var(${bc})`));
      },
    );

    it.each([
      ["outlined", "--pluto-fill-hover", "--pluto-gray-l8"],
      ["filled", "--pluto-primary-p1", "--pluto-primary-p1"],
    ] as const)(
      "should paint a hovered %s button from its hover tokens",
      async (variant, bg, bc) => {
        const el = await renderButton({ variant });
        await userEvent.hover(el);
        expect(background(el)).toBe(resolve(`var(${bg})`));
        expect(border(el)).toBe(resolve(`var(${bc})`));
      },
    );

    it("should leave a text button transparent and without a border", async () => {
      const el = await renderButton({ variant: "text" });
      expect(background(el)).toBe(resolve("transparent"));
      expect(computed(el, "border-top-style")).toBe("none");
    });

    it("should label a filled button with the text color on primary", async () => {
      const el = await renderButton({ variant: "filled" });
      expect(computed(el, "color")).toBe(resolve("var(--pluto-text-on-primary)"));
    });

    it("should fill an error button with the error color", async () => {
      const el = await renderButton({ variant: "filled", status: "error" });
      expect(background(el)).toBe(resolve("var(--pluto-error-z)"));
    });
  });

  describe("disabled", () => {
    it("should keep its rest fill on hover", async () => {
      const el = await renderButton({ disabled: true });
      const rest = background(el);
      await userEvent.hover(el, { force: true });
      expect(background(el)).toBe(rest);
      expect(computed(el, "cursor")).toBe("not-allowed");
    });

    it("should dim its label", async () => {
      const el = await renderButton({ disabled: true });
      expect(computed(el, "color")).toBe(resolve("var(--pluto-text-disabled)"));
    });
  });

  describe("sizes", () => {
    it.each<Component.Size>(["tiny", "small", "medium", "large", "huge"])(
      "should be as tall as the %s height token",
      async (size) => {
        const el = await renderButton({ size });
        expect(height(el)).toBe(
          parseFloat(resolve(`var(--pluto-height-${size})`, "height")),
        );
      },
    );

    it("should be square when it holds only an icon", async () => {
      const { getByRole } = await renderThemed(
        <Button.Button aria-label="add">
          <Icon.Add />
        </Button.Button>,
      );
      const el = getByRole("button").element();
      const rect = el.getBoundingClientRect();
      expect(rect.width).toBe(rect.height);
    });
  });

  it("should draw its focus ring in the primary color", async () => {
    const el = await renderButton();
    // Safari's Tab key skips buttons, so a key press marks the focus as keyboard focus.
    await userEvent.keyboard("{Shift}");
    el.focus();
    expect(document.activeElement).toBe(el);
    expect(computed(el, "outline-color")).toBe(resolve("var(--pluto-primary-z)"));
    expect(computed(el, "outline-style")).toBe("solid");
  });

  it("should paint a glyph button's icon from the glyph colors", async () => {
    const { getByRole } = await renderThemed(
      <Button.Button variant="glyph" aria-label="close">
        <Icon.Close />
      </Button.Button>,
    );
    const el = getByRole("button").element();
    const icon = el.querySelector("svg")!;
    expect(computed(icon, "fill")).toBe(resolve("var(--pluto-gray-l9)"));
    await userEvent.hover(el);
    expect(computed(icon, "fill")).toBe(resolve("var(--pluto-gray-l11)"));
  });

  it("should never fill or border a glyph button", async () => {
    const el = await renderButton({ variant: "glyph" });
    expect(computed(el, "border-top-style")).toBe("none");
    await userEvent.hover(el);
    expect(background(el)).toBe(resolve("transparent"));
  });

  it("should rise above its packed neighbor while hovered", async () => {
    const { getByRole } = await renderThemed(
      <Flex.Box x pack>
        <Button.Button>First</Button.Button>
        <Button.Button>Second</Button.Button>
      </Flex.Box>,
    );
    const first = getByRole("button", { name: "First" }).element();
    const second = getByRole("button", { name: "Second" }).element();
    await userEvent.hover(first);
    expect(Number(computed(first, "z-index"))).toBeGreaterThan(
      Number(computed(second, "z-index")),
    );
  });

  it("should hide a reveal button until its container is hovered", async () => {
    const { getByRole, getByTestId } = await renderThemed(
      <Flex.Box className={CSS.M("reveals")} data-testid="host">
        <Button.Button reveal>Remove</Button.Button>
      </Flex.Box>,
    );
    const el = getByRole("button").element();
    expect(computed(el, "opacity")).toBe("0");
    await userEvent.hover(getByTestId("host"));
    expect(computed(el, "opacity")).toBe("1");
  });
});
