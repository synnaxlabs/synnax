// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ClientDirective } from "astro";

const STARTED = "data-interaction";

/**
 * Hydrates an island on the first click inside it, then replays the click on the
 * hydrated island. Hovering the island starts the download. A string value is a key
 * that, with Ctrl or Cmd, clicks the island's first button. After a failed download,
 * the next click tries again.
 */
const interaction: ClientDirective = (load, options, el) => {
  // Astro runs the directive again when a page swap moves a persisted island.
  if (el.hasAttribute(STARTED)) return;
  el.setAttribute(STARTED, "");
  const { value }: { value: unknown } = options;
  const hotkey = typeof value === "string" ? value.toLowerCase() : null;

  const preload = (): void => void load();

  const open = async (target: Element): Promise<void> => {
    const hydrate = await load();
    await hydrate();
    // Astro logs a failed download and leaves the island as the server rendered it.
    if (el.hasAttribute("ssr")) return listen();
    target.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
  };

  const onClick = (e: MouseEvent): void => {
    if (!(e.target instanceof Element)) return;
    e.preventDefault();
    stop();
    open(e.target).catch((err: unknown) => {
      console.error(err);
      listen();
    });
  };

  const onKeyDown = (e: KeyboardEvent): void => {
    // A page swap can drop the island.
    if (!el.isConnected) return document.removeEventListener("keydown", onKeyDown);
    if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== hotkey) return;
    const button = el.querySelector("button");
    if (button == null) throw new Error("the island has no button to click");
    e.preventDefault();
    button.click();
  };

  const listen = (): void => {
    el.addEventListener("pointerover", preload, { once: true });
    el.addEventListener("click", onClick);
    if (hotkey != null) document.addEventListener("keydown", onKeyDown);
  };

  const stop = (): void => {
    el.removeEventListener("pointerover", preload);
    el.removeEventListener("click", onClick);
    document.removeEventListener("keydown", onKeyDown);
  };

  listen();
};

export default interaction;
