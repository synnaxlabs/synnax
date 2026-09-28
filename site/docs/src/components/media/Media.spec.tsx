// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, assert, beforeEach, describe, expect, it, vi } from "vitest";

import { Video } from "@/components/media/Media";

class MockIntersectionObserver {
  static current: MockIntersectionObserver | undefined;
  callback: IntersectionObserverCallback;
  constructor(callback: IntersectionObserverCallback) {
    this.callback = callback;
    MockIntersectionObserver.current = this;
  }
  observe = vi.fn();
  unobserve = vi.fn();
  intersect(isIntersecting: boolean): void {
    act(() => {
      this.callback(
        [{ isIntersecting } as IntersectionObserverEntry],
        this as unknown as IntersectionObserver,
      );
    });
  }
}

describe("Video", () => {
  let touch: boolean;
  let paused: boolean;

  const renderVideo = (): HTMLVideoElement => {
    const { container } = render(<Video id="clip" />);
    const video = container.querySelector("video");
    assert(video != null);
    return video;
  };

  const observer = (): MockIntersectionObserver => {
    assert(MockIntersectionObserver.current != null);
    return MockIntersectionObserver.current;
  };

  const overlay = (): Element | null => document.querySelector(".docs-video__play");

  beforeEach(() => {
    touch = false;
    paused = true;
    MockIntersectionObserver.current = undefined;
    vi.stubGlobal("IntersectionObserver", MockIntersectionObserver);
    vi.stubGlobal("matchMedia", (query: string) => ({
      matches: query.includes("pointer: coarse") && touch,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }));
    // jsdom has no media playback, so these act as the browser would.
    vi.spyOn(HTMLMediaElement.prototype, "paused", "get").mockImplementation(
      () => paused,
    );
    vi.spyOn(HTMLMediaElement.prototype, "load").mockImplementation(() => {});
    vi.spyOn(HTMLMediaElement.prototype, "play").mockImplementation(function (
      this: HTMLMediaElement,
    ) {
      paused = false;
      fireEvent.play(this);
      return Promise.resolve();
    });
    vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(function (
      this: HTMLMediaElement,
    ) {
      paused = true;
      fireEvent.pause(this);
    });
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("autoplays in view without an overlay on pointer devices", () => {
    renderVideo();
    observer().intersect(true);
    expect(paused).toBe(false);
    expect(overlay()).toBeNull();
  });

  describe("touch devices", () => {
    beforeEach(() => {
      touch = true;
    });

    it("waits for a tap instead of autoplaying in view", () => {
      renderVideo();
      observer().intersect(true);
      expect(paused).toBe(true);
      expect(overlay()).not.toBeNull();
    });

    it("toggles playback and the overlay on tap", () => {
      const video = renderVideo();
      fireEvent.click(video);
      expect(paused).toBe(false);
      expect(overlay()).toBeNull();
      fireEvent.click(video);
      expect(paused).toBe(true);
      expect(overlay()).not.toBeNull();
    });

    it("pauses and shows the overlay out of view", () => {
      const video = renderVideo();
      fireEvent.click(video);
      observer().intersect(false);
      expect(paused).toBe(true);
      expect(overlay()).not.toBeNull();
    });
  });
});
