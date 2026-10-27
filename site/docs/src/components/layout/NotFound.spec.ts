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
import { describe, expect, it } from "vitest";

import NotFound from "@/components/layout/NotFound.astro";

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

const render = async (path: string): Promise<(name: string) => Element[]> => {
  const container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const html = await container.renderToString(NotFound, {
    request: new Request(new URL(path, "https://docs.synnaxlabs.com")),
  });
  const all = elements(fromHtml(html, { fragment: true }));
  return (name) =>
    all.filter(({ properties: { className } }) =>
      Array.isArray(className) ? className.includes(name) : false,
    );
};

describe("NotFound", () => {
  it("should show the path that has no page", async () => {
    const find = await render("/reference/channles");
    const [lede] = find("not-found__lede");
    const code = elements(lede).find(({ tagName }) => tagName === "code");
    expect(code != null && text(code)).toBe("/reference/channles");
  });

  it("should link the likely page by its name in the sidebar", async () => {
    const find = await render("/reference/concepts/channles");
    const [page] = find("not-found__page");
    expect(page.properties.href).toBe("/reference/concepts/channels");
    expect(text(find("not-found__name")[0])).toBe("Channels");
  });

  it("should show no similar pages for a path unlike any page", async () => {
    const find = await render("/asdf");
    expect(find("not-found__similar")).toEqual([]);
  });
});
