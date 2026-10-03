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

import CdnPicture from "@/components/common/CdnPicture.astro";
import { renderAstro, stubImageSize } from "@/testutil";

const SRC = "https://synnax.nyc3.cdn.digitaloceanspaces.com/landing/images/hero.png";

interface Rendered {
  source: HTMLSourceElement;
  img: HTMLImageElement;
}

const render = async (props: Record<string, unknown> = {}): Promise<Rendered> => {
  const doc = await renderAstro(CdnPicture, {
    props: { path: "images/hero.png", alt: "Hero", sizes: "100vw", ...props },
  });
  return { source: doc.querySelector("source")!, img: doc.querySelector("img")! };
};

const widths = (srcset: string): number[] =>
  srcset.split(", ").map((entry) => Number(entry.split(" ").at(-1)!.slice(0, -1)));

const hrefs = (srcset: string): string[] =>
  srcset.split(", ").map((entry) => {
    const url = new URL(entry.split(" ")[0], "https://www.synnaxlabs.com");
    return url.searchParams.get("href") ?? "";
  });

describe("CdnPicture", () => {
  describe("source", () => {
    it("should read the image from the CDN landing folder", async () => {
      const fetch = stubImageSize(1600, 900);
      const { img } = await render();
      expect(fetch).toHaveBeenCalledOnce();
      expect((fetch.mock.calls[0][0] as Request).url).toBe(SRC);
      expect(new Set(hrefs(img.srcset))).toEqual(new Set([SRC]));
    });

    it("should offer AVIF sources with a WebP fallback", async () => {
      stubImageSize(1600, 900);
      const { source, img } = await render();
      expect(source.type).toBe("image/avif");
      expect(source.srcset).toContain("f=avif");
      expect(img.src).toContain("f=webp");
      expect(img.srcset).toContain("f=webp");
    });
  });

  describe("widths", () => {
    it("should stop at 2560 pixels for a wider source", async () => {
      stubImageSize(3831, 2146);
      const { source, img } = await render();
      expect(widths(source.srcset)).toEqual([640, 960, 1280, 1920, 2560]);
      expect(widths(img.srcset)).toEqual([640, 960, 1280, 1920, 2560]);
      expect(img.getAttribute("width")).toBe("2560");
      expect(img.getAttribute("height")).toBe("1434");
    });

    it("should end at the width of a narrower source", async () => {
      stubImageSize(1000, 500);
      const { source, img } = await render();
      expect(widths(source.srcset)).toEqual([640, 960, 1000]);
      expect(img.getAttribute("width")).toBe("1000");
      expect(img.getAttribute("height")).toBe("500");
    });

    it("should offer only the source width for a source under 640 pixels", async () => {
      stubImageSize(300, 200);
      const { source } = await render();
      expect(widths(source.srcset)).toEqual([300]);
    });
  });

  describe("loading", () => {
    it("should load lazily by default", async () => {
      stubImageSize(1600, 900);
      const { img } = await render();
      expect(img.getAttribute("loading")).toBe("lazy");
      expect(img.getAttribute("decoding")).toBe("async");
      expect(img.hasAttribute("fetchpriority")).toBe(false);
    });

    it("should load eagerly at high priority with priority", async () => {
      stubImageSize(1600, 900);
      const { img } = await render({ priority: true });
      expect(img.getAttribute("loading")).toBe("eager");
      expect(img.getAttribute("decoding")).toBe("sync");
      expect(img.getAttribute("fetchpriority")).toBe("high");
    });
  });

  it("should pass the alt text, sizes, and class to the image", async () => {
    stubImageSize(1600, 900);
    const { source, img } = await render({
      alt: "Console",
      sizes: "(max-width: 900px) 100vw, 1240px",
      class: "hero-screenshot",
    });
    expect(img.alt).toBe("Console");
    expect(img.sizes).toBe("(max-width: 900px) 100vw, 1240px");
    expect(source.sizes).toBe("(max-width: 900px) 100vw, 1240px");
    expect(img.className).toBe("hero-screenshot");
  });
});
