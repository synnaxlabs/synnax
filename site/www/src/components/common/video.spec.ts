// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

import { start } from "@/components/common/video";

let intersect: IntersectionObserverCallback;
let options: IntersectionObserverInit | undefined;
const observe = vi.fn();
const unobserve = vi.fn();
const load = vi.fn();

class Observer {
  constructor(callback: IntersectionObserverCallback, init?: IntersectionObserverInit) {
    intersect = callback;
    options = init;
  }
  observe = observe;
  unobserve = unobserve;
}

const scroll = (video: HTMLVideoElement, isIntersecting: boolean): void =>
  intersect(
    [{ target: video, isIntersecting } as unknown as IntersectionObserverEntry],
    {} as IntersectionObserver,
  );

const videos = (): HTMLVideoElement[] => [...document.querySelectorAll("video")];

describe("video", () => {
  beforeEach(() => {
    observe.mockClear();
    unobserve.mockClear();
    load.mockClear();
    vi.stubGlobal("IntersectionObserver", Observer);
    // jsdom has no media loading.
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
  });

  it("should observe each lazy video and no other video", () => {
    const [first, second] = videos();
    expect(observe.mock.calls).toEqual([[first], [second]]);
  });

  it("should watch one viewport above and below the screen", () => {
    expect(options).toEqual({ rootMargin: "100% 0px" });
  });

  it("should not load a video before it comes near the screen", () => {
    expect(load).not.toHaveBeenCalled();
    expect(videos().map((v) => v.autoplay)).toEqual([false, false, false]);
  });

  it("should load and autoplay a video that comes near the screen", () => {
    const [first, second] = videos();
    scroll(first, true);
    expect(load.mock.calls).toEqual([[first]]);
    expect(first.autoplay).toBe(true);
    expect(second.autoplay).toBe(false);
  });

  it("should stop observing a video once it loads", () => {
    const [first] = videos();
    scroll(first, true);
    expect(unobserve).toHaveBeenCalledExactlyOnceWith(first);
  });

  it("should ignore a video that leaves the screen", () => {
    const [first] = videos();
    scroll(first, false);
    expect(load).not.toHaveBeenCalled();
    expect(unobserve).not.toHaveBeenCalled();
    expect(first.autoplay).toBe(false);
  });
});
