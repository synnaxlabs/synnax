// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Matches a touch device, which plays a video on a tap instead of in view. */
export const TOUCH = "(hover: none) and (pointer: coarse)";
/** Matches a reader who prefers the dark color scheme. */
export const DARK = "(prefers-color-scheme: dark)";
/** Matches a reader who asks for less motion. Videos then play only on request. */
export const REDUCED_MOTION = "(prefers-reduced-motion: reduce)";
const VIDEO = ".docs-video > video";
const PLAYING_CLASS = "docs-video--playing";
const PLAY = ".docs-video__play-button";
const PAUSE = ".docs-video__pause-button";
const REPLAY = ".docs-video__replay";
// Most of a video must be in view before it plays.
const VISIBLE = 0.85;

/**
 * Starts playing docs videos while they are in view, and draws their progress. A touch
 * device plays and pauses on a tap instead, to spare the reader's data, and a reader
 * who asks for less motion plays them by hand. Call it once; it follows page swaps by
 * itself.
 */
export const start = (): void => {
  const matches = (query: string): boolean => window.matchMedia(query).matches;
  const autoplay = (): boolean => !matches(TOUCH) && !matches(REDUCED_MOTION);
  // A video moves into the lightbox while it is expanded, so it keeps its frame here.
  const frames = new WeakMap<HTMLVideoElement, HTMLElement>();
  // Videos the reader paused, which stay paused when they scroll back into view.
  const held = new WeakSet<HTMLVideoElement>();
  const playing = new Set<HTMLVideoElement>();
  let observer: IntersectionObserver | null = null;
  let paint = 0;

  const bind = (): void => {
    observer?.disconnect();
    observer = new IntersectionObserver(
      (entries) => {
        for (const { target, isIntersecting } of entries) {
          if (!(target instanceof HTMLVideoElement)) continue;
          if (!isIntersecting) target.pause();
          else if (autoplay() && !held.has(target)) target.play().catch(console.error);
        }
      },
      { threshold: VISIBLE },
    );
    for (const video of document.querySelectorAll<HTMLVideoElement>(VIDEO)) {
      if (video.parentElement != null) frames.set(video, video.parentElement);
      observer.observe(video);
    }
  };

  const drawProgress = (): void => {
    for (const video of playing) {
      const bar = frames.get(video)?.querySelector<HTMLElement>(".lyra-progress-bar");
      if (bar != null)
        bar.style.width = `${(video.currentTime / video.duration) * 100}%`;
    }
    paint = playing.size > 0 ? requestAnimationFrame(drawProgress) : 0;
  };

  const onPlayback = (e: Event): void => {
    if (!(e.target instanceof HTMLVideoElement)) return;
    const video = e.target;
    const frame = frames.get(video);
    if (frame == null) return;
    // The play and pause buttons trade places, so focus moves to the one that shows.
    const focused = frame.querySelector(`${PLAY}:focus, ${PAUSE}:focus`) != null;
    frame.classList.toggle(PLAYING_CLASS, !video.paused);
    if (focused) frame.querySelector<HTMLElement>(video.paused ? PLAY : PAUSE)?.focus();
    if (video.paused) playing.delete(video);
    else playing.add(video);
    if (paint === 0 && playing.size > 0) paint = requestAnimationFrame(drawProgress);
  };
  // Media events do not bubble.
  document.addEventListener("play", onPlayback, { capture: true });
  document.addEventListener("pause", onPlayback, { capture: true });

  // The play, pause, and replay buttons work on any device. On a touch device, a tap
  // anywhere on the video plays and pauses it too.
  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    const replay = e.target.closest(REPLAY) != null;
    const onToggle = replay || e.target.closest(`${PLAY}, ${PAUSE}`) != null;
    if (!onToggle && (!matches(TOUCH) || e.target.closest("button") != null)) return;
    const video = e.target.closest(".docs-video")?.querySelector("video");
    if (video == null) return;
    if (replay) video.currentTime = 0;
    if (replay || video.paused) {
      held.delete(video);
      video.play().catch(console.error);
    } else {
      held.add(video);
      video.pause();
    }
  });

  // A video picks its source once, so a themed video reloads when the scheme changes.
  window.matchMedia(DARK).addEventListener("change", () => {
    for (const video of document.querySelectorAll<HTMLVideoElement>("video"))
      if (video.querySelector("source[media]") != null) video.load();
  });

  document.addEventListener("astro:after-swap", () => {
    playing.clear();
    bind();
  });
  bind();
};
