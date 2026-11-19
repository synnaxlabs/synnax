// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import reactRenderer from "@astrojs/react/server.js";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { JSDOM } from "jsdom";
import { describe, expect, it } from "vitest";

import Analytics from "./Analytics.astro";
import { hash, HASHES } from "./csp";

describe("csp", () => {
  describe("hash", () => {
    it("should return the base64 SHA-256 source of the text", () => {
      expect(hash("")).toBe("sha256-47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=");
    });
  });

  describe("HASHES", () => {
    it("should let every inline script of the analytics run", async () => {
      const container = await AstroContainer.create();
      container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
      const html = await container.renderToString(Analytics, {
        props: { privacy: "/privacy" },
      });
      const scripts = [
        ...new JSDOM(html).window.document.querySelectorAll("script:not([src])"),
      ];
      expect(scripts.length).toBeGreaterThan(0);
      for (const script of scripts) expect(HASHES).toContain(hash(script.textContent));
    });
  });
});
