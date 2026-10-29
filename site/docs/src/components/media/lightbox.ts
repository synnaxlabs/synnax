// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { REDUCED_MOTION, TOUCH } from "@/components/media/video";

type Media = HTMLImageElement | HTMLVideoElement;

interface Expanded {
  media: Media;
  /** The element that moved into the lightbox: the media, or a themed image's picture. */
  moved: Element;
  /** The hidden copy that holds the media's place on the page. */
  placeholder: Element;
  closing: boolean;
}

const OPEN_MS = 280;
const CLOSE_MS = 220;
const EASING = "cubic-bezier(0.2, 0, 0, 1)";
// Arrow keys step a video by this many seconds.
const STEP = 1;
// Fade in and out with the backdrop.
const CONTROLS = ".docs-lightbox__close, .docs-lightbox__bar";
// The speed button cycles through these rates.
const RATES = [1, 1.5, 2, 0.5];

// A video has no duration to seek in until its metadata loads.
const seek = (video: HTMLVideoElement, time: number): void => {
  if (Number.isFinite(video.duration))
    video.currentTime = Math.min(Math.max(time, 0), video.duration);
};

const query = <E extends Element>(root: ParentNode, selector: string): E => {
  const el = root.querySelector<E>(selector);
  if (el == null) throw new Error(`no element matches ${selector}`);
  return el;
};

const clock = (seconds: number): string => {
  const whole = Math.floor(Number.isFinite(seconds) ? seconds : 0);
  return `${Math.floor(whole / 60)}:${String(whole % 60).padStart(2, "0")}`;
};

// An image's text alternative names it best. Otherwise the heading of its section does.
const caption = (media: Media): string => {
  if (media instanceof HTMLImageElement && media.alt !== "") return media.alt;
  const headings = [...document.querySelectorAll("article :is(h1, h2, h3)")].filter(
    (h) => h.compareDocumentPosition(media) & Node.DOCUMENT_POSITION_FOLLOWING,
  );
  return headings.at(-1)?.textContent.trim() ?? "";
};

// The transform that draws media laid out in box `at` over box `over`. The media's top
// left corner is its transform origin.
const offset = (over: DOMRect, at: DOMRect): string =>
  `translate(${over.left - at.left}px, ${over.top - at.top}px) ` +
  `scale(${over.width / at.width}, ${over.height / at.height})`;

const fade = (el: Element, keyframes: Keyframe[], duration: number, pseudo?: string) =>
  el.animate(keyframes, {
    duration,
    easing: EASING,
    fill: "forwards",
    pseudoElement: pseudo,
  });

const IN = [{ opacity: 0 }, { opacity: 1 }];
const OUT = [{ opacity: 1 }, { opacity: 0 }];

/**
 * Lets a reader expand a docs video or image to fill most of the screen. A click on an
 * image, on a video's expand button, or on a video itself away from touch devices opens
 * it; Escape, the backdrop, or the close button returns it. Call it once; it follows
 * page swaps by itself.
 */
export const start = (): void => {
  let expanded: Expanded | null = null;
  let frame = 0;

  const still = (): boolean => window.matchMedia(REDUCED_MOTION).matches;
  const dialog = (): HTMLDialogElement =>
    query<HTMLDialogElement>(document, ".docs-lightbox");

  const drawControls = (): void => {
    if (!(expanded?.media instanceof HTMLVideoElement)) return;
    const video = expanded.media;
    const root = dialog();
    const playing = root.classList.contains("docs-lightbox--playing");
    if (playing === video.paused) {
      // The play and pause buttons trade places, so focus moves to the one that shows.
      const focused = root.querySelector(
        ".docs-lightbox__play:focus, .docs-lightbox__pause:focus",
      );
      root.classList.toggle("docs-lightbox--playing", !video.paused);
      if (focused != null)
        query<HTMLElement>(
          root,
          video.paused ? ".docs-lightbox__play" : ".docs-lightbox__pause",
        ).focus();
    }
    const position = video.duration > 0 ? video.currentTime / video.duration : 0;
    const scrubber = query<HTMLInputElement>(root, ".docs-lightbox__scrubber > input");
    // The reader's drag owns the scrubber while it lasts.
    if (!scrubber.matches(":active")) scrubber.value = String(position);
    query<HTMLElement>(
      root,
      ".docs-lightbox__scrubber .lyra-progress-bar",
    ).style.width = `${Number(scrubber.value) * 100}%`;
    query(root, ".docs-lightbox__time").textContent =
      `${clock(video.currentTime)} / ${clock(video.duration)}`;
    frame = requestAnimationFrame(drawControls);
  };

  const setRate = (video: HTMLVideoElement, rate: number): void => {
    video.playbackRate = rate;
    const button = query(dialog(), ".docs-lightbox__speed");
    button.textContent = `${rate}×`;
    button.setAttribute("aria-label", `Speed ${rate}×`);
  };

  const toggle = (video: HTMLVideoElement): void => {
    if (video.paused) video.play().catch(console.error);
    else video.pause();
  };

  const open = (media: Media): void => {
    if (expanded != null) return;
    const root = dialog();
    const from = media.getBoundingClientRect();
    const name = caption(media);
    const moved =
      media instanceof HTMLImageElement ? (media.closest("picture") ?? media) : media;
    const placeholder = document.importNode(moved, true);
    placeholder.classList.add("docs-lightbox__placeholder");
    placeholder.setAttribute("aria-hidden", "true");
    if (placeholder instanceof HTMLVideoElement)
      placeholder.style.aspectRatio = `${from.width} / ${from.height}`;
    moved.replaceWith(placeholder);
    query(root, ".docs-lightbox__stage").append(moved);
    query(root, ".docs-lightbox__caption").textContent = name;
    root.setAttribute("aria-label", name === "" ? "Expanded media" : name);
    const video = media instanceof HTMLVideoElement;
    root.classList.toggle("docs-lightbox--video", video);
    expanded = { media, moved, placeholder, closing: false };
    root.showModal();
    // On the dialog, not its first button, so Space plays and pauses a video.
    root.focus();
    if (media instanceof HTMLVideoElement) {
      if (!still()) media.play().catch(console.error);
      drawControls();
    }
    if (still()) return;
    const to = media.getBoundingClientRect();
    media.animate([{ transform: offset(from, to) }, { transform: "none" }], {
      duration: OPEN_MS,
      easing: EASING,
    });
    fade(root, IN, OPEN_MS, "::backdrop");
    for (const el of root.querySelectorAll(CONTROLS)) fade(el, IN, OPEN_MS);
  };

  const close = (): void => {
    if (expanded == null || expanded.closing) return;
    expanded.closing = true;
    const root = dialog();
    const { media, placeholder } = expanded;
    if (still()) {
      root.close();
      return;
    }
    const slot =
      placeholder instanceof HTMLPictureElement
        ? query(placeholder, "img")
        : placeholder;
    const at = media.getBoundingClientRect();
    fade(root, OUT, CLOSE_MS, "::backdrop");
    for (const el of root.querySelectorAll(CONTROLS)) fade(el, OUT, CLOSE_MS / 2);
    const flight = media.animate(
      [{ transform: "none" }, { transform: offset(slot.getBoundingClientRect(), at) }],
      { duration: CLOSE_MS, easing: EASING, fill: "forwards" },
    );
    flight.onfinish = () => root.close();
  };

  // The close event follows every way the dialog closes, so the media returns here.
  const restore = (): void => {
    if (expanded == null) return;
    cancelAnimationFrame(frame);
    const { media, moved, placeholder } = expanded;
    if (media instanceof HTMLVideoElement) setRate(media, 1);
    for (const animation of dialog().getAnimations({ subtree: true }))
      animation.cancel();
    placeholder.replaceWith(moved);
    expanded = null;
  };

  document.addEventListener(
    "close",
    (e) => {
      if (e.target instanceof HTMLDialogElement && e.target.matches(".docs-lightbox"))
        restore();
    },
    { capture: true },
  );

  document.addEventListener(
    "cancel",
    (e) => {
      if (!(e.target instanceof Element) || !e.target.matches(".docs-lightbox")) return;
      // Without a recent click, the browser closes the dialog at once, and the close
      // event returns the media.
      if (!e.cancelable) return;
      e.preventDefault();
      close();
    },
    { capture: true },
  );

  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    const target = e.target;
    if (target.closest(".docs-lightbox") != null) {
      const video = expanded?.media instanceof HTMLVideoElement ? expanded.media : null;
      if (
        video != null &&
        target.closest(".docs-lightbox__play, .docs-lightbox__pause")
      ) {
        toggle(video);
        return;
      }
      if (video != null && target.closest(".docs-lightbox__replay") != null) {
        video.currentTime = 0;
        video.play().catch(console.error);
        return;
      }
      if (video != null && target.closest(".docs-lightbox__speed") != null) {
        setRate(video, RATES[(RATES.indexOf(video.playbackRate) + 1) % RATES.length]);
        return;
      }
      if (target.closest(".docs-lightbox__controls") != null) return;
      if (target === expanded?.media && expanded.media instanceof HTMLVideoElement) {
        toggle(expanded.media);
        return;
      }
      close();
      return;
    }
    const expand = target.closest(".docs-video__expand");
    if (expand != null) {
      const video = expand.closest(".docs-video")?.querySelector("video");
      if (video != null) open(video);
      return;
    }
    if (
      target instanceof HTMLVideoElement &&
      target.parentElement?.matches(".docs-video") === true &&
      !window.matchMedia(TOUCH).matches
    ) {
      open(target);
      return;
    }
    if (target instanceof HTMLImageElement && target.matches(".docs-image"))
      if (target.closest("a") == null) open(target);
  });

  document.addEventListener("input", (e) => {
    if (
      !(e.target instanceof HTMLInputElement) ||
      !e.target.matches(".docs-lightbox__scrubber > input") ||
      !(expanded?.media instanceof HTMLVideoElement)
    )
      return;
    seek(expanded.media, Number(e.target.value) * expanded.media.duration);
  });

  document.addEventListener("keydown", (e) => {
    if (!(expanded?.media instanceof HTMLVideoElement)) return;
    const video = expanded.media;
    const onControl = e.target instanceof Element && e.target.closest("button, input");
    if (e.key === " " && onControl == null) {
      e.preventDefault();
      toggle(video);
    } else if ((e.key === "ArrowLeft" || e.key === "ArrowRight") && onControl == null) {
      e.preventDefault();
      seek(video, video.currentTime + (e.key === "ArrowLeft" ? -STEP : STEP));
    }
  });

  document.addEventListener("astro:after-swap", () => {
    cancelAnimationFrame(frame);
    expanded = null;
  });
};
