// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { start } from "@/components/common/video";

// Stubs the observer and media loading that jsdom lacks, renders two lazy videos and a
// plain one, and starts the script.
const setup = () => {
  const observer = {
    observe: vi.fn(),
    unobserve: vi.fn(),
    options: undefined as IntersectionObserverInit | undefined,
    callback: (() => {}) as IntersectionObserverCallback,
  };
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      constructor(
        callback: IntersectionObserverCallback,
        init?: IntersectionObserverInit,
      ) {
        observer.callback = callback;
        observer.options = init;
      }
      observe = observer.observe;
      unobserve = observer.unobserve;
    },
  );
  const load = vi.fn();
  vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(function (
    this: HTMLMediaElement,
  ) {
    load(this);
  });
  document.body.innerHTML = `
    <video data-lazy-video preload="none" muted loop playsinline></video>
    <video data-lazy-video preload="none" muted loop playsinline></video>
    <video muted></video>`;
  start();
  const scroll = (video: HTMLVideoElement, isIntersecting: boolean): void =>
    observer.callback(
      [{ target: video, isIntersecting } as unknown as IntersectionObserverEntry],
      {} as IntersectionObserver,
    );
  return { observer, load, scroll, videos: [...document.querySelectorAll("video")] };
};

describe("video", () => {
  it("should observe each lazy video and no other video", () => {
    const { observer, videos } = setup();
    expect(observer.observe.mock.calls).toEqual([[videos[0]], [videos[1]]]);
  });

  it("should watch one viewport above and below the screen", () => {
    const { observer } = setup();
    expect(observer.options).toEqual({ rootMargin: "100% 0px" });
  });

  it("should not load a video before it comes near the screen", () => {
    const { load, videos } = setup();
    expect(load).not.toHaveBeenCalled();
    expect(videos.map((v) => v.autoplay)).toEqual([false, false, false]);
  });

  it("should load and autoplay a video that comes near the screen", () => {
    const { load, scroll, videos } = setup();
    const [first, second] = videos;
    scroll(first, true);
    expect(load.mock.calls).toEqual([[first]]);
    expect(first.autoplay).toBe(true);
    expect(second.autoplay).toBe(false);
  });

  it("should stop observing a video once it loads", () => {
    const { observer, scroll, videos } = setup();
    scroll(videos[0], true);
    expect(observer.unobserve).toHaveBeenCalledExactlyOnceWith(videos[0]);
  });

  it("should ignore a video that leaves the screen", () => {
    const { observer, load, scroll, videos } = setup();
    scroll(videos[0], false);
    expect(load).not.toHaveBeenCalled();
    expect(observer.unobserve).not.toHaveBeenCalled();
    expect(videos[0].autoplay).toBe(false);
  });
});
