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
import { describe, expect, it } from "vitest";

import { Releases as Components } from "@/components/releases";
import { type Release, Releases } from "@/util/releases";

const RELEASES: Release[] = [
  {
    tag_name: "core/v0.61.0",
    prerelease: false,
    draft: false,
    published_at: "2026-11-12T05:57:37Z",
  },
  {
    tag_name: "core/v0.62.0-rc.1",
    prerelease: true,
    draft: false,
    published_at: "2026-11-20T05:57:37Z",
  },
];

const CORE_TAG = "https://github.com/synnaxlabs/synnax/tree/core/v";

const render = async (props: { version: string; date?: string }): Promise<string> => {
  const container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const releases = new Releases({
    fetch: async () => new Response(JSON.stringify(RELEASES)),
  });
  return await container.renderToString(Components.Release, {
    locals: { releases },
    props: { title: "Notes", ...props },
  });
};

describe("Release", () => {
  it("should date the notes from the stable Core release", async () => {
    const html = await render({ version: "0.61.0" });
    expect(html).toContain("Nov 11, 2026");
    expect(html).toContain(`${CORE_TAG}0.61.0`);
  });

  it("should show a version with no stable Core release as unreleased", async () => {
    const html = await render({ version: "0.62.0" });
    expect(html).toContain("Unreleased");
    expect(html).not.toContain(`${CORE_TAG}0.62.0`);
  });

  it("should keep the date that the notes pass", async () => {
    const html = await render({ version: "0.58.0", date: "Sep 1, 2026" });
    expect(html).toContain("Sep 1, 2026");
    expect(html).toContain("synnax-v0.58.0");
  });
});
