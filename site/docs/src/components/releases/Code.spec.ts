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
import type { ComponentProps } from "astro/types";
import { describe, expect, it } from "vitest";

import { Releases as Components } from "@/components/releases";
import { type Release, Releases } from "@/util/releases";

const release = (tag_name: string): Release => ({
  tag_name,
  prerelease: false,
  draft: false,
  published_at: null,
});

const RELEASES: Release[] = [
  release("core/v0.62.1"),
  release("core/v0.61.4"),
  release("driver/v0.62.0"),
  release("driver/v0.61.5"),
];

const render = async (
  props: Omit<ComponentProps<typeof Components.Code>, "lang">,
): Promise<string> => {
  const container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  const releases = new Releases({
    fetch: async () => new Response(JSON.stringify(RELEASES)),
  });
  return await container.renderToString(Components.Code, {
    locals: { releases },
    props: { lang: "text", ...props },
  });
};

describe("Code", () => {
  it("should build the code from the documented and previous releases", async () => {
    const text = await render({
      code: ({ current, previous }) =>
        `${current.version} ${current.train} ${previous.version} ${previous.train}`,
    });
    expect(text).toContain("0.62.1 0.62 0.61.4 0.61");
  });

  it("should use the releases of the product", async () => {
    const text = await render({
      product: "driver",
      code: ({ current }) => current.version,
    });
    expect(text).toContain("0.62.0");
  });
});
