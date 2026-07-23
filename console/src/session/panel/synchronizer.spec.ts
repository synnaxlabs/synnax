// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Session } from "@/session";
import { renderHookWithConsole, type TestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const leaf = (...tabKeys: string[]): panel.Node => ({
  variant: "leaf",
  tabs: tabKeys.map((key) => ({ variant: "view", key, type: "t", args: {} })),
});

const createPanel = async (key: panel.Key, root: panel.Node): Promise<panel.Panel> =>
  await client.panels.create(
    panel.panelZ.parse({ key, name: uniqueName("panel"), root }),
  );

const selectTab = (store: TestStore, key: panel.Key, tabKey: panel.TabKey): void =>
  void store.dispatch(
    Session.Panel.internalSelectTab({ key, tabKey, otherTabKeys: [tabKey] }),
  );

const mount = async () =>
  await renderHookWithConsole(
    () => Session.Panel.WINDOW_SYNCHRONIZERS.useReconcileTabSelections(),
    { client },
  );

describe("Panel.WINDOW_SYNCHRONIZERS", () => {
  it("reconciles a stale selection when the panel document loads", async () => {
    const panelKey = uuid.create();
    const [tab, ghost] = [uuid.create(), uuid.create()];
    const { store } = await mount();
    act(() => selectTab(store, panelKey, ghost));
    await act(async () => {
      await createPanel(panelKey, leaf(tab));
    });
    await waitFor(() => {
      expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([
        tab,
      ]);
    });
  });

  it("converges the selection when the selected tab is removed from the tree", async () => {
    const panelKey = uuid.create();
    const [tabA, tabB] = [uuid.create(), uuid.create()];
    const { store } = await mount();
    act(() => selectTab(store, panelKey, tabA));
    await act(async () => {
      await createPanel(panelKey, leaf(tabA, tabB));
    });
    await waitFor(() => {
      expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([
        tabA,
      ]);
    });
    await act(async () => {
      await client.panels.dispatch(panelKey, panel.removeTab({ key: tabA }));
    });
    await waitFor(() => {
      expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([
        tabB,
      ]);
    });
  });

  it("reconciles from the cache when an already-cached panel is selected", async () => {
    const panelKey = uuid.create();
    const tab = uuid.create();
    const { store } = await mount();
    await act(async () => {
      await createPanel(panelKey, leaf(tab));
    });
    expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([]);
    act(() => {
      store.dispatch(Session.Panel.select({ key: panelKey }));
    });
    await waitFor(() => {
      expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([
        tab,
      ]);
    });
  });
});
