// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { renderToString } from "react-dom/server";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { Picker } from "@/components/tabs/Picker";
import { CHANGE_EVENT, type Query, start } from "@/components/tabs/sync";
import { Tabs } from "@/components/tabs/Tabs";

const TABS = [
  { tabKey: "python", name: "Python" },
  { tabKey: "typescript", name: "TypeScript" },
];

const OPTIONS = TABS.map(({ tabKey, name }) => ({ key: tabKey, name, icon: <i /> }));

const panels = <p>py</p>;

let stored: string | null;
const changes = vi.fn();
const client: Query = { initial: () => stored, onChange: changes };

// The page arrives as server HTML, and the tabs script re-reads it after each swap.
const show = (page: ReactElement): void => {
  document.body.innerHTML = renderToString(page);
  document.dispatchEvent(new Event("astro:after-swap"));
};

const selected = (frame: Element): (string | undefined)[] =>
  [...frame.querySelectorAll<HTMLElement>('[role="tab"][aria-selected="true"]')].map(
    (tab) => tab.dataset.tabKey,
  );

const frames = (): HTMLElement[] => [
  ...document.querySelectorAll<HTMLElement>(".pluto-tabs"),
];

const tab = (frame: Element, key: string): HTMLElement =>
  frame.querySelector(`[data-tab-key="${key}"]`) as HTMLElement;

const visiblePanels = (frame: Element): string[] =>
  [...frame.querySelectorAll<HTMLElement>('[role="tabpanel"]')]
    .filter((panel) => !panel.hidden)
    .map((panel) => panel.textContent ?? "");

describe("tabs sync", () => {
  beforeAll(() => start({ client, context: {} }));

  beforeEach(() => {
    stored = null;
    changes.mockClear();
    window.history.replaceState(null, "", "/reference/page");
  });

  describe("on load", () => {
    it("should show the first tab when nothing picks another", () => {
      show(<Tabs tabs={TABS} queryParamKey="client" python={panels} typescript="ts" />);
      expect(selected(frames()[0])).toEqual(["python"]);
      expect(visiblePanels(frames()[0])).toEqual(["py"]);
      expect(window.location.search).toBe("");
    });

    it("should show the choice the query remembers and put it in the URL", () => {
      stored = "typescript";
      show(<Tabs tabs={TABS} queryParamKey="client" python={panels} typescript="ts" />);
      expect(selected(frames()[0])).toEqual(["typescript"]);
      expect(visiblePanels(frames()[0])).toEqual(["ts"]);
      expect(window.location.search).toBe("?client=typescript");
      expect(changes).toHaveBeenCalledWith("typescript");
    });

    it("should prefer the URL over the remembered choice", () => {
      stored = "python";
      window.history.replaceState(null, "", "/reference/page?client=typescript");
      show(<Tabs tabs={TABS} queryParamKey="client" python={panels} typescript="ts" />);
      expect(selected(frames()[0])).toEqual(["typescript"]);
    });

    it("should announce the change so the outline can follow", () => {
      const listener = vi.fn();
      document.addEventListener(CHANGE_EVENT, listener);
      show(<Tabs tabs={TABS} queryParamKey="client" python={panels} typescript="ts" />);
      document.removeEventListener(CHANGE_EVENT, listener);
      expect(listener).toHaveBeenCalled();
    });
  });

  describe("on click", () => {
    it("should select the tab in every block sharing the key", () => {
      show(
        <>
          <Tabs tabs={TABS} queryParamKey="client" python="a" typescript="b" />
          <Tabs tabs={TABS} queryParamKey="client" python="c" typescript="d" />
        </>,
      );
      tab(frames()[1], "typescript").click();
      expect(frames().map(selected)).toEqual([["typescript"], ["typescript"]]);
      expect(frames().map(visiblePanels)).toEqual([["b"], ["d"]]);
      expect(changes).toHaveBeenCalledWith("typescript");
    });

    it("should put the choice in the URL without a new history entry", () => {
      show(<Tabs tabs={TABS} queryParamKey="client" python="a" typescript="b" />);
      const length = window.history.length;
      tab(frames()[0], "typescript").click();
      expect(window.location.search).toBe("?client=typescript");
      expect(window.history.length).toBe(length);
    });

    it("should leave a block without the chosen tab on its own tab", () => {
      show(
        <>
          <Tabs tabs={TABS} queryParamKey="client" python="a" typescript="b" />
          <Tabs tabs={TABS.slice(0, 1)} queryParamKey="client" python="c" />
        </>,
      );
      tab(frames()[0], "typescript").click();
      expect(frames().map(selected)).toEqual([["typescript"], ["python"]]);
    });

    it("should switch a block without a key alone", () => {
      show(
        <>
          <Tabs tabs={TABS} python="a" typescript="b" />
          <Tabs tabs={TABS} python="c" typescript="d" />
        </>,
      );
      tab(frames()[0], "typescript").click();
      expect(frames().map(selected)).toEqual([["typescript"], ["python"]]);
      expect(window.location.search).toBe("");
    });

    it("should keep the clicked block in place while blocks above it resize", () => {
      show(<Tabs tabs={TABS} queryParamKey="client" python="a" typescript="b" />);
      const frame = frames()[0];
      const tops = [100, 160];
      vi.spyOn(frame, "getBoundingClientRect").mockImplementation(
        () => ({ top: tops.shift() ?? 160 }) as DOMRect,
      );
      const scrollBy = vi.fn();
      vi.stubGlobal("scrollBy", scrollBy);
      tab(frame, "typescript").click();
      vi.unstubAllGlobals();
      expect(scrollBy).toHaveBeenCalledWith(0, 60);
    });
  });

  describe("on key press", () => {
    it("should select the tab under focus on Enter", () => {
      show(<Tabs tabs={TABS} queryParamKey="client" python="a" typescript="b" />);
      const target = tab(frames()[0], "typescript");
      target.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Enter", bubbles: true }),
      );
      expect(selected(frames()[0])).toEqual(["typescript"]);
    });

    it("should move to the next tab with an arrow key, wrapping at the end", () => {
      show(<Tabs tabs={TABS} queryParamKey="client" python="a" typescript="b" />);
      const target = tab(frames()[0], "python");
      target.dispatchEvent(
        new KeyboardEvent("keydown", { key: "ArrowLeft", bubbles: true }),
      );
      expect(selected(frames()[0])).toEqual(["typescript"]);
      expect(document.activeElement).toBe(tab(frames()[0], "typescript"));
    });
  });

  describe("pickers", () => {
    it("should show the choice of the blocks it shares a key with", () => {
      stored = "typescript";
      show(
        <>
          <Picker query="client" options={OPTIONS} />
          <Tabs tabs={TABS} queryParamKey="client" python="a" typescript="b" />
        </>,
      );
      const value = document.querySelector("[data-tabs-value]");
      expect(value?.textContent).toBe("TypeScript");
    });

    it("should select the chosen option in every block", () => {
      show(
        <>
          <Picker query="client" options={OPTIONS} />
          <Tabs tabs={TABS} queryParamKey="client" python="a" typescript="b" />
        </>,
      );
      const hide = vi.fn();
      const menu = document.querySelector<HTMLElement>("[popover]");
      if (menu != null) menu.hidePopover = hide;
      (
        document.querySelector(
          '[role="option"][data-tab-key="typescript"]',
        ) as HTMLElement
      ).click();
      expect(selected(frames()[0])).toEqual(["typescript"]);
      expect(document.querySelector("[data-tabs-value]")?.textContent).toBe(
        "TypeScript",
      );
      expect(hide).toHaveBeenCalled();
    });
  });

  // Runs last: the failed start leaves its listeners on the document.
  it("should throw for an element whose key has no query", () => {
    document.body.innerHTML = renderToString(
      <Tabs tabs={TABS} queryParamKey="platform" python="a" typescript="b" />,
    );
    expect(() => start({})).toThrow("no tab query for key platform");
  });
});
