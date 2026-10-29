// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Form } from "@/form";
import { Input } from "@/input";
import { computed, renderThemed, resolve } from "@/styletest";

const renderSections = async (
  x: boolean,
): Promise<{ inputs: Element[]; sections: Element[] }> => {
  const { getByRole, getByText } = await renderThemed(
    <Form.Sections x={x}>
      <Form.Section title="General">
        <Input.Item label="Name">
          <Input.Text value="" onChange={() => {}} />
        </Input.Item>
      </Form.Section>
      <Form.Section title="Timing">
        <Input.Item label="A much longer label">
          <Input.Text value="" onChange={() => {}} />
        </Input.Item>
      </Form.Section>
    </Form.Sections>,
  );
  return {
    inputs: getByRole("textbox")
      .elements()
      .map((input) => input.closest(`.${CSS.B("input")}`)!),
    sections: ["General", "Timing"].map((title) =>
      getByText(title)
        .element()
        .closest(`.${CSS.B("form-section")}`)!,
    ),
  };
};

describe("Form", () => {
  describe("stacked sections", () => {
    it("should line up every control in one column past the widest label", async () => {
      const { inputs } = await renderSections(false);
      const [a, b] = inputs.map((el) => el.getBoundingClientRect());
      expect(a.left).toBe(b.left);
      const label = document.getElementById(
        inputs[1].querySelector("input")!.getAttribute("aria-labelledby")!,
      )!;
      expect(label.getBoundingClientRect().right).toBeLessThanOrEqual(b.left);
    });

    it("should rule a hairline after each section title", async () => {
      const { sections } = await renderSections(false);
      const header = sections[0].querySelector(`.${CSS.BE("form-section", "header")}`)!;
      expect(getComputedStyle(header, "::after").backgroundColor).toBe(
        resolve("var(--pluto-gray-l6)"),
      );
    });
  });

  describe("side by side sections", () => {
    it("should place each section after the last with a divider", async () => {
      const { sections } = await renderSections(true);
      const [a, b] = sections.map((el) => el.getBoundingClientRect());
      expect(b.left).toBeGreaterThanOrEqual(a.right);
      expect(computed(sections[1], "border-left-color")).toBe(
        resolve("var(--pluto-gray-l6)"),
      );
      expect(computed(sections[0], "border-left-style")).toBe("none");
    });
  });
});
