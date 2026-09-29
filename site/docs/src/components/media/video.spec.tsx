// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderToString } from "react-dom/server";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { Video } from "@/components/media/Media";
import { start } from "@/components/media/video";

let touch = false;
let schemeChange: (() => void) | null = null;
let intersect: IntersectionObserverCallback | null = null;
const load = vi.fn();

class Observer {
  constructor(callback: IntersectionObserverCallback) {
    intersect = callback;
  }
  observe = vi.fn();
  disconnect = vi.fn();
}

const show = (...ids: string[]): HTMLVideoElement[] => {
  document.body.innerHTML = ids.map((id) => renderToString(<Video id={id} />)).join("");
  document.dispatchEvent(new Event("astro:after-swap"));
  return [...document.querySelectorAll("video")];
};

const scroll = (video: HTMLVideoElement, isIntersecting: boolean): void =>
  intersect?.(
    [{ target: video, isIntersecting } as unknown as IntersectionObserverEntry],
    {} as IntersectionObserver,
  );

const playing = (video: HTMLVideoElement): boolean =>
  video.parentElement?.classList.contains("docs-video--playing") ?? false;

describe("video", () => {
  beforeAll(() => {
    vi.stubGlobal("IntersectionObserver", Observer);
    vi.stubGlobal("matchMedia", (query: string) => ({
      matches: query.includes("pointer: coarse") && touch,
      addEventListener: (_: string, listener: () => void) => {
        if (query.includes("color-scheme")) schemeChange = listener;
      },
    }));
    // jsdom has no media playback, so these act as the browser would.
    const paused = new WeakMap<HTMLMediaElement, boolean>();
    vi.spyOn(HTMLMediaElement.prototype, "paused", "get").mockImplementation(function (
      this: HTMLMediaElement,
    ) {
      return paused.get(this) ?? true;
    });
    vi.spyOn(HTMLMediaElement.prototype, "play").mockImplementation(function (
      this: HTMLMediaElement,
    ) {
      paused.set(this, false);
      this.dispatchEvent(new Event("play"));
      return Promise.resolve();
    });
    vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(function (
      this: HTMLMediaElement,
    ) {
      paused.set(this, true);
      this.dispatchEvent(new Event("pause"));
    });
    vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(function (
      this: HTMLMediaElement,
    ) {
      load(this);
    });
    start();
  });

  afterAll(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  beforeEach(() => {
    touch = false;
    load.mockClear();
  });

  describe("pointer devices", () => {
    it("should play a video in view and hide its play button", () => {
      const [video] = show("clip");
      scroll(video, true);
      expect(video.paused).toBe(false);
      expect(playing(video)).toBe(true);
    });

    it("should pause a video that leaves the view", () => {
      const [video] = show("clip");
      scroll(video, true);
      scroll(video, false);
      expect(video.paused).toBe(true);
      expect(playing(video)).toBe(false);
    });

    it("should ignore taps", () => {
      const [video] = show("clip");
      video.click();
      expect(video.paused).toBe(true);
    });
  });

  describe("touch devices", () => {
    beforeEach(() => {
      touch = true;
    });

    it("should wait for a tap instead of playing in view", () => {
      const [video] = show("clip");
      scroll(video, true);
      expect(video.paused).toBe(true);
    });

    it("should toggle playback on a tap on the video or its play button", () => {
      const [video] = show("clip");
      (document.querySelector(".docs-video__play") as HTMLElement).click();
      expect(playing(video)).toBe(true);
      video.click();
      expect(playing(video)).toBe(false);
    });

    it("should pause a video that leaves the view", () => {
      const [video] = show("clip");
      video.click();
      scroll(video, false);
      expect(video.paused).toBe(true);
    });
  });

  it("should reload themed videos when the color scheme changes", () => {
    document.body.innerHTML =
      renderToString(<Video id="themed" />) +
      renderToString(<Video id="plain" themed={false} />);
    schemeChange?.();
    expect(load).toHaveBeenCalledExactlyOnceWith(document.querySelector("video"));
  });
});
