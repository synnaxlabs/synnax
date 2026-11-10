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

import Header from "@/components/layout/Header.astro";
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

const render = async (version: Version | null = null): Promise<Element[]> => {
  const container = await experimental_AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const html = await container.renderToString(Header, {
    props: { version },
    request: new Request("https://docs.synnaxlabs.com/reference/"),
  });
  return elements(fromHtml(html, { fragment: true }));
};

const hrefs = (nodes: Element[], name: string): unknown[] =>
  nodes
    .filter(({ tagName }) => tagName === "a")
    .filter((node) => text(node) === name)
    .map(({ properties: { href } }) => href);

const classed = (nodes: Element[], name: string): Element[] =>
  nodes.filter(({ properties: { className } }) =>
    Array.isArray(className) ? className.includes(name) : false,
  );

describe("Header", () => {
  it("should show no version menu when the build has no version", async () => {
    expect(classed(await render(), "version-menu")).toEqual([]);
  });

  it("should link GitHub from the nav and the mobile menu", async () => {
    expect(hrefs(await render(), "GitHub")).toEqual([
      "https://github.com/synnaxlabs/synnax",
      "https://github.com/synnaxlabs/synnax",
    ]);
  });

  it("should link each version at the current path", async () => {
    const nodes = await render(resolve("0.59", ["0.60", "0.59"], "0.59.2"));
    const links = classed(nodes, "version-menu__list").flatMap(elements);
    expect(
      links
        .filter(({ tagName }) => tagName === "a")
        .map(({ properties: { href } }) => href),
    ).toEqual([
      "https://next.docs.synnaxlabs.com/reference/",
      `${ROOT}/reference/`,
      "https://v0-59.docs.synnaxlabs.com/reference/",
    ]);
  });

  it("should name the root by its train only on the newest train", async () => {
    const labels = async (version: Version): Promise<string[]> =>
      classed(await render(version), "version-menu__list")
        .flatMap(elements)
        .filter(({ tagName }) => tagName === "a")
        .map((link) => text(elements(link)[0]));
    expect(await labels(resolve("0.60", ["0.60", "0.59"], "0.60.1"))).toEqual([
      "Next",
      "v0.60",
      "v0.59",
    ]);
    expect(await labels(resolve("0.59", ["0.60", "0.59"], "0.59.2"))).toEqual([
      "Next",
      "Latest",
      "v0.59",
    ]);
  });

  it("should name the newest release of the train on the trigger", async () => {
    const trigger = async (version: Version): Promise<string> =>
      text(classed(await render(version), "version-menu__trigger")[0]);
    expect(await trigger(resolve("0.60", ["0.61", "0.60"], "0.60.1"))).toBe("v0.60.1");
    expect(await trigger(resolve("next", ["0.61", "0.60"], null))).toBe("Next");
  });

  it("should send the pages no version owns to the root", async () => {
    expect(hrefs(await render(resolve("next", ["0.60"], null)), "Blog")).toEqual([
      `${ROOT}/blog/`,
      `${ROOT}/blog/`,
    ]);
    expect(hrefs(await render(resolve("0.60", ["0.60"], "0.60.1")), "Blog")).toEqual([
      "/blog/",
      "/blog/",
    ]);
  });

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
