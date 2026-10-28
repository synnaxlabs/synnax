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
import { CSS } from "@/css";
import { Input } from "@/input";
import { computed, height, renderThemed, resolve } from "@/styletest";

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
});
