// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { beforeAll, describe, expect, it } from "vitest";

import LazyVideo from "@/components/common/LazyVideo.astro";
import { renderAstro } from "@/testutil";

const CDN = "https://synnax.nyc3.cdn.digitaloceanspaces.com/landing";

const sources = (video: HTMLVideoElement): Array<[string | null, string]> =>
  [...video.querySelectorAll("source")].map((s) => [
    s.getAttribute("media"),
    s.getAttribute("src") ?? "",
  ]);

describe("LazyVideo", () => {
  let doc: Document;
  let lazy: HTMLVideoElement;
  let fallback: HTMLVideoElement;

  beforeAll(async () => {
    doc = await renderAstro(LazyVideo, {
      props: {
        wide: "videos/demo-2560.mp4",
        narrow: "videos/demo-1280.mp4",
        width: 2560,
        height: 1440,
        class: "demo-video",
      },
    });
    lazy = doc.querySelector("video[data-lazy-video]")!;
    fallback = doc.querySelector("noscript video")!;
  });

  describe("lazy video", () => {
    it("should not download or play before the script starts it", () => {
      expect(lazy.getAttribute("preload")).toBe("none");
      expect(lazy.hasAttribute("autoplay")).toBe(false);
    });

    it("should play muted, looped, and inline", () => {
      expect(lazy.hasAttribute("muted")).toBe(true);
      expect(lazy.hasAttribute("loop")).toBe(true);
      expect(lazy.hasAttribute("playsinline")).toBe(true);
    });

    it("should serve the narrow encode to narrow screens", () => {
      expect(sources(lazy)).toEqual([
        ["(max-width: 900px)", `${CDN}/videos/demo-1280.mp4`],
        [null, `${CDN}/videos/demo-2560.mp4`],
      ]);
    });

    it("should reserve its size and take the class", () => {
      expect(lazy.getAttribute("width")).toBe("2560");
      expect(lazy.getAttribute("height")).toBe("1440");
      expect(lazy.className).toBe("demo-video");
    });
  });

  describe("without JavaScript", () => {
    it("should autoplay a copy with the same sources", () => {
      expect(fallback.hasAttribute("autoplay")).toBe(true);
      expect(fallback.hasAttribute("muted")).toBe(true);
      expect(fallback.hasAttribute("preload")).toBe(false);
      expect(sources(fallback)).toEqual(sources(lazy));
    });

    it("should give the copy the same size and class", () => {
      expect(fallback.getAttribute("width")).toBe("2560");
      expect(fallback.getAttribute("height")).toBe("1440");
      expect(fallback.className).toBe("demo-video");
    });

    it("should hide the lazy video from inside the noscript", () => {
      const style = doc.querySelector("noscript style")!;
      expect(style.textContent).toMatch(
        /video\[data-lazy-video\]\s*{\s*display: none !important;\s*}/,
      );
      expect(doc.querySelectorAll("style:not(noscript style)")).toHaveLength(0);
    });
  });
});
