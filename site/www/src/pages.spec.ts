// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import Company from "@/pages/company.astro";
import Index from "@/pages/index.astro";
import Privacy from "@/pages/privacy.astro";
import { GET } from "@/pages/sitemap.xml";
import Sponsorships from "@/pages/sponsorships.astro";
import { renderAstro, stubImageSize } from "@/testutil";

import config from "../astro.config";

const CDN = "https://synnax.nyc3.cdn.digitaloceanspaces.com/";

const render = (
  page: Parameters<typeof renderAstro>[0],
  path: string,
): Promise<Document> =>
  renderAstro(page, { request: new Request(new URL(path, config.site)) });

const all = <E extends Element>(parent: ParentNode, selector: string): E[] => [
  ...parent.querySelectorAll<E>(selector),
];

const cdnImages = (doc: Document): HTMLImageElement[] =>
  all<HTMLImageElement>(doc, "picture img").filter((img) =>
    decodeURIComponent(img.src).includes(CDN),
  );

describe("pages", () => {
  beforeEach(() => {
    stubImageSize(1600, 900);
  });

  describe("home", () => {
    let doc: Document;

    beforeAll(async () => {
      stubImageSize(1600, 900);
      doc = await render(Index, "/");
    });

    describe("navigation", () => {
      it("should link only to sections that exist on the page", () => {
        const ids = new Set(all(doc, "[id]").map((el) => el.id));
        const targets = all(doc, 'a[href^="/#"], a[href^="#"]').map(
          (a) => a.getAttribute("href")!.split("#")[1],
        );
        expect(targets.length).toBeGreaterThan(0);
        for (const target of new Set(targets)) expect(ids).toContain(target);
      });

      it("should number each product card as its section is numbered", () => {
        const number = (el: Element): string => el.textContent.split(" -> ")[0];
        const cards = all<HTMLAnchorElement>(doc, ".product-card");
        expect(cards).toHaveLength(6);
        for (const card of cards) {
          const id = card.getAttribute("href")!.split("#")[1];
          const label = doc.querySelector(`[id="${id}"] .section-label`)!;
          expect(number(card.querySelector(".product-card__title")!)).toBe(
            number(label),
          );
        }
      });
    });

    describe("JavaScript", () => {
      it("should hydrate every island only when it scrolls into view", () => {
        const islands = all(doc, "astro-island");
        expect(islands.length).toBeGreaterThan(0);
        expect(new Set(islands.map((i) => i.getAttribute("client")))).toEqual(
          new Set(["visible"]),
        );
      });

      it("should render the product menu on the server", () => {
        const nav = doc.querySelector("header .nav-links")!;
        expect(nav.closest("astro-island")).toBeNull();
        expect(all(nav, ".product-dropdown .product-card")).toHaveLength(6);
      });
    });

    describe("integrations", () => {
      it("should give each workflow tab a panel", () => {
        const steps = all(doc, ".workflow-tab").map((t) => t.getAttribute("data-step"));
        const panels = all(doc, ".workflow-panel").map((p) =>
          p.getAttribute("data-panel"),
        );
        expect(steps.length).toBeGreaterThan(0);
        expect(steps).toEqual(panels);
      });

      it("should name the vendor of each integration cell", () => {
        const cells = all(doc, ".integration-cell");
        expect(cells.length).toBeGreaterThan(0);
        for (const cell of cells) expect(cell.getAttribute("data-vendor")).toBeTruthy();
      });
    });

    describe("videos", () => {
      it("should not download or play a video at load", () => {
        const videos = all<HTMLVideoElement>(doc, "video:not(noscript video)");
        expect(videos.length).toBeGreaterThan(0);
        for (const video of videos) {
          expect(video.getAttribute("preload")).toBe("none");
          expect(video.hasAttribute("autoplay")).toBe(false);
        }
      });

      it("should autoplay a copy of each video without JavaScript", () => {
        const lazy = all<HTMLVideoElement>(doc, "video[data-lazy-video]");
        const fallbacks = all<HTMLVideoElement>(doc, "noscript video[autoplay]");
        const src = (v: HTMLVideoElement): string[] =>
          all<HTMLSourceElement>(v, "source").map((s) => s.getAttribute("src")!);
        expect(fallbacks.map(src)).toEqual(lazy.map(src));
      });

      it("should serve every video from the CDN", () => {
        for (const source of all<HTMLSourceElement>(doc, "video source"))
          expect(source.getAttribute("src")).toMatch(/^https:\/\/synnax\.nyc3\.cdn\./);
      });
    });

    describe("images", () => {
      it("should load only the hero screenshot at high priority", () => {
        const eager = all<HTMLImageElement>(doc, 'img[fetchpriority="high"]');
        expect(eager.map((img) => img.alt)).toEqual(["Synnax Console"]);
        expect(eager[0].getAttribute("loading")).toBe("eager");
      });

      it("should load every other CDN image lazily", () => {
        const images = cdnImages(doc).filter((img) => img.alt !== "Synnax Console");
        expect(images.length).toBeGreaterThan(0);
        for (const img of images) expect(img.getAttribute("loading")).toBe("lazy");
      });

      it("should reserve the size of every CDN image", () => {
        for (const img of cdnImages(doc)) {
          expect(img.getAttribute("width")).toMatch(/^\d+$/);
          expect(img.getAttribute("height")).toMatch(/^\d+$/);
        }
      });

      it("should describe every image", () => {
        for (const img of all<HTMLImageElement>(doc, "img"))
          expect(img.getAttribute("alt")).not.toBeNull();
      });
    });

    describe("structured data", () => {
      const script = (): HTMLScriptElement =>
        doc.querySelector('script[type="application/ld+json"]')!;

      it("should not let the JSON close its script tag", () => {
        expect(script().textContent).not.toContain("<");
      });

      it("should describe the organization with an absolute logo URL", () => {
        const data = JSON.parse(script().textContent) as {
          "@graph": Array<Record<string, unknown>>;
        };
        const organization = data["@graph"].find((n) => n["@type"] === "Organization")!;
        expect(organization.url).toBe("https://www.synnaxlabs.com");
        expect(String(organization.logo)).toMatch(/^https:\/\/www\.synnaxlabs\.com\//);
        const app = data["@graph"].find((n) => n["@type"] === "SoftwareApplication")!;
        expect(app.publisher).toEqual({ "@id": organization["@id"] });
      });
    });
  });

  describe("company", () => {
    it("should load every image lazily", async () => {
      const doc = await render(Company, "/company");
      expect(doc.title).toBe("Company | Synnax");
      const images = cdnImages(doc);
      expect(images.length).toBeGreaterThan(0);
      for (const img of images) expect(img.getAttribute("loading")).toBe("lazy");
    });
  });

  describe("sponsorships", () => {
    let doc: Document;

    beforeAll(async () => {
      stubImageSize(1600, 900);
      doc = await render(Sponsorships, "/sponsorships");
    });

    it("should show each university logo once", () => {
      const names = all<HTMLImageElement>(doc, ".sponsorships img").map((i) => i.alt);
      expect(names).toContain("UCLA");
      expect(names).toContain("University of Cincinnati");
      expect(new Set(names).size).toBe(names.length);
    });

    it("should serve SVG logos as they are and PNG logos through the CDN", () => {
      const svg = doc.querySelector<HTMLImageElement>('img[alt="UCLA"]')!;
      expect(svg.closest("picture")).toBeNull();
      expect(svg.getAttribute("src")).toMatch(/ucla\.svg/);
      const png = doc.querySelector<HTMLImageElement>('img[alt="Florida Rocket Lab"]')!;
      expect(png.closest("picture")).not.toBeNull();
      expect(decodeURIComponent(png.src)).toContain(`${CDN}landing/images/frl.png`);
    });

    it("should scale a logo by its own factor and leave the rest at 1", () => {
      const scale = (name: string): string | undefined =>
        doc
          .querySelector(`img[alt="${name}"]`)!
          .closest<HTMLElement>("[style*='--logo-scale']")
          ?.style.getPropertyValue("--logo-scale");
      expect(scale("San Diego State")).toBe("1.3");
      expect(scale("UC San Diego")).toBe("0.7");
      expect(scale("UCLA")).toBe("1");
    });
  });

  describe("privacy", () => {
    it("should render under its own title", async () => {
      const doc = await render(Privacy, "/privacy");
      expect(doc.title).toBe("Privacy policy | Synnax");
    });
  });

  describe("sitemap", () => {
    it("should list every page on the site", async () => {
      const response = GET({ site: new URL(config.site!) });
      const xml = await response.text();
      const locs = [...xml.matchAll(/<loc>([^<]+)<\/loc>/g)].map((m) => m[1]);
      expect(locs.sort()).toEqual([
        "https://www.synnaxlabs.com/",
        "https://www.synnaxlabs.com/company",
        "https://www.synnaxlabs.com/privacy",
        "https://www.synnaxlabs.com/sponsorships",
      ]);
    });
  });
});
