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
import { describe, expect, it } from "vitest";

import Reference from "@/layouts/Reference.astro";

const elements = (node: Root | Element): Element[] =>
  node.children.flatMap((child) =>
    child.type === "element" ? [child, ...elements(child)] : [],
  );

const ROUTE = "/reference/channels/create";

const breadcrumbs = async (breadcrumbHidden: boolean): Promise<unknown[]> => {
  const container = await experimental_AstroContainer.create({
    astroConfig: { site: "https://docs.synnaxlabs.com" },
  });
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const html = await container.renderToString(Reference, {
    request: new Request(new URL(ROUTE, "https://docs.synnaxlabs.com")),
    props: {
      headings: [],
      frontmatter: {
        title: "Create channels",
        description: "Create channels.",
        url: ROUTE,
        breadcrumbHidden,
        onThisPageHidden: false,
      },
    },
  });
  return elements(fromHtml(html, { fragment: true }))
    .filter(({ properties: { className } }) =>
      Array.isArray(className) ? className.includes("lyra-breadcrumb__segment") : false,
    )
    .map(({ properties: { href } }) => href);
};

describe("Reference", () => {
  it("should show a breadcrumb", async () => {
    expect(await breadcrumbs(false)).not.toEqual([]);
  });

  it("should hide the breadcrumb when the page sets breadcrumbHidden", async () => {
    expect(await breadcrumbs(true)).toEqual([]);
  });
});
