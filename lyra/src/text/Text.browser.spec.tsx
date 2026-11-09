// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Icon } from "@/icon";
import { computed, renderThemed, resolve } from "@/styletest";
import { Text } from "@/text";

const LEVELS: text.Level[] = ["h1", "h2", "h3", "h4", "h5", "p", "small", "tiny"];

describe("Text", () => {
  it.each(LEVELS)(
    "should set the type of a %s from its theme tokens",
    async (level) => {
      const { getByText } = await renderThemed(
        <Text.Text level={level}>Label</Text.Text>,
      );
      const el = getByText("Label").element();
      for (const [property, token] of [
        ["font-size", "size"],
        ["line-height", "line-height"],
        ["font-weight", "weight"],
      ])
        expect(computed(el, property), property).toBe(
          resolve(`var(--pluto-${level}-${token})`, property),
        );
    },
  );

  it("should shrink the font size at each level from h1 to tiny", async () => {
    const { getByText } = await renderThemed(
      LEVELS.map((level) => (
        <Text.Text key={level} level={level}>
          {level}
        </Text.Text>
      )),
    );
    const sizes = LEVELS.map((level) =>
      parseFloat(computed(getByText(level, { exact: true }).element(), "font-size")),
    );
    for (let i = 1; i < sizes.length; i++) expect(sizes[i]).toBeLessThan(sizes[i - 1]);
  });

  it("should color text with the theme's text color", async () => {
    const { getByText } = await renderThemed(<Text.Text>Label</Text.Text>);
    const el = getByText("Label").element();
    expect(computed(el, "color")).toBe(resolve("var(--pluto-text-color)"));
  });

  describe("fade", () => {
    it("should keep a gap between an icon and its label", async () => {
      const { getByText } = await renderThemed(
        <Text.Text overflow="fade">
          <Icon.Range />
          Label
        </Text.Text>,
      );
      const el = getByText("Label").element();
      const label = document.createRange();
      label.selectNode(el.lastChild as Node);
      const icon = el.querySelector("svg") as SVGElement;
      expect(label.getBoundingClientRect().left).toBeGreaterThan(
        icon.getBoundingClientRect().right,
      );
    });

    it("should clip a label wider than its box", async () => {
      const { getByText } = await renderThemed(
        <div style={{ width: "5rem" }}>
          <Text.Text overflow="fade">A label far wider than its box</Text.Text>
        </div>,
      );
      const el = getByText("A label far wider than its box").element();
      expect(computed(el, "overflow-x")).toBe("hidden");
      expect(el.scrollWidth).toBeGreaterThan(el.clientWidth);
    });
  });
});
