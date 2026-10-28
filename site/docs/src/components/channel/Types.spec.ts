// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import reactRenderer from "@astrojs/react/server.js";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { type Element, type Root } from "hast";
import { fromHtml } from "hast-util-from-html";
import { beforeAll, describe, expect, it } from "vitest";

import { Channel } from "@/components/channel";

interface TypeProps {
  name?: string;
  type?: string;
  group?: string;
}

const VOLTAGE: TypeProps = { name: "Voltage", type: "ai_voltage" };
const CURRENT: TypeProps = { name: "Current", type: "ai_current" };
const STRAIN: TypeProps = { name: "Strain", type: "ai_strain" };

const elements = (node: Root | Element): Element[] =>
  node.children.flatMap((child) =>
    child.type === "element" ? [child, ...elements(child)] : [],
  );

const text = (node: Element): string =>
  node.children
    .map((child) =>
      child.type === "text" ? child.value : child.type === "element" ? text(child) : "",
    )
    .join("");

const withClass = (node: Root | Element, name: string): Element[] =>
  elements(node).filter(({ properties: { className } }) =>
    Array.isArray(className) ? className.includes(name) : false,
  );

const withRole = (node: Element, role: string): Element[] =>
  elements(node).filter((child) => child.properties.role === role);

/** The tabs of the groups. */
const groups = (root: Root): Element[] =>
  withClass(root, "docs-tabs__header").flatMap((header) => withRole(header, "tab"));

/** The chips of each set of types. */
const chips = (root: Root): Element[][] =>
  withClass(root, "channel-types__options").map((options) => withRole(options, "tab"));

/** The type that the icon on a tab holds. */
const icon = (tab: Element): unknown =>
  elements(tab).find((node) => node.tagName === "svg")?.properties.dataType;

let container: AstroContainer;

beforeAll(async () => {
  container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
});

// Renders a Channel.Type as a page does, with an icon that holds its type.
const renderType = (props: TypeProps): Promise<string> =>
  container.renderToString(Channel.Type, {
    props: { ...props },
    slots: {
      icon: `<svg data-type="${props.type}"></svg>`,
      default: "<p>Reads a value.</p>",
    },
  });

const renderTypes = async (slot: string): Promise<Root> => {
  const html = await container.renderToString(Channel.Types, {
    slots: { default: slot },
  });
  return fromHtml(html, { fragment: true });
};

const render = async (...types: TypeProps[]): Promise<Root> => {
  const panes = await Promise.all(types.map(renderType));
  return await renderTypes(panes.join("\n"));
};

describe("Channel.Types", () => {
  describe("without groups", () => {
    it("should show a chip for each type in order", async () => {
      const root = await render(VOLTAGE, CURRENT);
      expect(groups(root)).toEqual([]);
      expect(chips(root).map((set) => set.map(text))).toEqual([["Voltage", "Current"]]);
    });

    it("should select the first type", async () => {
      const [set] = chips(await render(VOLTAGE, CURRENT));
      expect(set.map((chip) => chip.properties.ariaSelected)).toEqual([
        "true",
        "false",
      ]);
    });

    it("should show each type in its own pane", async () => {
      const [options] = withClass(
        await render(VOLTAGE, CURRENT),
        "channel-types__options",
      );
      const panes = withRole(options, "tabpanel").map((pane) =>
        withClass(pane, "channel-type").map((type) => type.properties.id),
      );
      expect(panes).toEqual([["ai_voltage"], ["ai_current"]]);
    });

    it("should show the icon of each type on its chip", async () => {
      const [set] = chips(await render(VOLTAGE, CURRENT));
      expect(set.map(icon)).toEqual(["ai_voltage", "ai_current"]);
    });
  });

  describe("with groups", () => {
    const TYPES: TypeProps[] = [
      { ...VOLTAGE, group: "Electrical" },
      { ...STRAIN, group: "Mechanical" },
      { ...CURRENT, group: "Electrical" },
    ];

    it("should show the groups in the order of their first type", async () => {
      const root = await render(...TYPES);
      expect(groups(root).map(text)).toEqual(["Electrical", "Mechanical"]);
    });

    it("should show the types of each group as its chips", async () => {
      const root = await render(...TYPES);
      expect(chips(root).map((set) => set.map(text))).toEqual([
        ["Voltage", "Current"],
        ["Strain"],
      ]);
    });

    it("should show the icon of the first type of a group on the group", async () => {
      const root = await render(...TYPES);
      expect(groups(root).map(icon)).toEqual(["ai_voltage", "ai_strain"]);
    });
  });

  describe("names", () => {
    it("should show a name with an ampersand or a quote as written", async () => {
      const root = await render({ name: 'Voltage & "RMS"', type: "ai_voltage_rms" });
      expect(chips(root).map((set) => set.map(text))).toEqual([['Voltage & "RMS"']]);
    });

    it("should show a group with an ampersand as written", async () => {
      const root = await render(
        { ...VOLTAGE, group: "Pressure & Flow" },
        { ...STRAIN, group: "Mechanical" },
      );
      expect(groups(root).map(text)).toEqual(["Pressure & Flow", "Mechanical"]);
    });
  });

  describe("errors", () => {
    it("should throw when it holds no Channel.Type", async () => {
      await expect(renderTypes("<p>Reads a value.</p>")).rejects.toThrow(
        "Channel.Types needs at least one Channel.Type",
      );
    });

    it("should throw when it holds content outside a Channel.Type", async () => {
      const slot = `${await renderType(VOLTAGE)}<p>Reads a value.</p>`;
      await expect(renderTypes(slot)).rejects.toThrow(
        "Channel.Types holds content outside a Channel.Type",
      );
    });

    it("should throw when a Channel.Type has no type", async () => {
      const tag = '<section class="channel-type" data-name="Voltage">';
      await expect(render({ name: "Voltage" })).rejects.toThrow(
        `Channel.Type is missing id: ${tag}`,
      );
    });

    it("should throw when a Channel.Type has no name", async () => {
      const tag = '<section class="channel-type" id="ai_voltage">';
      await expect(render({ type: "ai_voltage" })).rejects.toThrow(
        `Channel.Type is missing data-name: ${tag}`,
      );
    });

    it("should throw when only some types have a group", async () => {
      await expect(render({ ...VOLTAGE, group: "Electrical" }, STRAIN)).rejects.toThrow(
        "Channel.Types needs a group on every Channel.Type or on none",
      );
    });

    it("should throw when every type has the same group", async () => {
      await expect(
        render(
          { ...VOLTAGE, group: "Electrical" },
          { ...CURRENT, group: "Electrical" },
        ),
      ).rejects.toThrow("Channel.Types needs at least two groups or none");
    });
  });
});
