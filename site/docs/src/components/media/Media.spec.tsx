// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderToString } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { Media } from "@/components/media";
import { CDN } from "@/components/media/cdn";
import versions from "@/components/media/videos.json";

const [[ID, VERSION]] = Object.entries(versions);

const show = (html: string): void => {
  document.body.innerHTML = html;
};

const sources = (): [string | null, string | null][] =>
  [...document.querySelectorAll("source")].map((source) => [
    source.getAttribute("media"),
    source.getAttribute("src") ?? source.getAttribute("srcset"),
  ]);

describe("Media", () => {
  describe("Video", () => {
    it("should offer the dark and light versions by color scheme", () => {
      show(renderToString(<Media.Video id={ID} />));
      expect(sources()).toEqual([
        [
          "(prefers-color-scheme: dark)",
          `${CDN.ROOT}/${ID}/${VERSION}-dark.mp4#t=0.001`,
        ],
        [null, `${CDN.ROOT}/${ID}/${VERSION}-light.mp4#t=0.001`],
      ]);
    });

    it("should offer one version when not themed", () => {
      show(renderToString(<Media.Video id={ID} themed={false} />));
      expect(sources()).toEqual([[null, `${CDN.ROOT}/${ID}/${VERSION}.mp4#t=0.001`]]);
    });

    it("should throw for a video with no uploaded version", () => {
      expect(() => renderToString(<Media.Video id="console/missing" />)).toThrow(
        "no version of video console/missing: add an entry to studio/videos.ts, then run pnpm upload console/missing in studio/",
      );
    });

    it("should render a muted looping video that loads only its metadata", () => {
      show(renderToString(<Media.Video id={ID} />));
      const video = document.querySelector(".docs-video > video") as HTMLVideoElement;
      expect(video.loop).toBe(true);
      expect(video.hasAttribute("muted")).toBe(true);
      expect(video.hasAttribute("playsinline")).toBe(true);
      expect(video.getAttribute("preload")).toBe("metadata");
      expect(document.querySelector(".docs-video__play")).not.toBeNull();
    });
  });

  describe("Image", () => {
    it("should offer the dark version by color scheme and fall back to light", () => {
      show(renderToString(<Media.Image id="shot" alt="Shot" />));
      expect(sources()).toEqual([
        ["(prefers-color-scheme: dark)", `${CDN.ROOT}/shot-dark.png`],
      ]);
      const img = document.querySelector("picture > img") as HTMLImageElement;
      expect(img.getAttribute("src")).toBe(`${CDN.ROOT}/shot-light.png`);
      expect(img.alt).toBe("Shot");
      expect(img.getAttribute("loading")).toBe("lazy");
    });

    it("should render a plain image when not themed", () => {
      show(renderToString(<Media.Image id="logo" extension="svg" themed={false} />));
      expect(document.querySelector("picture")).toBeNull();
      expect(document.querySelector("img")?.getAttribute("src")).toBe(
        `${CDN.ROOT}/logo.svg`,
      );
    });
  });
});
