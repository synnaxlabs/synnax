// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

import { start } from "@/components/header";

const header = (): Element => document.querySelector(".header")!;

const scrollTo = (y: number): void => {
  vi.stubGlobal("scrollY", y);
  window.dispatchEvent(new Event("scroll"));
};

describe("header", () => {
  beforeEach(() => {
    document.body.innerHTML = '<header class="header"></header>';
  });

  it("should mark the header when the page loads at the top", () => {
    vi.stubGlobal("scrollY", 0);
    start();
    expect(header().classList).toContain("header--top");
  });

  it("should not mark the header when the page loads scrolled down", () => {
    vi.stubGlobal("scrollY", 400);
    start();
    expect(header().classList).not.toContain("header--top");
  });

  it("should unmark the header when the page scrolls past 10 pixels", () => {
    vi.stubGlobal("scrollY", 0);
    start();
    scrollTo(10);
    expect(header().classList).toContain("header--top");
    scrollTo(11);
    expect(header().classList).not.toContain("header--top");
  });

  it("should mark the header again when the page scrolls back to the top", () => {
    vi.stubGlobal("scrollY", 400);
    start();
    scrollTo(5);
    expect(header().classList).toContain("header--top");
  });
});
