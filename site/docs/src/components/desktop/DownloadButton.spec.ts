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
import { type Element, type Properties, type Root } from "hast";
import { fromHtml } from "hast-util-from-html";
import { describe, expect, it } from "vitest";

import DownloadButton from "@/components/desktop/DownloadButton.astro";
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
  { tag_name: "desktop/v1.1.0-rc.1", prerelease: true, draft: false },
  { tag_name: "desktop/v1.0.0", prerelease: false, draft: false },
  { tag_name: "desktop/v1.0.0-rc.2", prerelease: true, draft: false },
  { tag_name: "console/v0.60.0", prerelease: false, draft: false },
];

const DOWNLOAD = "https://github.com/synnaxlabs/synnax/releases/download/desktop";

interface Installer {
  agent: Properties[string];
  href: Properties[string];
  label: string;
  caption: string;
}

const renderTree = async (): Promise<Root> => {
  const container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const releases = new Releases({
    fetch: async () => new Response(JSON.stringify(RELEASES)),
  });
  const html = await container.renderToString(DownloadButton, { locals: { releases } });
  return fromHtml(html, { fragment: true });
};

const render = async (): Promise<Installer[]> =>
  elements(await renderTree())
    .filter((node) => node.properties.dataAgent != null)
    .map((platform) => {
      const [link, caption] = platform.children.filter(
        (child): child is Element => child.type === "element",
      );
      return {
        agent: platform.properties.dataAgent,
        href: link.properties.href,
        label: text(link),
        caption: text(caption),
      };
    });

describe("Desktop download button", () => {
  it("should offer the installers of the latest stable Desktop release", async () => {
    const installers = await render();
    expect(installers.map(({ label }) => label)).toEqual([
      "Download Synnax Desktop",
      "Download Synnax Desktop",
    ]);
    expect(installers.map(({ href }) => href)).toEqual([
      `${DOWNLOAD}/v1.0.0/Synnax-Desktop_1.0.0_aarch64.dmg`,
      `${DOWNLOAD}/v1.0.0/Synnax-Desktop_1.0.0_x64-setup.exe`,
    ]);
  });

  it("should name the version and platform under each installer", async () => {
    const installers = await render();
    expect(installers.map(({ caption }) => caption)).toEqual([
      "v1.0.0 for macOS",
      "v1.0.0 for Windows",
    ]);
  });

  it("should match each installer to the user agents of its platform", async () => {
    const installers = await render();
    expect(installers.map(({ agent }) => agent)).toEqual(["Macintosh", "Windows NT"]);
  });

  it("should pick the platform again after each client-side navigation", async () => {
    const scripts = elements(await renderTree()).filter(
      (node) => node.tagName === "script",
    );
    expect(scripts.map(({ properties }) => properties.dataAstroRerun)).toEqual([""]);
  });
});
