// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { describe, expect, it } from "vitest";

import Layout from "@/layouts/Layout.astro";
import { renderAstro } from "@/testutil";

const render = (url: string, title?: string): Promise<Document> =>
  renderAstro(Layout, {
    props: { title, description: "Telemetry for hardware teams." },
    request: new Request(url),
  });

const meta = (doc: Document, key: string): string | null =>
  doc
    .querySelector(`meta[property="${key}"], meta[name="${key}"]`)
    ?.getAttribute("content") ?? null;

const canonical = (doc: Document): string | null =>
  doc.querySelector('link[rel="canonical"]')?.getAttribute("href") ?? null;

describe("Layout", () => {
  describe("title", () => {
    it("should use the home title when the page sets none", async () => {
      const doc = await render("https://www.synnaxlabs.com/");
      expect(doc.title).toBe(
        "Synnax | Hardware data acquisition, control, and analysis",
      );
    });

    it("should suffix the page title with the site name", async () => {
      const doc = await render("https://www.synnaxlabs.com/company", "Company");
      expect(doc.title).toBe("Company | Synnax");
      expect(meta(doc, "og:title")).toBe("Company | Synnax");
      expect(meta(doc, "twitter:title")).toBe("Company | Synnax");
    });
  });

  describe("canonical URL", () => {
    it("should keep the slash of the home page", async () => {
      const doc = await render("https://www.synnaxlabs.com/");
      expect(canonical(doc)).toBe("https://www.synnaxlabs.com/");
    });

    it("should drop a trailing slash", async () => {
      const doc = await render("https://www.synnaxlabs.com/company/");
      expect(canonical(doc)).toBe("https://www.synnaxlabs.com/company");
    });

    it("should drop the query string", async () => {
      const doc = await render("https://www.synnaxlabs.com/company?ref=linkedin");
      expect(canonical(doc)).toBe("https://www.synnaxlabs.com/company");
    });

    it("should resolve against the site, not the request host", async () => {
      const doc = await render("http://localhost:4321/privacy");
      expect(canonical(doc)).toBe("https://www.synnaxlabs.com/privacy");
      expect(meta(doc, "og:url")).toBe("https://www.synnaxlabs.com/privacy");
    });
  });

  it("should describe the page to search engines and social cards", async () => {
    const doc = await render("https://www.synnaxlabs.com/");
    expect(meta(doc, "description")).toBe("Telemetry for hardware teams.");
    expect(meta(doc, "og:description")).toBe("Telemetry for hardware teams.");
    expect(meta(doc, "twitter:card")).toBe("summary_large_image");
    expect(meta(doc, "og:image")).toBe(
      "https://synnax.nyc3.cdn.digitaloceanspaces.com/docs/og-default.png",
    );
  });
});
