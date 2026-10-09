// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Run decorative animation only when its section and the document are visible. */
export const observeMotion = (): (() => void) => {
  const elements = Array.from(
    document.querySelectorAll<HTMLElement | SVGElement>("[data-viewport-motion]"),
  );
  const visible = new Set<Element>();
  const sync = (): void => {
    for (const element of elements)
      element.dataset.motion =
        visible.has(element) && !document.hidden ? "running" : "paused";
  };
  const observer =
    typeof IntersectionObserver === "undefined"
      ? undefined
      : new IntersectionObserver((entries) => {
          for (const entry of entries)
            if (entry.isIntersecting) visible.add(entry.target);
            else visible.delete(entry.target);
          sync();
        });
  for (const element of elements)
    if (observer) observer.observe(element);
    else visible.add(element);
  document.addEventListener("visibilitychange", sync);
  sync();
  return () => {
    observer?.disconnect();
    document.removeEventListener("visibilitychange", sync);
  };
};
