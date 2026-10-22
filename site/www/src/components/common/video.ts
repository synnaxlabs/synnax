// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const VIDEO = "video[data-lazy-video]";

/**
 * Starts the download of each lazy video when it comes within one viewport of the
 * screen, then plays it. Call it once per page.
 */
export const start = (): void => {
  const observer = new IntersectionObserver(
    (entries) => {
      for (const { target, isIntersecting } of entries) {
        if (!isIntersecting || !(target instanceof HTMLVideoElement)) continue;
        observer.unobserve(target);
        target.autoplay = true;
        target.load();
      }
    },
    { rootMargin: "100% 0px" },
  );
  for (const video of document.querySelectorAll(VIDEO)) observer.observe(video);
};
