// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { hash } from "@synnaxlabs/site-common/csp";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { JSDOM } from "jsdom";
import { describe, expect, it } from "vitest";

import { STORAGE_KEY } from "@/components/nav/collapse";
import Mark from "@/components/platform/Mark.astro";
import { COLLAPSE, HASHES } from "@/inline";

describe("inline", () => {
  describe("HASHES", () => {
    it("should let the rendered platform mark run", async () => {
      const container = await AstroContainer.create();
      const html = await container.renderToString(Mark);
      const script = new JSDOM(html).window.document.querySelector("script");
      expect(HASHES).toContain(hash(script?.textContent ?? ""));
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
