// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderToString } from "react-dom/server";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Nav } from "@/components/nav";
import { start } from "@/components/nav/sidebar";

const NODES: Nav.PageNavNode[] = [
  {
    key: "channels",
    name: "Channels",
    children: [
      { href: "/reference/channels", name: "Overview" },
      {
        href: "/reference/channels/create",
        name: "Create channels",
      },
    ],
  },
  {
    key: "control",
    name: "Control",
    children: [
      {
        name: "Arc",
        href: "/reference/control/arc",
        children: [
          {
            href: "/reference/control/arc/get-started",
            name: "Get started",
          },
          {
            key: "concepts",
            name: "Concepts",
            children: [
              {
                href: "/reference/control/arc/concepts/stages",
                name: "Stages",
              },
            ],
          },
        ],
      },
    ],
  },
];

const show = (path: string, nodes: Nav.PageNavNode[] = NODES): void => {
  window.history.replaceState(null, "", path);
  document.body.innerHTML = renderToString(
    <Nav.Page nodes={nodes} currentPage={path} />,
  );
};

const item = (name: string): HTMLElement => {
  const found = [...document.querySelectorAll<HTMLElement>('[role="treeitem"]')].find(
    (el) => el.textContent === name,
  );
  if (found == null) throw new Error(`no item named ${name}`);
  return found;
};

const visible = (): string[] =>
  [...document.querySelectorAll<HTMLElement>('[role="treeitem"]')]
    .filter((el) => !el.hidden)
    .map((el) => el.textContent ?? "");

const selected = (): string[] =>
  [...document.querySelectorAll<HTMLElement>('[aria-selected="true"]')].map(
    (el) => el.textContent ?? "",
  );

const caret = (name: string): Element => {
  const found = item(name).querySelector(".lyra-tree__expansion-indicator");
  if (found == null) throw new Error(`no caret on ${name}`);
  return found;
};

const click = (el: Element): MouseEvent => {
  const event = new MouseEvent("click", { bubbles: true, cancelable: true });
  el.dispatchEvent(event);
  return event;
};

const swap = (path: string): void => {
  window.history.replaceState(null, "", path);
  document.dispatchEvent(new Event("astro:after-swap"));
};

describe("sidebar", () => {
  beforeAll(() => start());

  describe("server render", () => {
    it("should expand the sections and the current page's ancestors", () => {
      show("/reference/control/arc/concepts/stages/");
      expect(visible()).toEqual([
        "Channels",
        "Overview",
        "Create channels",
        "Control",
        "Arc",
        "Get started",
        "Concepts",
        "Stages",
      ]);
      expect(selected()).toEqual(["Stages"]);
    });

    it("should collapse nodes off the current page's path", () => {
      show("/reference/channels/create");
      expect(visible()).toEqual([
        "Channels",
        "Overview",
        "Create channels",
        "Control",
        "Arc",
      ]);
      expect(item("Arc").getAttribute("aria-expanded")).toBe("false");
      expect(selected()).toEqual(["Create channels"]);
    });

    it("should expand a group whose key is not part of the page's path", () => {
      show("/reference/enterprise/certificates", [
        {
          key: "enterprise",
          name: "Enterprise",
          children: [
            {
              key: "tls",
              name: "TLS",
              children: [
                { href: "/reference/enterprise/certificates", name: "Certificates" },
              ],
            },
          ],
        },
      ]);
      expect(visible()).toEqual(["Enterprise", "TLS", "Certificates"]);
    });

    it("should collapse a group whose key matches another part of the path", () => {
      show("/reference/control/arc", [
        {
          key: "control",
          name: "Control",
          children: [
            { href: "/reference/control/arc", name: "Arc" },
            {
              key: "reference",
              name: "Language reference",
              children: [
                { href: "/reference/control/arc/reference/syntax", name: "Syntax" },
              ],
            },
          ],
        },
      ]);
      expect(visible()).toEqual(["Control", "Arc", "Language reference"]);
    });
  });

  describe("clicks", () => {
    it("should expand a node when its caret is clicked", () => {
      show("/reference/channels");
      click(caret("Arc"));
      expect(item("Arc").getAttribute("aria-expanded")).toBe("true");
      expect(caret("Arc").classList).toContain("lyra--location-bottom");
      expect(visible()).toContain("Get started");
      expect(visible()).not.toContain("Stages");
    });

    it("should hide every descendant when a node collapses", () => {
      show("/reference/control/arc/concepts/stages");
      click(caret("Arc"));
      expect(caret("Arc").classList).toContain("lyra--location-right");
      expect(visible()).toEqual([
        "Channels",
        "Overview",
        "Create channels",
        "Control",
        "Arc",
      ]);
    });

    it("should toggle a node without a page when its name is clicked", () => {
      show("/reference/control/arc/get-started");
      item("Concepts").click();
      expect(visible()).toContain("Stages");
      item("Concepts").click();
      expect(visible()).not.toContain("Stages");
    });

    it("should let a node with a page navigate instead of toggling", () => {
      show("/reference/channels");
      const name = item("Arc").querySelector("p");
      if (name == null) throw new Error("no name on Arc");
      expect(click(name).defaultPrevented).toBe(false);
      expect(item("Arc").getAttribute("aria-expanded")).toBe("false");
    });

    it("should keep sections open", () => {
      show("/reference/channels");
      click(caret("Channels"));
      expect(visible()).toContain("Overview");
    });
  });

  describe("page swaps", () => {
    it("should select the new page and expand its ancestors", () => {
      show("/reference/channels");
      swap("/reference/control/arc/concepts/stages/");
      expect(selected()).toEqual(["Stages"]);
      expect(visible()).toContain("Stages");
    });

    it("should expand the new page when it has children", () => {
      show("/reference/channels");
      swap("/reference/control/arc");
      expect(selected()).toEqual(["Arc"]);
      expect(visible()).toContain("Get started");
      expect(visible()).not.toContain("Stages");
    });

    it("should clear the selection on a page outside the tree", () => {
      show("/reference/channels");
      swap("/reference/unknown");
      expect(selected()).toEqual([]);
    });
  });

  describe("scroll", () => {
    const place = (el: Element, top: number, height: number): void => {
      el.getBoundingClientRect = () =>
        ({ top, bottom: top + height, height }) as DOMRect;
    };

    const scrollable = (): (() => number) => {
      const tree = document.querySelector(".reference-tree");
      if (tree == null) throw new Error("no tree");
      let scrollTop = 0;
      Object.defineProperty(tree, "scrollTop", {
        get: () => scrollTop,
        set: (value: number) => {
          scrollTop = value;
        },
      });
      place(tree, 0, 100);
      return () => scrollTop;
    };

    it("should center the new page when it is out of view", () => {
      show("/reference/channels");
      const scrollTop = scrollable();
      place(item("Stages"), 300, 20);
      swap("/reference/control/arc/concepts/stages");
      expect(scrollTop()).toBe(260);
    });

    it("should not scroll when the new page is in view", () => {
      show("/reference/channels");
      const scrollTop = scrollable();
      place(item("Create channels"), 40, 20);
      swap("/reference/channels/create");
      expect(scrollTop()).toBe(0);
    });
  });

  // Last, because start adds a second set of document listeners.
  it("should start when the browser blocks storage", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new DOMException("blocked", "SecurityError");
    });
    document.body.innerHTML = `<nav class="page-nav">${renderToString(
      <Nav.Page nodes={NODES} currentPage="/reference/channels" />,
    )}</nav>`;
    expect(() => start()).not.toThrow();
  });
});
