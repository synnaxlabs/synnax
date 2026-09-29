// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const TOUCH = "(hover: none) and (pointer: coarse)";
const DARK = "(prefers-color-scheme: dark)";
const PLAYING_CLASS = "docs-video--playing";
// Most of a video must be in view before it plays.
const VISIBLE = 0.85;

/**
 * Starts playing docs videos while they are in view. A touch device plays and pauses
 * on a tap instead, to spare the reader's data. Call it once; it follows page swaps by
 * itself.
 */
export const start = (): void => {
  const touch = (): boolean => window.matchMedia(TOUCH).matches;
  let observer: IntersectionObserver | null = null;

  const bind = (): void => {
    observer?.disconnect();
    observer = new IntersectionObserver(
      (entries) => {
        for (const { target, isIntersecting } of entries) {
          if (!(target instanceof HTMLVideoElement)) continue;
          if (!isIntersecting) target.pause();
          else if (!touch()) void target.play();
        }
      },
      { threshold: VISIBLE },
    );
    for (const video of document.querySelectorAll(".docs-video > video"))
      observer.observe(video);
  };

  const onPlayback = (e: Event): void => {
    if (!(e.target instanceof HTMLVideoElement)) return;
    e.target.parentElement?.classList.toggle(PLAYING_CLASS, !e.target.paused);
  };
  // Media events do not bubble.
  document.addEventListener("play", onPlayback, { capture: true });
  document.addEventListener("pause", onPlayback, { capture: true });

  document.addEventListener("click", (e) => {
    if (!touch() || !(e.target instanceof Element)) return;
    const video = e.target.closest(".docs-video")?.querySelector("video");
    if (video == null) return;
    if (video.paused) void video.play();
    else video.pause();
  });

  // A video picks its source once, so a themed video reloads when the scheme changes.
  window.matchMedia(DARK).addEventListener("change", () => {
    for (const video of document.querySelectorAll<HTMLVideoElement>(
      ".docs-video > video",
    ))
      if (video.querySelector("source[media]") != null) video.load();
  });

  document.addEventListener("astro:after-swap", bind);
  bind();
};
