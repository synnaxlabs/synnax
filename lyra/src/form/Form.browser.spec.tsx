// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Flex } from "@/flex";
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

    it("should line up an aligned child with the controls, not the labels", async () => {
      const { getByRole, getByText } = await renderThemed(
        <Form.Sections>
          <Form.Section title="General">
            <Input.Item label="A long label">
              <Input.Text value="" onChange={() => {}} />
            </Input.Item>
            <Form.Aligned>
              <span>Aligned note</span>
            </Form.Aligned>
          </Form.Section>
        </Form.Sections>,
      );
      const input = getByRole("textbox")
        .element()
        .closest(`.${CSS.B("input")}`)!;
      const aligned = getByText("Aligned note").element();
      expect(aligned.getBoundingClientRect().left).toBe(
        input.getBoundingClientRect().left,
      );
    });

    it("should rule a hairline after each section title", async () => {
      const { sections } = await renderSections(false);
      const header = sections[0].querySelector(`.${CSS.BE("form-section", "header")}`)!;
      expect(getComputedStyle(header, "::after").backgroundColor).toBe(
        resolve("var(--pluto-gray-l6)"),
      );
    });

    describe("rows", () => {
      const field = (label: string): ReactElement => (
        <Input.Item label={label}>
          <Input.Numeric value={0} onChange={() => {}} />
        </Input.Item>
      );

      const textField = (label: string): ReactElement => (
        <Input.Item label={label}>
          <Input.Text value="" onChange={() => {}} />
        </Input.Item>
      );

      const rect = (label: string): { label: DOMRect; control: DOMRect } => {
        const el = [...document.querySelectorAll(`.${CSS.B("input-label")}`)].find(
          (l) => l.textContent.trim() === label,
        )!;
        return {
          label: el.getBoundingClientRect(),
          control: el.nextElementSibling!.firstElementChild!.getBoundingClientRect(),
        };
      };

      it("should keep the first label in the label column and the rest on its line", async () => {
        await renderThemed(
          <Form.Sections>
            <Form.Section>
              {textField("Transport")}
              <Flex.Box x>
                {textField("Host")}
                {field("Port")}
              </Flex.Box>
            </Form.Section>
          </Form.Sections>,
        );
        const transport = rect("Transport");
        const host = rect("Host");
        const port = rect("Port");
        expect(host.label.left).toBe(transport.label.left);
        expect(host.control.left).toBe(transport.control.left);
        expect(port.label.top).toBe(host.label.top);
        expect(port.label.left).toBeGreaterThan(host.control.right);
        expect(port.control.right).toBe(transport.control.right);
      });

      it("should give a number field one width wherever it sits", async () => {
        await renderThemed(
          <Form.Sections>
            <Form.Section>
              {textField("Host")}
              {field("Unit ID")}
              <Flex.Box x>
                {field("Timeout")}
                {field("Retries")}
              </Flex.Box>
            </Form.Section>
          </Form.Sections>,
        );
        const unit = rect("Unit ID").control;
        expect(unit.width).toBeLessThan(rect("Host").control.width);
        expect(rect("Timeout").control.width).toBe(unit.width);
        expect(rect("Retries").control.width).toBe(unit.width);
      });

      it("should line up the later fields of every row", async () => {
        await renderThemed(
          <Form.Sections>
            <Form.Section>
              <Flex.Box x>
                {field("Timeout")}
                {field("Retries")}
                {field("Request delay")}
              </Flex.Box>
              <Flex.Box x>
                {field("Max registers")}
                {field("Max gap")}
              </Flex.Box>
            </Form.Section>
          </Form.Sections>,
        );
        const delay = rect("Request delay");
        const gap = rect("Max gap");
        expect(gap.label.left).toBe(delay.label.left);
        expect(gap.control.left).toBe(delay.control.left);
        expect(rect("Retries").label.left).toBeGreaterThan(
          rect("Timeout").control.right,
        );
      });

      it("should keep the first control when the later fields do not fit", async () => {
        await renderThemed(
          <div style={{ width: "60rem" }}>
            <Form.Sections>
              <Form.Section>
                <Flex.Box x>
                  {field("Timeout")}
                  {field("Retries")}
                  {field("Request delay")}
                </Flex.Box>
              </Form.Section>
            </Form.Sections>
          </div>,
        );
        const timeout = rect("Timeout").control;
        expect(timeout.width).toBeGreaterThan(0);
        expect(rect("Retries").label.left).toBeGreaterThan(timeout.right);
      });
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
