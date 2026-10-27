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

import Header from "@/components/layout/Header.astro";
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
  { tag_name: "core/v0.60.0", prerelease: false, draft: false },
];

const render = async (): Promise<Element[]> => {
  const container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const releases = new Releases({
    fetch: async () => new Response(JSON.stringify(RELEASES)),
  });
  const html = await container.renderToString(Header, {
    locals: { releases },
    request: new Request("https://docs.synnaxlabs.com/reference/"),
  });
  return elements(fromHtml(html, { fragment: true }));
};

describe("Header", () => {
  it("should link Log in to the sign-in page of the portal", async () => {
    const logins = (await render()).filter(({ properties: { className } }) =>
      Array.isArray(className) ? className.includes("login-button") : false,
    );
    expect(logins.map(text)).toEqual(["Log in"]);
    expect(logins.map(({ properties: { href } }) => href)).toEqual([
      "https://portal.synnaxlabs.com/login",
    ]);
  });
});
