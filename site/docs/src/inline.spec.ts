// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { Icon } from "@synnaxlabs/lyra/icon";
import { hash, LOADER } from "@synnaxlabs/site-common/csp";
import { type experimental_AstroContainer as AstroContainer } from "astro/container";
import { JSDOM } from "jsdom";
import { describe, expect, it } from "vitest";

import Buttons from "@/components/download/Buttons.astro";
import { STORAGE_KEY } from "@/components/nav/collapse";
import Mark from "@/components/platform/Mark.astro";
import { COLLAPSE, HASHES, MARK } from "@/inline";
import Reference from "@/layouts/Reference.astro";
import { createContainer } from "@/testutil/container";

/** inlineHashes renders a component and hashes each of its inline scripts. */
const inlineHashes = async (
  render: (container: AstroContainer) => Promise<string>,
): Promise<string[]> => {
  const container = await createContainer("https://docs.synnaxlabs.com");
  const { document } = new JSDOM(await render(container)).window;
  return [...document.querySelectorAll("script:not([src]):not([type])")].map((s) =>
    hash(s.textContent),
  );
};

describe("inline", () => {
  describe("HASHES", () => {
    it("should let the rendered platform mark run", async () => {
      const hashes = await inlineHashes(async (c) => await c.renderToString(Mark));
      expect(hashes).toHaveLength(1);
      for (const h of hashes) expect(HASHES).toContain(h);
    });

    it("should let the rendered download picker run", async () => {
      const hashes = await inlineHashes(
        async (c) =>
          await c.renderToString(Buttons, {
            props: {
              product: "console",
              version: "0.1.0",
              label: "Download",
              platforms: [
                {
                  name: "macOS",
                  asset: "a.dmg",
                  agent: /Macintosh/,
                  icon: Icon.Logo.Apple,
                },
              ],
            },
          }),
      );
      expect(hashes).toHaveLength(1);
      for (const h of hashes) expect(HASHES).toContain(h);
    });

    it("should match the scripts a reference page renders", async () => {
      const hashes = await inlineHashes(
        async (c) =>
          await c.renderToString(Reference, {
            props: {
              frontmatter: { title: "Channels", url: "/reference/channels" },
              headings: [],
            },
            request: new Request("https://docs.synnaxlabs.com/reference/channels"),
          }),
      );
      for (const script of [COLLAPSE, MARK, LOADER])
        expect(hashes).toContain(hash(script));
    });
  });

  describe("COLLAPSE", () => {
    const run = (stored: string | null): boolean => {
      const { window } = new JSDOM("", {
        url: "https://docs.synnaxlabs.com/",
        runScripts: "outside-only",
      });
      if (stored != null) window.localStorage.setItem(STORAGE_KEY, stored);
      window.eval(COLLAPSE);
      return window.document.documentElement.dataset.pageNavCollapsed != null;
    };

    it("should collapse the sidebar when the reader collapsed it before", () => {
      expect(run("true")).toBe(true);
    });

    it("should leave the sidebar open otherwise", () => {
      expect(run("false")).toBe(false);
      expect(run(null)).toBe(false);
    });
  });
});
