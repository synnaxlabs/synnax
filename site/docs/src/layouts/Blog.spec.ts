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

import Blog from "@/layouts/Blog.astro";
import { type Release, Releases } from "@/util/releases";

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

const RELEASES: Release[] = [
  { tag_name: "core/v0.60.0", prerelease: false, draft: false, published_at: null },
  { tag_name: "desktop/v1.0.0", prerelease: false, draft: false, published_at: null },
];

const listing = async (): Promise<Response> => new Response(JSON.stringify(RELEASES));

const renderPost = async (
  path: string,
  fetch: typeof globalThis.fetch = listing,
): Promise<Element[]> => {
  const container = await experimental_AstroContainer.create({
    astroConfig: { site: "https://docs.synnaxlabs.com" },
  });
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const releases = new Releases({ fetch });
  const html = await container.renderToString(Blog, {
    locals: { releases },
    request: new Request(new URL(path, "https://docs.synnaxlabs.com")),
    props: {
      headings: [],
      frontmatter: { title: "A post", description: "A post.", published: "2024-10-15" },
    },
  });
  return elements(fromHtml(html, { fragment: true }));
};

const segmentHrefs = async (path: string): Promise<unknown[]> =>
  (await renderPost(path))
    .filter(({ properties: { className } }) =>
      Array.isArray(className) ? className.includes("lyra-breadcrumb__segment") : false,
    )
    .map(({ properties: { href } }) => href);

describe("Blog", () => {
  it.each(["/blog/a-post", "/blog/a-post/"])(
    "should link one breadcrumb segment per path segment at %s",
    async (path) => {
      expect(await segmentHrefs(path)).toEqual(["/blog", "/blog/a-post"]);
    },
  );

  it("should end with a Desktop download that reports the blog as its location", async () => {
    const tracked = (await renderPost("/blog/a-post")).filter(
      ({ properties }) => properties.dataEvent === "desktop_download_clicked",
    );
    expect(tracked).toHaveLength(2);
    expect(tracked.map(({ properties }) => properties.dataLocation)).toEqual([
      "blog",
      "blog",
    ]);
  });

  it("should give the outline no heading for the download", async () => {
    const headings = (await renderPost("/blog/a-post")).filter(({ tagName }) =>
      ["h1", "h2", "h3"].includes(tagName),
    );
    expect(headings.map(text)).not.toContain("Try Synnax Desktop");
  });

  it("should render the post without the download when GitHub is down", async () => {
    const post = await renderPost("/blog/a-post", async () => {
      throw new Error("GitHub is down");
    });
    expect(post.some(({ tagName }) => tagName === "article")).toBe(true);
    expect(post.filter(({ properties }) => properties.dataEvent != null)).toEqual([]);
  });
});
