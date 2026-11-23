// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

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
  props: ComponentProps<typeof Components.Version>,
): Promise<string> => {
  const container = await AstroContainer.create();
  const releases = new Releases({
    fetch: async () => new Response(JSON.stringify(RELEASES)),
  });
  return await container.renderToString(Components.Version, {
    locals: { releases },
    props,
  });
};

describe("Version", () => {
  it("should show the documented Core release", async () => {
    expect((await render({})).trim()).toEqual("0.62.1");
  });

  it("should show only the train", async () => {
    expect((await render({ train: true })).trim()).toEqual("0.62");
  });

  it("should show the release before the documented one", async () => {
    expect((await render({ previous: true })).trim()).toEqual("0.61.4");
  });

  it("should show the version as inline code", async () => {
    expect(await render({ code: true })).toContain("<code data-short>0.62.1</code>");
  });

  it("should show the release of another product", async () => {
    const html = await render({ product: "driver", previous: true });
    expect(html.trim()).toEqual("0.61.5");
  });
});
