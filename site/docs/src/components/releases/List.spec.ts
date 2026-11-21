// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import type { Element, Root } from "hast";
import { fromHtml } from "hast-util-from-html";
import { describe, expect, it } from "vitest";

import { Releases as Components } from "@/components/releases";
import { createContainer } from "@/testutil/container";

interface Entry {
  slug: string;
  text: string;
  date?: string;
}

const render = async (releases: Entry[]): Promise<string> => {
  const container = await createContainer();
  return await container.renderToString(Components.List, { props: { releases } });
};

const elements = (node: Root | Element): Element[] =>
  node.children.flatMap((child) =>
    child.type === "element" ? [child, ...elements(child)] : [],
  );

const named = (node: Root | Element, tag: string): Element[] =>
  elements(node).filter(({ tagName }) => tagName === tag);

const text = (node: Element): string =>
  node.children
    .map((child) =>
      child.type === "text" ? child.value : child.type === "element" ? text(child) : "",
    )
    .join("")
    .trim();

const groups = (html: string): Element[] =>
  named(fromHtml(html, { fragment: true }), "details");

// Each year with the releases under it, and an asterisk on the open year.
const years = (html: string): string[] =>
  groups(html).map((group) => {
    const year = text(named(group, "summary")[0]);
    const releases = named(group, "a").map(text);
    return `${year}${group.properties.open === true ? "*" : ""}: ${releases.join(" ")}`;
  });

describe("List", () => {
  it("should group the releases under the year they shipped", async () => {
    const html = await render([
      { slug: "062", text: "0.62", date: "Nov 19, 2026" },
      { slug: "050", text: "0.50", date: "Jan 26, 2026" },
      { slug: "049", text: "0.49", date: "Dec 9, 2025" },
    ]);
    expect(years(html)).toEqual(["2026*: 0.62 0.50", "2025: 0.49"]);
  });

  it("should let one year be open at a time", async () => {
    const html = await render([
      { slug: "062", text: "0.62", date: "Nov 19, 2026" },
      { slug: "049", text: "0.49", date: "Dec 9, 2025" },
      { slug: "037", text: "0.37", date: "Dec 31, 2024" },
    ]);
    const names = groups(html).map(({ properties }) => properties.name);
    expect(names).toEqual(["release-years", "release-years", "release-years"]);
  });

  it("should list a release that has not shipped as unreleased", async () => {
    const html = await render([
      { slug: "063", text: "0.63" },
      { slug: "062", text: "0.62", date: "Nov 19, 2026" },
    ]);
    expect(years(html)).toEqual(["Unreleased*: 0.63", "2026: 0.62"]);
  });

  it("should throw when a release date has no year", async () => {
    await expect(
      render([{ slug: "062", text: "0.62", date: "Nov 19" }]),
    ).rejects.toThrow("The release date Nov 19 has no year");
  });
});
