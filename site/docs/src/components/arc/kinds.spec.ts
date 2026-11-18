// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { start } from "@/components/arc/kinds";

const kind = (id: string): HTMLDetailsElement => {
  const found = document.getElementById(id);
  if (!(found instanceof HTMLDetailsElement))
    throw new Error(`no kind has the id ${id}`);
  return found;
};

const click = (el: Element, init: MouseEventInit = {}): MouseEvent => {
  const event = new MouseEvent("click", { bubbles: true, cancelable: true, ...init });
  el.dispatchEvent(event);
  return event;
};

describe("Arc kinds", () => {
  let stop: () => void;

  beforeEach(() => {
    window.history.replaceState(null, "", "/errors");
    document.body.innerHTML = `
      <details class="arc-kind" id="e4004">
        <summary><span>E4004</span><a href="#e4004">undefined symbol</a></summary>
      </details>
      <details class="arc-kind" id="e4005"><summary></summary></details>
      <details id="other"><summary><a href="#other">Other</a></summary></details>
    `;
  });

  afterEach(() => stop());

  it("should open the kind that the URL hash names", () => {
    window.history.replaceState(null, "", "#e4004");
    stop = start();
    expect(kind("e4004").open).toBe(true);
    expect(kind("e4005").open).toBe(false);
  });

  it("should open the kind after a page swap", () => {
    stop = start();
    window.history.replaceState(null, "", "#e4005");
    document.dispatchEvent(new Event("astro:page-load"));
    expect(kind("e4005").open).toBe(true);
  });

  it("should open the kind when the hash changes", () => {
    stop = start();
    window.history.replaceState(null, "", "#e4005");
    window.dispatchEvent(new Event("hashchange"));
    expect(kind("e4005").open).toBe(true);
  });

  it("should leave other details closed", () => {
    window.history.replaceState(null, "", "#other");
    stop = start();
    expect(kind("other").open).toBe(false);
  });

  it("should open and close a kind when the reader clicks its title", () => {
    stop = start();
    const title = kind("e4004").querySelector("a") as HTMLAnchorElement;
    expect(click(title).defaultPrevented).toBe(true);
    expect(kind("e4004").open).toBe(true);
    click(title);
    expect(kind("e4004").open).toBe(false);
  });

  it("should leave a click with a modifier key to the browser", () => {
    stop = start();
    const title = kind("e4004").querySelector("a") as HTMLAnchorElement;
    expect(click(title, { metaKey: true }).defaultPrevented).toBe(false);
    expect(kind("e4004").open).toBe(false);
  });

  it("should leave the title of other details alone", () => {
    stop = start();
    const title = kind("other").querySelector("a") as HTMLAnchorElement;
    expect(click(title).defaultPrevented).toBe(false);
    expect(kind("other").open).toBe(false);
  });

  it("should write the kind that opens to the URL", () => {
    stop = start();
    kind("e4005").open = true;
    kind("e4005").dispatchEvent(new Event("toggle"));
    expect(window.location.hash).toEqual("#e4005");
  });

  it("should keep the URL when a kind closes", () => {
    window.history.replaceState(null, "", "#e4004");
    stop = start();
    kind("e4004").open = false;
    kind("e4004").dispatchEvent(new Event("toggle"));
    expect(window.location.hash).toEqual("#e4004");
  });
});
