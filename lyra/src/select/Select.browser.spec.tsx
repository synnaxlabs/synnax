// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Select } from "@/select";
import { computed, renderThemed, resolve } from "@/styletest";

const renderButtons = async (
  variant: Select.ButtonsProps["variant"],
): Promise<(name: string) => Element> => {
  const { getByRole } = await renderThemed(
    <Select.Buttons value={1} onChange={() => {}} variant={variant}>
      <Select.Item itemKey={1}>One</Select.Item>
      <Select.Item itemKey={2}>A much longer second option</Select.Item>
    </Select.Buttons>,
  );
  return (name) => getByRole("button", { name }).element();
};

describe("Select", () => {
  describe("text buttons", () => {
    it("should mark the chosen button in the subtle selection tier", async () => {
      const button = await renderButtons("text");
      const chosen = button("One");
      expect(computed(chosen, "background-color")).toBe(
        resolve("var(--pluto-selected-subtle-bg)"),
      );
      expect(computed(chosen, "color")).toBe(
        resolve("var(--pluto-selected-subtle-color)"),
      );
      expect(computed(chosen, "box-shadow")).toContain(
        resolve("var(--pluto-selected-subtle-border-color)"),
      );
    });

    it("should shade the other buttons", async () => {
      const button = await renderButtons("text");
      expect(computed(button("A much longer second option"), "color")).toBe(
        resolve("var(--pluto-gray-l9)"),
      );
    });
  });

  describe("cards", () => {
    it("should share the width equally", async () => {
      const button = await renderButtons("card");
      expect(button("One").getBoundingClientRect().width).toBe(
        button("A much longer second option").getBoundingClientRect().width,
      );
    });

    it("should rest an unchosen card on the page color", async () => {
      const button = await renderButtons("card");
      const card = button("A much longer second option");
      expect(computed(card, "background-color")).toBe(resolve("var(--pluto-gray-l0)"));
      expect(computed(card, "border-top-color")).toBe(resolve("var(--pluto-gray-l6)"));
    });
  });
});
