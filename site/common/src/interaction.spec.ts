// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import interaction from "./interaction";

describe("interaction", () => {
  let island: HTMLElement;
  let button: HTMLButtonElement;
  let clicks: MouseEvent[];
  // Astro's hydrate leaves the island unhydrated after a failed download.
  let outcome: "hydrated" | "failed" | Error;
  const hydrate = vi.fn(async () => {
    if (outcome instanceof Error) throw outcome;
    if (outcome === "failed") return;
    island.removeAttribute("ssr");
    button.addEventListener("click", (e) => clicks.push(e));
  });
  const load = vi.fn(async () => hydrate);

  // A bare directive passes true, which Astro's types call a string.
  const start = (value: string | true): void =>
    Reflect.apply(interaction, undefined, [load, { name: "Search", value }, island]);

  const flush = async (): Promise<void> =>
    await new Promise((resolve) => setTimeout(resolve));

  const press = (key: string, metaKey: boolean): KeyboardEvent => {
    const e = new KeyboardEvent("keydown", { key, metaKey, cancelable: true });
    document.dispatchEvent(e);
    return e;
  };

  beforeEach(() => {
    island = document.createElement("astro-island");
    island.setAttribute("ssr", "");
    outcome = "hydrated";
    button = document.createElement("button");
    island.append(button);
    document.body.append(island);
    clicks = [];
    vi.clearAllMocks();
  });

  afterEach(() => document.body.replaceChildren());

  it("should start the download when the reader hovers the island", () => {
    start(true);
    button.dispatchEvent(new PointerEvent("pointerover", { bubbles: true }));
    expect(load).toHaveBeenCalledOnce();
    expect(hydrate).not.toHaveBeenCalled();
  });

  it("should hydrate on the first click and replay it", async () => {
    start(true);
    const click = new MouseEvent("click", { bubbles: true, cancelable: true });
    button.dispatchEvent(click);
    expect(click.defaultPrevented).toBe(true);
    await flush();
    expect(hydrate).toHaveBeenCalledOnce();
    expect(clicks).toHaveLength(1);
  });

  it("should leave later clicks to the hydrated island", async () => {
    start(true);
    button.click();
    await flush();
    button.click();
    await flush();
    expect(hydrate).toHaveBeenCalledOnce();
    expect(clicks).toHaveLength(2);
  });

  it("should click the island's button on its hotkey", async () => {
    start("K");
    expect(press("k", true).defaultPrevented).toBe(true);
    await flush();
    expect(hydrate).toHaveBeenCalledOnce();
    expect(clicks).toHaveLength(1);
  });

  it("should ignore the key without Ctrl or Cmd", async () => {
    start("k");
    expect(press("k", false).defaultPrevented).toBe(false);
    await flush();
    expect(load).not.toHaveBeenCalled();
  });

  it("should not listen for a hotkey after the island leaves the page", async () => {
    start("k");
    island.remove();
    expect(press("k", true).defaultPrevented).toBe(false);
    document.body.append(island);
    expect(press("k", true).defaultPrevented).toBe(false);
    await flush();
    expect(load).not.toHaveBeenCalled();
  });

  it("should do nothing when Astro starts it again on the same island", async () => {
    start(true);
    start(true);
    button.click();
    await flush();
    expect(hydrate).toHaveBeenCalledOnce();
    expect(clicks).toHaveLength(1);
    start(true);
    button.click();
    await flush();
    expect(hydrate).toHaveBeenCalledOnce();
    expect(clicks).toHaveLength(2);
  });

  it("should let the next click retry after a failed download", async () => {
    start(true);
    outcome = "failed";
    button.click();
    await flush();
    expect(clicks).toHaveLength(0);
    outcome = "hydrated";
    button.click();
    await flush();
    expect(hydrate).toHaveBeenCalledTimes(2);
    expect(clicks).toHaveLength(1);
  });

  it("should log a failed hydration and let the next click retry", async () => {
    const logged = vi.spyOn(console, "error").mockImplementation(() => {});
    start(true);
    outcome = new Error("offline");
    button.click();
    await flush();
    expect(logged).toHaveBeenCalledExactlyOnceWith(outcome);
    logged.mockRestore();
    outcome = "hydrated";
    button.click();
    await flush();
    expect(clicks).toHaveLength(1);
  });
});
