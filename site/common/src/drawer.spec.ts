// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, describe, expect, it } from "vitest";

import { start } from "./drawer";

// Mirrors the markup of MenuButton.astro and Drawer.astro.
const HEADER = `
  <button class="mobile-menu-btn" data-mobile-open><span></span></button>
  <div class="mobile-overlay" data-mobile-close></div>
  <nav class="mobile-drawer">
    <button class="mobile-close-btn" data-mobile-close></button>
    <a href="/reference/">Reference</a>
    <span class="label">Reference</span>
  </nav>
`;

const click = (selector: string): void =>
  (document.querySelector(selector) as HTMLElement).click();

const open = (): boolean => document.body.classList.contains("mobile-menu-open");

describe("mobile menu", () => {
  beforeAll(() => start());

  const show = (): void => {
    document.body.className = "";
    document.body.innerHTML = HEADER;
    click(".mobile-menu-btn span");
  };

  it("should open the drawer and overlay from the menu button", () => {
    show();
    expect(open()).toBe(true);
    expect(document.querySelector(".mobile-drawer")?.classList).toContain(
      "mobile-drawer--open",
    );
    expect(document.querySelector(".mobile-overlay")?.classList).toContain(
      "mobile-overlay--open",
    );
  });

  it("should close from the close button", () => {
    show();
    click(".mobile-close-btn");
    expect(open()).toBe(false);
  });

  it("should close from the overlay", () => {
    show();
    click(".mobile-overlay");
    expect(open()).toBe(false);
  });

  it("should close when a link in the drawer is followed", () => {
    show();
    document.querySelector("a")?.addEventListener("click", (e) => e.preventDefault());
    click(".mobile-drawer a");
    expect(open()).toBe(false);
  });

  it("should stay open on a click inside the drawer that is not a link", () => {
    show();
    click(".mobile-drawer .label");
    expect(open()).toBe(true);
  });

  it("should close after a page swap", () => {
    show();
    document.dispatchEvent(new Event("astro:after-swap"));
    expect(open()).toBe(false);
  });
});
