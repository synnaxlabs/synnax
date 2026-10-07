// Copyright 2026 Synnax Labs, Inc. Licensed under licenses/BSL.txt.

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
