// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { onDemand } from "@/util/island";

const unmount = vi.fn();
const mount = vi.fn((_: HTMLElement) => unmount);
let failure: Error | null = null;
const load = vi.fn(async () => {
  if (failure != null) throw failure;
  return { mount };
});

const trigger = (): HTMLElement =>
  document.querySelector('[data-island="search"]') as HTMLElement;

const show = (): void => {
  document.body.innerHTML = '<div data-island="search"><button>Search</button></div>';
};

const press = (key: string, metaKey = false): KeyboardEvent => {
  const event = new KeyboardEvent("keydown", { key, metaKey, cancelable: true });
  document.dispatchEvent(event);
  return event;
};

const flush = async (): Promise<void> => {
  await new Promise((resolve) => setTimeout(resolve));
};

describe("onDemand", () => {
  beforeAll(() => {
    vi.stubEnv("DEV", false);
    onDemand({
      selector: '[data-island="search"]',
      load,
      hotkey: (e) => e.metaKey && e.key === "k",
    });
  });

  beforeEach(() => {
    show();
    failure = null;
    vi.clearAllMocks();
  });

  // Drops the islands each spec mounted.
  afterEach(() => {
    document.body.innerHTML = "";
    document.dispatchEvent(new Event("astro:after-swap"));
  });

  it("should start the download when the reader hovers the trigger", () => {
    trigger()
      .querySelector("button")
      ?.dispatchEvent(new PointerEvent("pointerover", { bubbles: true }));
    expect(load).toHaveBeenCalled();
    expect(mount).not.toHaveBeenCalled();
  });

  it("should mount the island into its trigger on a click", async () => {
    const click = new MouseEvent("click", { bubbles: true, cancelable: true });
    trigger().querySelector("button")?.dispatchEvent(click);
    await flush();
    expect(click.defaultPrevented).toBe(true);
    expect(mount).toHaveBeenCalledExactlyOnceWith(trigger());
  });

  it("should leave later clicks to the mounted island", async () => {
    trigger().click();
    await flush();
    const click = new MouseEvent("click", { bubbles: true, cancelable: true });
    trigger().dispatchEvent(click);
    await flush();
    expect(click.defaultPrevented).toBe(false);
    expect(mount).toHaveBeenCalledOnce();
  });

  it("should mount the island on its hotkey", async () => {
    expect(press("k", true).defaultPrevented).toBe(true);
    await flush();
    expect(mount).toHaveBeenCalledExactlyOnceWith(trigger());
  });

  it("should ignore other keys", async () => {
    expect(press("k").defaultPrevented).toBe(false);
    await flush();
    expect(mount).not.toHaveBeenCalled();
  });

  it("should unmount the island when a swap removes its trigger", async () => {
    trigger().click();
    await flush();
    show();
    document.dispatchEvent(new Event("astro:after-swap"));
    expect(unmount).toHaveBeenCalledOnce();
  });

  it("should keep an island whose trigger persists across a swap", async () => {
    trigger().click();
    await flush();
    document.dispatchEvent(new Event("astro:after-swap"));
    expect(unmount).not.toHaveBeenCalled();
  });

  it("should not mount an island whose trigger left before the download finished", async () => {
    trigger().click();
    show();
    document.dispatchEvent(new Event("astro:after-swap"));
    await flush();
    expect(mount).not.toHaveBeenCalled();
  });

  it("should let the next click retry after a failed download", async () => {
    failure = new Error("offline");
    const rejections: unknown[] = [];
    const onRejection = (e: unknown): void => {
      rejections.push(e);
    };
    process.on("unhandledRejection", onRejection);
    trigger().click();
    await flush();
    process.off("unhandledRejection", onRejection);
    expect(rejections).toEqual([failure]);
    failure = null;
    trigger().click();
    await flush();
    expect(mount).toHaveBeenCalledOnce();
  });
});
