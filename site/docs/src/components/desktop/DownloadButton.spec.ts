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

const render = async (): Promise<Element[]> => {
  const container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const releases = new Releases({
    fetch: async () => new Response(JSON.stringify(RELEASES)),
  });
  const html = await container.renderToString(DownloadButton, { locals: { releases } });
  return elements(fromHtml(html, { fragment: true })).filter(
    (node) => node.properties.dataAgent != null,
  );
};

describe("Desktop download button", () => {
  it("should offer the installers of the latest stable Desktop release", async () => {
    const buttons = await render();
    expect(buttons.map(text)).toEqual([
      "Download v1.0.0 for macOS",
      "Download v1.0.0 for Windows",
    ]);
    expect(buttons.map((button) => button.properties.href)).toEqual([
      `${DOWNLOAD}/v1.0.0/Synnax-Desktop_1.0.0_aarch64.dmg`,
      `${DOWNLOAD}/v1.0.0/Synnax-Desktop_1.0.0_x64-setup.exe`,
    ]);
  });

  it("should match each installer to the user agents of its platform", async () => {
    const buttons = await render();
    expect(buttons.map((button) => button.properties.dataAgent)).toEqual([
      "Macintosh",
      "Windows NT",
    ]);
  });
});
