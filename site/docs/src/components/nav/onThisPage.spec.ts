// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, describe, expect, it } from "vitest";

import { start } from "@/components/nav/onThisPage";
import { CHANGE_EVENT } from "@/components/tabs/sync";

// Mirrors the markup of the article and of OnThisPage.astro.
const PAGE = `
  <article>
    <h2 id="create">Create</h2>
    <div role="tabpanel"><h3 id="python">Python</h3></div>
    <div role="tabpanel" hidden><h3 id="typescript">TypeScript</h3></div>
    <h2 id="delete">Delete</h2>
  </article>
  <div class="on-this-page-menu" data-outline="article :is(h2, h3)">
    <div class="on-this-page-indicator" data-outline-indicator></div>
    <a href="#create" data-item-key="create">Create</a>
    <a href="#python" data-item-key="python">Python</a>
    <a href="#typescript" data-item-key="typescript">TypeScript</a>
    <a href="#delete" data-item-key="delete">Delete</a>
  </div>
`;

// Places each heading at a viewport offset; a heading left out sits far below.
const scrollTo = (tops: Record<string, number>): void => {
  for (const heading of document.querySelectorAll("article :is(h2, h3)"))
    heading.getBoundingClientRect = () =>
      ({ top: tops[heading.id] ?? 2000, height: 30 }) as DOMRect;
};

const show = (tops: Record<string, number> = {}): void => {
  document.body.innerHTML = PAGE;
  scrollTo(tops);
  document.dispatchEvent(new Event("astro:after-swap"));
};

const shown = (): string[] =>
  [...document.querySelectorAll<HTMLElement>("[data-item-key]")]
    .filter((item) => !item.hidden)
    .map((item) => item.dataset.itemKey ?? "");

const active = (): string[] =>
  [...document.querySelectorAll<HTMLElement>("[data-item-key].active")].map(
    (item) => item.dataset.itemKey ?? "",
  );

const indicator = (): HTMLElement =>
  document.querySelector(".on-this-page-indicator") as HTMLElement;

describe("on this page", () => {
  beforeAll(() => start());

  describe("tabbed headings", () => {
    it("should hide entries for headings in hidden tab panels", () => {
      show();
      expect(shown()).toEqual(["create", "python", "delete"]);
    });

    it("should follow a tab change", () => {
      show();
      const [python, typescript] =
        document.querySelectorAll<HTMLElement>('[role="tabpanel"]');
      python.hidden = true;
      typescript.hidden = false;
      document.dispatchEvent(new CustomEvent(CHANGE_EVENT));
      expect(shown()).toEqual(["create", "typescript", "delete"]);
    });
  });

  describe("reading position", () => {
    it("should mark nothing before the first heading reaches the top", () => {
      show();
      expect(active()).toEqual([]);
      expect(indicator().style.opacity).toBe("");
    });

    it("should mark the heading closest to the reading line", () => {
      show({ create: -400, python: 100, delete: 600 });
      expect(active()).toEqual(["python"]);
      expect(indicator().style.opacity).toBe("1");
    });

    it("should mark the last heading passed when none is near the top", () => {
      show({ create: -900, python: -400, delete: 800 });
      expect(active()).toEqual(["python"]);
    });

    it("should skip headings in hidden tab panels", () => {
      // A browser places a heading in a hidden panel at the top of the viewport.
      show({ create: -900, python: -400, typescript: 0, delete: 800 });
      expect(active()).toEqual(["python"]);
    });

    it("should follow the scroll", () => {
      show({ create: 100, python: 400 });
      scrollTo({ create: -300, python: 110 });
      window.dispatchEvent(new Event("scroll"));
      expect(active()).toEqual(["python"]);
    });

    it("should mark an entry when it is clicked", () => {
      show({ create: 100 });
      (document.querySelector('[data-item-key="delete"]') as HTMLElement).click();
      expect(active()).toEqual(["delete"]);
    });
  });

  it("should follow each outline menu with its own headings", () => {
    document.body.innerHTML = `
      <article><h2 id="intro">Intro</h2><h6 id="v1">v1</h6></article>
      <div data-outline="article h6"><a href="#v1" data-item-key="v1">v1</a></div>
    `;
    scrollTo({ intro: 100 });
    for (const heading of document.querySelectorAll("h6"))
      heading.getBoundingClientRect = () => ({ top: 110, height: 30 }) as DOMRect;
    document.dispatchEvent(new Event("astro:after-swap"));
    expect(active()).toEqual(["v1"]);
  });

  it("should stop listening to a page after it swaps out", () => {
    show();
    const stale = document.querySelector<HTMLElement>('[data-item-key="create"]');
    show();
    scrollTo({ create: 100 });
    window.dispatchEvent(new Event("scroll"));
    expect(stale?.classList).not.toContain("active");
  });
});
