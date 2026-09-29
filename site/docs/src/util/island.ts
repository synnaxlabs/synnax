// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Renders an open island into its element and returns a function that removes it. */
export type Mount = (el: HTMLElement) => () => void;

export interface OnDemandProps {
  /** Matches the elements that hold the island's server-rendered trigger. */
  selector: string;
  /** Imports the module that mounts the island. */
  load: () => Promise<{ mount: Mount }>;
  /** Reports whether a key press opens the first matching island. */
  hotkey?: (e: KeyboardEvent) => boolean;
}

// Astro installs React's refresh runtime only on pages with hydrated islands, and a
// dev build refuses to run React modules without it.
const DEV_PREAMBLE = "/@id/astro:scripts/before-hydration.js";

const importIsland = async (load: OnDemandProps["load"]): Promise<{ mount: Mount }> => {
  if (import.meta.env.DEV) await import(/* @vite-ignore */ DEV_PREAMBLE);
  return await load();
};

/**
 * Loads a React island only when a reader uses it: a click on its trigger or its
 * hotkey mounts it open, and hovering the trigger starts the download. Until then the
 * page ships no React. Call it once per island; it follows page swaps by itself.
 */
export const onDemand = ({ selector, load, hotkey }: OnDemandProps): void => {
  const mounted = new Map<HTMLElement, (() => void) | null>();

  const open = async (el: HTMLElement): Promise<void> => {
    mounted.set(el, null);
    let island: { mount: Mount } | null = null;
    try {
      island = await importIsland(load);
    } finally {
      // A failed download clears the island so the next click retries it.
      if (island == null) mounted.delete(el);
    }
    if (mounted.has(el)) mounted.set(el, island.mount(el));
  };

  document.addEventListener("pointerover", (e) => {
    if (e.target instanceof Element && e.target.closest(selector) != null)
      void importIsland(load);
  });

  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    const el = e.target.closest<HTMLElement>(selector);
    if (el == null || mounted.has(el)) return;
    e.preventDefault();
    void open(el);
  });

  if (hotkey != null)
    document.addEventListener("keydown", (e) => {
      const el = document.querySelector<HTMLElement>(selector);
      if (el == null || mounted.has(el) || !hotkey(e)) return;
      e.preventDefault();
      void open(el);
    });

  // A swap drops the islands outside persisted elements.
  document.addEventListener("astro:after-swap", () => {
    for (const [el, unmount] of mounted)
      if (!el.isConnected) {
        unmount?.();
        mounted.delete(el);
      }
  });
};
