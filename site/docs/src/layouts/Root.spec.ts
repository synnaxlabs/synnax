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
import { experimental_AstroContainer } from "astro/container";
import { type Element, type Root } from "hast";
import { fromHtml } from "hast-util-from-html";
import { describe, expect, it, vi } from "vitest";

import { resolve, ROOT, type Version } from "@/util/version";

const elements = (node: Root | Element): Element[] =>
  node.children.flatMap((child) =>
    child.type === "element" ? [child, ...elements(child)] : [],
  );

const text = (node: Element): string =>
  node.children
    .map((child) =>
      child.type === "text" ? child.value : child.type === "element" ? text(child) : "",
    )
    .join("")
    .trim();

const TRAINS = ["0.61", "0.60"];
const PATH = "/reference/control/";

// The layout reads its version from the build, so each render loads it anew.
const render = async (version: Version | null): Promise<Element[]> => {
  vi.resetModules();
  vi.doMock("@/version", () => ({ VERSION: version }));
  const { default: Layout } = await import("@/layouts/Root.astro");
  const container = await experimental_AstroContainer.create({
    astroConfig: { site: ROOT },
  });
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const html = await container.renderToString(Layout, {
    props: { frontmatter: { title: "Control" } },
    request: new Request(`https://docs.synnaxlabs.com${PATH}`),
  });
  return elements(fromHtml(html));
};

const robots = (nodes: Element[]): unknown[] =>
  nodes
    .filter(
      ({ tagName, properties }) => tagName === "meta" && properties.name === "robots",
    )
    .map(({ properties: { content } }) => content);

const banners = (nodes: Element[]): Element[] =>
  nodes.filter(({ tagName }) => tagName === "aside");

const hrefs = (nodes: Element[], name: string): unknown[] =>
  nodes
    .filter(({ tagName }) => tagName === "a")
    .filter((node) => text(node) === name)
    .map(({ properties: { href } }) => href);

const feeds = (nodes: Element[]): unknown[] =>
  nodes
    .filter(({ properties: { type } }) => type === "application/rss+xml")
    .map(({ properties: { href } }) => href);

describe("Root", () => {
  describe.each([
    ["an unversioned build", null],
    ["the newest train", resolve("0.61", TRAINS, "0.61.0")],
  ])("on %s", (_, version) => {
    it("should let search engines index the pages", async () => {
      expect(robots(await render(version)).join()).not.toContain("noindex");
    });

    it("should show no banner", async () => {
      expect(banners(await render(version))).toEqual([]);
    });

    it("should serve the pages no version owns itself", async () => {
      const nodes = await render(version);
      // The footer and the analytics notice.
      expect(hrefs(nodes, "Privacy policy")).toEqual([
        "/privacy-policy",
        "/privacy-policy",
      ]);
      expect(feeds(nodes)).toEqual(["/blog/rss.xml"]);
    });
  });

  describe.each([
    [
      "an earlier train",
      resolve("0.60", TRAINS, "0.60.1"),
      "These docs describe Synnax 0.60.",
    ],
    ["next", resolve("next", TRAINS, null), "These docs describe unreleased work."],
  ])("on %s", (_, version, notice) => {
    it("should keep search engines out", async () => {
      expect(robots(await render(version)).join()).toContain("noindex");
    });

    it("should link the banner to the same page on the newest train", async () => {
      const [banner] = banners(await render(version));
      expect(text(banner)).toContain(notice);
      expect(hrefs(elements(banner), "Read the docs for the latest release")).toEqual([
        `${ROOT}${PATH}`,
      ]);
    });

    it("should send the pages no version owns to the root", async () => {
      const nodes = await render(version);
      expect(hrefs(nodes, "Privacy policy")).toEqual([
        `${ROOT}/privacy-policy`,
        `${ROOT}/privacy-policy`,
      ]);
      expect(feeds(nodes)).toEqual([`${ROOT}/blog/rss.xml`]);
    });
  });
});
