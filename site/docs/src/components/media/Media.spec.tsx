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
import videos from "@/components/media/videos.json";

const [[ID, { version: VERSION, width: WIDTH, height: HEIGHT }]] =
  Object.entries(videos);

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
      show(renderToString(<Media.Video id={ID} label="A video" />));
      expect(sources()).toEqual([
        [
          "(prefers-color-scheme: dark)",
          `${CDN.ROOT}/${ID}/${VERSION}-dark.mp4#t=0.001`,
        ],
        [null, `${CDN.ROOT}/${ID}/${VERSION}-light.mp4#t=0.001`],
      ]);
    });

    it("should offer one version when not themed", () => {
      show(renderToString(<Media.Video id={ID} label="A video" themed={false} />));
      expect(sources()).toEqual([[null, `${CDN.ROOT}/${ID}/${VERSION}.mp4#t=0.001`]]);
    });

    it("should throw for a video with no uploaded version", () => {
      expect(() =>
        renderToString(<Media.Video id="console/missing" label="A video" />),
      ).toThrow(
        "no version of video console/missing: add an entry to studio/videos.ts, then run pnpm upload console/missing in studio/",
      );
    });

    it("should render a muted looping video that loads only its metadata", () => {
      show(renderToString(<Media.Video id={ID} label="A video" />));
      const video = document.querySelector(".docs-video > video") as HTMLVideoElement;
      expect(video.loop).toBe(true);
      expect(video.hasAttribute("muted")).toBe(true);
      expect(video.hasAttribute("playsinline")).toBe(true);
      expect(video.getAttribute("preload")).toBe("metadata");
      expect(document.querySelector(".docs-video__play")).not.toBeNull();
    });

    it("should reserve the space of the video before it loads", () => {
      show(renderToString(<Media.Video id={ID} label="A video" />));
      expect(document.querySelector("video")?.style.aspectRatio).toBe(
        `auto ${WIDTH} / ${HEIGHT}`,
      );
    });

    it("should name the video with its label", () => {
      show(renderToString(<Media.Video id={ID} label="Create a line plot" />));
      expect(document.querySelector("video")?.getAttribute("aria-label")).toBe(
        "Create a line plot",
      );
    });
  });
});
