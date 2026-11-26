// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type panel, ranger, type resource } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { uuid } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Panel } from "@/platform/panel";
import { createServerPanel } from "@/platform/panel/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const client = createTestClient();

const OTHER_WINDOW = "other";

const createID = (): resource.ID => ranger.resourceID(uuid.create());

const createTab = (id: resource.ID): panel.Tab => ({
  variant: "resource",
  key: uuid.create(),
  resource: id,
});

interface Shown {
  panelKey: panel.Key;
  tabKey: panel.TabKey;
  /** The window that shows the tab. Defaults to the current window. */
  windowKey?: string;
}

const show = (store: TestStore, { panelKey, tabKey, windowKey }: Shown): void => {
  store.dispatch(Session.Panel.select({ key: panelKey, windowKey }));
  store.dispatch(
    Session.Panel.internalSelectTab({
      key: panelKey,
      tabKey,
      otherTabKeys: [],
      windowKey,
    }),
  );
};

/** Renders the check with a second window, which has the given focus. */
const renderIsShown = async (otherFocused = false) => {
  const main = Drift.ZERO_SLICE_STATE.windows[MAIN_WINDOW];
  const { wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: {
      [Drift.SLICE_NAME]: {
        ...Drift.ZERO_SLICE_STATE,
        windows: {
          [MAIN_WINDOW]: main,
          [OTHER_WINDOW]: { ...main, key: OTHER_WINDOW, reserved: true },
        },
        labelKeys: { [MAIN_WINDOW]: MAIN_WINDOW, [OTHER_WINDOW]: OTHER_WINDOW },
        keyLabels: { [MAIN_WINDOW]: MAIN_WINDOW, [OTHER_WINDOW]: OTHER_WINDOW },
      },
    },
  });
  store.dispatch(Drift.setWindowProps({ key: OTHER_WINDOW, focus: otherFocused }));
  const { result } = renderHook(() => Panel.useGetIsShown(), { wrapper });
  return { isShown: result.current, store };
};

describe("Panel.useGetIsShown", () => {
  it("should be true when the current window shows a tab of the resource", async () => {
    const id = createID();
    const tab = createTab(id);
    const { key } = await createServerPanel(client, { variant: "leaf", tabs: [tab] });
    const { isShown, store } = await renderIsShown();
    show(store, { panelKey: key, tabKey: tab.key });
    expect(await isShown(id)).toBe(true);
  });

  it("should be false when the window shows another tab of the same leaf", async () => {
    const id = createID();
    const tab = createTab(id);
    const other = createTab(createID());
    const { key } = await createServerPanel(client, {
      variant: "leaf",
      tabs: [tab, other],
    });
    const { isShown, store } = await renderIsShown();
    show(store, { panelKey: key, tabKey: other.key });
    expect(await isShown(id)).toBe(false);
  });

  it("should be false when no panel holds a tab of the resource", async () => {
    const tab = createTab(createID());
    const { key } = await createServerPanel(client, { variant: "leaf", tabs: [tab] });
    const { isShown, store } = await renderIsShown();
    show(store, { panelKey: key, tabKey: tab.key });
    expect(await isShown(createID())).toBe(false);
  });

  it("should be true when another focused window shows the tab", async () => {
    const id = createID();
    const tab = createTab(id);
    const { key } = await createServerPanel(client, { variant: "leaf", tabs: [tab] });
    const { isShown, store } = await renderIsShown(true);
    show(store, { panelKey: key, tabKey: tab.key, windowKey: OTHER_WINDOW });
    expect(await isShown(id)).toBe(true);
  });

  it("should be false when only an unfocused window shows the tab", async () => {
    const id = createID();
    const tab = createTab(id);
    const { key } = await createServerPanel(client, { variant: "leaf", tabs: [tab] });
    const { isShown, store } = await renderIsShown(false);
    show(store, { panelKey: key, tabKey: tab.key, windowKey: OTHER_WINDOW });
    expect(await isShown(id)).toBe(false);
  });

  it("should reject when the Core does not have the selected panel", async () => {
    const { isShown, store } = await renderIsShown();
    show(store, { panelKey: uuid.create(), tabKey: uuid.create() });
    await expect(isShown(createID())).rejects.toThrow(NotFoundError);
  });
});
