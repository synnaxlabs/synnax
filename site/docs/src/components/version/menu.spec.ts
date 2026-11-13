// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { start } from "@/components/version/menu";

const menu = (): HTMLDetailsElement => {
  const details = document.createElement("details");
  details.className = "menu";
  details.innerHTML = '<summary>0.60</summary><a data-origin="https://x">0.59</a>';
  details.open = true;
  document.body.append(details);
  return details;
};

describe("start", () => {
  beforeAll(() => start(".menu"));
  beforeEach(() => document.body.replaceChildren());

  it("should close a menu on a click outside it", () => {
    const details = menu();
    document.body.click();
    expect(details.open).toBe(false);
  });

  it("should keep a menu open on a click inside it", () => {
    const details = menu();
    details.querySelector("a")?.addEventListener("click", (e) => e.preventDefault());
    details.querySelector("a")?.click();
    expect(details.open).toBe(true);
  });

  it("should close a menu on Escape", () => {
    const details = menu();
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
    expect(details.open).toBe(false);
  });

  it("should close a menu on a page load", () => {
    const details = menu();
    document.dispatchEvent(new Event("astro:page-load"));
    expect(details.open).toBe(false);
  });
});
