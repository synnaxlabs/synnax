// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import reactRenderer from "@astrojs/react/server.js";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { renderToString } from "react-dom/server";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { builtinEnvironments } from "vitest/runtime";

import { Media } from "@/components/media";
import { start } from "@/components/media/lightbox";
import Lightbox from "@/components/media/Lightbox.astro";
import { REDUCED_MOTION } from "@/components/media/video";

let dialogHTML = "";
let duration = 10;
let teardown: (global: typeof globalThis) => unknown = () => {};

const show = (media: string): void => {
  const article = `<article><h2>Create a plot</h2>${media}</article>`;
  document.body.innerHTML = dialogHTML + article;
  document.dispatchEvent(new Event("astro:after-swap"));
};

const find = <E extends Element>(selector: string): E => {
  const el = document.querySelector<E>(selector);
  if (el == null) throw new Error(`no element matches ${selector}`);
  return el;
};

const dialog = (): HTMLDialogElement => find(".docs-lightbox");

const control = (name: string): HTMLElement => find(`.docs-lightbox__${name}`);

const openVideo = (): HTMLVideoElement => {
  show(renderToString(<Media.Video id="clip" />));
  const video = find<HTMLVideoElement>("video");
  video.click();
  return video;
};

const press = (key: string): void => {
  dialog().dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true }));
};

const scrub = (value: number): void => {
  const input = find<HTMLInputElement>(".docs-lightbox__scrubber > input");
  input.value = String(value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
};

describe("lightbox", () => {
  beforeAll(async () => {
    const container = await AstroContainer.create();
    container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
    dialogHTML = await container.renderToString(Lightbox);
    // Astro renders only outside the browser, so the DOM arrives after the render.
    ({ teardown } = await builtinEnvironments.jsdom.setup(globalThis, {}));
    // Reduced motion skips the open and close animations, which jsdom cannot run.
    vi.stubGlobal("matchMedia", (query: string) => ({
      matches: query === REDUCED_MOTION,
      addEventListener: () => {},
    }));
    // jsdom has no modal dialogs or media playback, so these act as the browser would.
    Object.defineProperties(HTMLDialogElement.prototype, {
      showModal: {
        configurable: true,
        value(this: HTMLDialogElement) {
          this.open = true;
        },
      },
      close: {
        configurable: true,
        value(this: HTMLDialogElement) {
          this.open = false;
          this.dispatchEvent(new Event("close"));
        },
      },
    });
    Object.defineProperty(Element.prototype, "getAnimations", {
      configurable: true,
      value: () => [],
    });
    const paused = new WeakMap<HTMLMediaElement, boolean>();
    const times = new WeakMap<HTMLMediaElement, number>();
    vi.spyOn(HTMLMediaElement.prototype, "paused", "get").mockImplementation(function (
      this: HTMLMediaElement,
    ) {
      return paused.get(this) ?? true;
    });
    vi.spyOn(HTMLMediaElement.prototype, "play").mockImplementation(function (
      this: HTMLMediaElement,
    ) {
      paused.set(this, false);
      return Promise.resolve();
    });
    vi.spyOn(HTMLMediaElement.prototype, "pause").mockImplementation(function (
      this: HTMLMediaElement,
    ) {
      paused.set(this, true);
    });
    vi.spyOn(HTMLMediaElement.prototype, "duration", "get").mockImplementation(
      () => duration,
    );
    vi.spyOn(HTMLMediaElement.prototype, "currentTime", "get").mockImplementation(
      function (this: HTMLMediaElement) {
        return times.get(this) ?? 0;
      },
    );
    vi.spyOn(HTMLMediaElement.prototype, "currentTime", "set").mockImplementation(
      function (this: HTMLMediaElement, time: number) {
        if (!Number.isFinite(time)) throw new TypeError("non-finite currentTime");
        times.set(this, time);
      },
    );
    start();
  });

  afterAll(async () => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    await teardown(globalThis);
  });

  beforeEach(() => {
    duration = 10;
  });

  describe("open and close", () => {
    it("should move a clicked video into the dialog and hold its place", () => {
      const video = openVideo();
      expect(dialog().open).toBe(true);
      expect(control("stage").contains(video)).toBe(true);
      expect(
        document.querySelector("article .docs-lightbox__placeholder"),
      ).not.toBeNull();
    });

    it("should caption a video with the heading of its section", () => {
      openVideo();
      expect(control("caption").textContent).toBe("Create a plot");
      expect(dialog().getAttribute("aria-label")).toBe("Create a plot");
    });

    it("should caption an image with its text alternative", () => {
      show(renderToString(<Media.Image id="still" alt="The plot toolbar" />));
      find<HTMLElement>(".docs-image").click();
      expect(control("caption").textContent).toBe("The plot toolbar");
    });

    it("should return the video to the page when the close button is clicked", () => {
      const video = openVideo();
      control("close").click();
      expect(dialog().open).toBe(false);
      expect(document.querySelector("article")?.contains(video)).toBe(true);
      expect(document.querySelector(".docs-lightbox__placeholder")).toBeNull();
    });

    it("should close on Escape", () => {
      openVideo();
      dialog().dispatchEvent(new Event("cancel", { cancelable: true }));
      expect(dialog().open).toBe(false);
    });
  });

  describe("video controls", () => {
    it("should play and pause from the buttons", () => {
      const video = openVideo();
      control("play").click();
      expect(video.paused).toBe(false);
      control("pause").click();
      expect(video.paused).toBe(true);
    });

    it("should seek to the scrubber position", () => {
      const video = openVideo();
      scrub(0.5);
      expect(video.currentTime).toBe(5);
    });

    it("should step with the arrow keys within the video", () => {
      const video = openVideo();
      press("ArrowLeft");
      expect(video.currentTime).toBe(0);
      scrub(0.95);
      press("ArrowRight");
      expect(video.currentTime).toBe(10);
    });

    it("should not seek before the video has a duration", () => {
      duration = NaN;
      const errors: unknown[] = [];
      const onError = (e: ErrorEvent): number => errors.push(e.error);
      window.addEventListener("error", onError);
      const video = openVideo();
      scrub(0.5);
      press("ArrowRight");
      window.removeEventListener("error", onError);
      expect(errors).toEqual([]);
      expect(video.currentTime).toBe(0);
    });

    it("should cycle the speed and reset it on close", () => {
      const video = openVideo();
      control("speed").click();
      expect(video.playbackRate).toBe(1.5);
      expect(control("speed").textContent).toBe("1.5×");
      control("close").click();
      expect(video.playbackRate).toBe(1);
    });
  });
});
