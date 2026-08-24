// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Drift } from "@synnaxlabs/drift";
import { Text, Triggers as PTriggers } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Triggers } from "@/app/triggers";
import { useSelectorVisible } from "@/app/vis/Selector";
import { Panel as PlatformPanel } from "@/platform/panel";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { Selector } from "@/platform/selector";
import { Session } from "@/session";
import { Modals } from "@/session/modals";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const client = createTestClient();

const PANEL = uuid.create();
const TAB = uuid.create();
const TAB_NAME = "tab name";

const CONTROL = "ControlLeft";

const hold = (...codes: string[]): void =>
  codes.forEach((code) => fireEvent.keyDown(window, { code }));

const release = (...codes: string[]): void =>
  [...codes].reverse().forEach((code) => fireEvent.keyUp(window, { code }));

const press = (...codes: string[]): void => {
  hold(...codes);
  release(...codes);
};

const pressEscape = () => {
  fireEvent.keyDown(window, { key: "Escape", code: "Escape" });
  fireEvent.keyUp(window, { key: "Escape", code: "Escape" });
};

/** Makes tabKey the focused tab of key, and key the window's selected panel. */
const focusTab = (
  store: TestStore,
  key: panel.Key = PANEL,
  tabKey: panel.TabKey = TAB,
): void =>
  void act(() => {
    store.dispatch(
      Session.Panel.internalSelectTab({ key, tabKey, otherTabKeys: [tabKey] }),
    );
    store.dispatch(Session.Panel.select({ key }));
  });

const isWindowOpen = (store: TestStore): boolean =>
  Drift.selectWindow(store.getState()) != null;

// ui renders alongside the trigger mount, for shortcuts that act on the DOM.
const renderTriggers = async (ui?: ReactNode) => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client: null });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <PTriggers.Provider>
        {children}
        {ui}
      </PTriggers.Provider>
    </Console>
  );
  Wrapper.displayName = "Wrapper";
  const { result } = renderHook(
    () => {
      Triggers.use();
      return {
        overlaid: Session.Panel.useSelectOverlaid(),
        modals: Modals.useStore("app/triggers.spec"),
      };
    },
    { wrapper: Wrapper },
  );
  return { result, store };
};

// The cluster-writing shortcuts need a panel that exists on the Core. The triggers
// mount above every panel scope, the way the app shell mounts them.
const renderLiveTriggers = async (panelKey: panel.Key) => {
  const { wrapper: Panels, store } = await createPanelWrapper({ client });
  await primePanel(Panels, panelKey);
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Panels>
      <PTriggers.Provider>{children}</PTriggers.Provider>
    </Panels>
  );
  Wrapper.displayName = "LiveWrapper";
  const { result } = renderHook(
    () => {
      Triggers.use();
      return {
        canCreateTab: useSelectorVisible(),
        canEditPanel: PlatformPanel.useCanEditActive(),
      };
    },
    { wrapper: Wrapper },
  );
  return {
    store,
    canCreateTab: () => result.current.canCreateTab,
    canEditPanel: () => result.current.canEditPanel,
  };
};

const viewTab = (key: panel.TabKey = uuid.create()): panel.Tab => ({
  variant: "view",
  key,
  type: "probe",
  args: {},
});

const leafTabs = async (key: panel.Key): Promise<panel.Tab[]> => {
  const { root } = await client.panels.retrieve(key);
  if (root.variant !== "leaf") throw new Error("expected a single-leaf root");
  return root.tabs;
};

const Noop: Modals.Content = () => null;

describe("app/triggers", () => {
  beforeEach(() => {
    mocks.engine = "web";
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("escape", () => {
    it("should stop overlaying the focused tab", async () => {
      const { result, store } = await renderTriggers();
      act(() => void store.dispatch(Session.Panel.startOverlaying({})));
      expect(result.current.overlaid).toBe(true);
      act(pressEscape);
      expect(result.current.overlaid).toBe(false);
    });

    it("should stop overlaying when a first press was claimed elsewhere", async () => {
      const { result, store } = await renderTriggers();
      // A code editor claims the first press to blur itself, so the press that reaches
      // the overlay is the second of a pair and arrives as a double trigger.
      act(pressEscape);
      act(() => void store.dispatch(Session.Panel.startOverlaying({})));
      act(pressEscape);
      expect(result.current.overlaid).toBe(false);
    });
  });

  describe("focus", () => {
    it("should toggle the focused tab in and out of focus mode", async () => {
      const { result, store } = await renderTriggers();
      focusTab(store);
      act(() => press(CONTROL, "KeyL"));
      expect(result.current.overlaid).toBe(true);
      act(() => press(CONTROL, "KeyL"));
      expect(result.current.overlaid).toBe(false);
    });

    it("should also leave focus mode on Escape", async () => {
      const { result, store } = await renderTriggers();
      focusTab(store);
      act(() => press(CONTROL, "KeyL"));
      expect(result.current.overlaid).toBe(true);
      act(pressEscape);
      expect(result.current.overlaid).toBe(false);
    });

    it("should stay out of focus mode when no tab is focused", async () => {
      const { result } = await renderTriggers();
      act(() => press(CONTROL, "KeyL"));
      expect(result.current.overlaid).toBe(false);
    });
  });

  describe("rename", () => {
    const editable = (
      <Text.Editable
        id={PlatformPanel.tabNameID(TAB)}
        value={TAB_NAME}
        onChange={() => {}}
      />
    );

    it("should start an in-place edit of the focused tab's name", async () => {
      const { store } = await renderTriggers(editable);
      focusTab(store);
      const name = screen.getByText(TAB_NAME);
      expect(name.getAttribute("contenteditable")).not.toBe("true");
      act(() => press(CONTROL, "KeyE"));
      expect(name.getAttribute("contenteditable")).toBe("true");
    });

    it("should leave the name alone when no tab is focused", async () => {
      await renderTriggers(editable);
      act(() => press(CONTROL, "KeyE"));
      expect(screen.getByText(TAB_NAME).getAttribute("contenteditable")).not.toBe(
        "true",
      );
    });
  });

  describe("close", () => {
    it("should close the top modal and leave the window open", async () => {
      const { result, store } = await renderTriggers();
      act(() => void result.current.modals.push(Noop, {}, () => {}));
      expect(result.current.modals.isAnyOpen()).toBe(true);
      act(() => press(CONTROL, "KeyW"));
      expect(result.current.modals.isAnyOpen()).toBe(false);
      expect(isWindowOpen(store)).toBe(true);
    });

    it("should close the window when the shortcut is held with nothing open", async () => {
      mocks.engine = "tauri";
      const { store } = await renderTriggers();
      expect(isWindowOpen(store)).toBe(true);
      vi.useFakeTimers();
      act(() => hold(CONTROL, "KeyW"));
      act(() => void vi.advanceTimersByTime(400));
      expect(isWindowOpen(store)).toBe(false);
    });

    // A browser page cannot close its own tab, so the reducer must not drop the
    // window either: the panel selectors key off it.
    it("should spare the window in the browser", async () => {
      const { store } = await renderTriggers();
      vi.useFakeTimers();
      act(() => hold(CONTROL, "KeyW"));
      act(() => void vi.advanceTimersByTime(400));
      expect(isWindowOpen(store)).toBe(true);
    });

    it("should spare the window when the shortcut is released first", async () => {
      mocks.engine = "tauri";
      const { store } = await renderTriggers();
      vi.useFakeTimers();
      act(() => hold(CONTROL, "KeyW"));
      act(() => release(CONTROL, "KeyW"));
      act(() => void vi.advanceTimersByTime(400));
      expect(isWindowOpen(store)).toBe(true);
    });

    it("should remove the focused tab from the panel on the cluster", async () => {
      const closed = viewTab();
      const kept = viewTab();
      const pan = await createServerPanel(client, {
        variant: "leaf",
        tabs: [closed, kept],
      });
      const { store, canEditPanel } = await renderLiveTriggers(pan.key);
      focusTab(store, pan.key, closed.key);
      await waitFor(() => expect(canEditPanel()).toBe(true));
      act(() => press(CONTROL, "KeyW"));
      await waitFor(async () => expect(await leafTabs(pan.key)).toEqual([kept]));
    });
  });

  describe("open window", () => {
    const openedWindows = (store: TestStore): string[] =>
      Drift.selectWindows(store.getState())
        .filter(({ key, reserved }) => key !== Drift.MAIN_WINDOW && reserved)
        .map(({ key }) => key);

    const renderWithSelectedPanel = async (): Promise<TestStore> => {
      const { store } = await renderTriggers();
      act(() => void store.dispatch(Session.Panel.select({ key: PANEL })));
      return store;
    };

    it("should open a window on the selected panel", async () => {
      mocks.engine = "tauri";
      const store = await renderWithSelectedPanel();
      act(() => press(CONTROL, "KeyO"));
      const [opened] = openedWindows(store);
      expect(opened).toBeDefined();
      expect(store.getState().panels.windows[opened]?.selected).toEqual(PANEL);
    });

    it("should do nothing in the browser", async () => {
      const store = await renderWithSelectedPanel();
      act(() => press(CONTROL, "KeyO"));
      expect(openedWindows(store)).toEqual([]);
    });
  });

  describe("create tab", () => {
    it("should open a component selector tab in the selected panel", async () => {
      const seed = viewTab();
      const pan = await createServerPanel(client, {
        variant: "leaf",
        tabs: [seed],
      });
      const { store, canCreateTab } = await renderLiveTriggers(pan.key);
      focusTab(store, pan.key, seed.key);
      // The shortcut is gated on being able to create at least one visualization.
      await waitFor(() => expect(canCreateTab()).toBe(true));
      act(() => press(CONTROL, "KeyT"));
      await waitFor(async () => {
        const tabs = await leafTabs(pan.key);
        expect(
          tabs.some((t) => t.variant === "view" && t.type === Selector.TAB_TYPE),
        ).toBe(true);
      });
    });
  });
});
