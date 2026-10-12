// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { enhanceMenu } from "@/components/menu";

const setup = () => {
  document.body.innerHTML = `
    <header class="site-header"><button data-mobile-open>Open menu</button></header>
    <nav class="mobile-drawer">
      <a href="/">Home</a>
      <button data-mobile-close>Close menu</button>
      <a href="#system">System</a>
      <a href="#contact">Show us your stack</a>
    </nav>
    <main><a href="#contact">Contact</a></main>
    <footer>Foundation by Synnax</footer>
  `;
  const drawer = document.querySelector<HTMLElement>(".mobile-drawer")!;
  const trigger = document.querySelector<HTMLButtonElement>("[data-mobile-open]")!;
  const close = document.querySelector<HTMLButtonElement>("[data-mobile-close]")!;
  const background = Array.from(
    document.querySelectorAll<HTMLElement>(".site-header, main, footer"),
  );
  const breakpoint = new EventTarget();
  vi.stubGlobal(
    "matchMedia",
    vi.fn(() => breakpoint),
  );

  // The shared drawer owns click handling; this fixture supplies its class changes.
  trigger.addEventListener("click", () => drawer.classList.add("mobile-drawer--open"));
  close.addEventListener("click", () => drawer.classList.remove("mobile-drawer--open"));
  enhanceMenu();

  const open = async (): Promise<void> => {
    trigger.focus();
    fireEvent.click(trigger);
    await waitFor(() => expect(drawer.inert).toBe(false));
  };
  return { drawer, trigger, close, background, breakpoint, open };
};

describe("enhanceMenu", () => {
  afterEach(() => {
    document.body.replaceChildren();
  });

  it("should keep the closed drawer inert and identify the controlled dialog", () => {
    const { drawer, trigger, background } = setup();
    expect(drawer.inert).toBe(true);
    expect(drawer.getAttribute("aria-hidden")).toBe("true");
    expect(drawer.getAttribute("role")).toBe("dialog");
    expect(drawer.getAttribute("aria-label")).toBe("Foundation navigation");
    expect(drawer.getAttribute("aria-modal")).toBe("true");
    expect(trigger.getAttribute("aria-expanded")).toBe("false");
    expect(trigger.getAttribute("aria-controls")).toBe(drawer.id);
    expect(background.every((element) => !element.inert)).toBe(true);
  });

  it("should focus the close control and make the background inert when opened", async () => {
    const { drawer, trigger, close, background, open } = setup();
    await open();
    expect(drawer.getAttribute("aria-hidden")).toBe("false");
    expect(trigger.getAttribute("aria-expanded")).toBe("true");
    expect(background.every((element) => element.inert)).toBe(true);
    expect(document.activeElement).toBe(close);
  });

  it("should close on Escape and restore focus and background interaction", async () => {
    const { drawer, trigger, background, open } = setup();
    await open();
    const lastLink = drawer.querySelectorAll("a")[2];
    lastLink.focus();
    expect(fireEvent.keyDown(lastLink, { key: "Escape" })).toBe(false);

    await waitFor(() => expect(drawer.inert).toBe(true));
    expect(trigger.getAttribute("aria-expanded")).toBe("false");
    expect(document.activeElement).toBe(trigger);
    expect(background.every((element) => !element.inert)).toBe(true);
  });

  it("should wrap Tab and Shift+Tab at the drawer boundaries", async () => {
    const { drawer, open } = setup();
    await open();
    const links = drawer.querySelectorAll("a");
    const first = links[0];
    const last = links[2];

    last.focus();
    expect(fireEvent.keyDown(last, { key: "Tab" })).toBe(false);
    expect(document.activeElement).toBe(first);

    expect(fireEvent.keyDown(first, { key: "Tab", shiftKey: true })).toBe(false);
    expect(document.activeElement).toBe(last);
  });

  it("should close an open drawer when switching to desktop navigation", async () => {
    const { drawer, trigger, background, breakpoint, open } = setup();
    await open();
    breakpoint.dispatchEvent(Object.assign(new Event("change"), { matches: true }));

    await waitFor(() => expect(drawer.inert).toBe(true));
    expect(document.activeElement).toBe(trigger);
    expect(background.every((element) => !element.inert)).toBe(true);
  });
});
