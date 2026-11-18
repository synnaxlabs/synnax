// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, query, view } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/lyra/icon";
import { uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, renderHook, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { assert, describe, expect, it, onTestFinished } from "vitest";

import { Panel } from "@/platform/panel";
import { View } from "@/platform/view";
import { createTabWrapper, type TabWrapper } from "@/platform/view/testutil";
import { Session } from "@/session";
import { awaitTextEditing, commitTextEdit, renderSuspended } from "@/testutil";

const client = createTestClient();

const TAB = View.createTab({
  range: {
    Content: () => <>range explorer</>,
    Name: () => <>Range explorer</>,
    Icon: Icon.Range,
  },
  status: {
    Content: () => <>status explorer</>,
    Name: () => <>Status explorer</>,
    Icon: Icon.Status,
  },
});

const createView = async (type: view.Type = "range"): Promise<view.View> => {
  const created = await client.views.create({ name: uniqueName("view"), type });
  onTestFinished(async () => await client.views.delete(created.key));
  return created;
};

/** @returns a tab that shows the saved view. */
const showing = (saved: view.View): panel.NewTab => ({
  variant: "resource",
  resource: view.resourceID(saved.key),
});

/** Renders the element in a tab that shows the saved view. */
const renderShowing = async (
  saved: view.View,
  ui: ReactElement,
): Promise<TabWrapper> => {
  const wrapped = await createTabWrapper({ client, tab: showing(saved) });
  await renderSuspended(ui, { wrapper: wrapped.wrapper });
  return wrapped;
};

describe("View.createTab", () => {
  it("shows the explorer of the type of its view", async () => {
    await renderShowing(await createView("status"), <TAB.Content />);
    expect(await screen.findByText("status explorer")).toBeTruthy();
  });

  it("names the tab after the type and the name of its view", async () => {
    const saved = await createView("status");
    await renderShowing(saved, <TAB.Name />);
    expect(await screen.findByText(saved.name)).toBeTruthy();
    expect(screen.getByText("Statuses ›")).toBeTruthy();
  });

  it("follows a rename of its view", async () => {
    const saved = await createView();
    await renderShowing(saved, <TAB.Name />);
    await screen.findByText(saved.name);
    const renamed = uniqueName("renamed");
    await act(async () => await client.views.rename(saved.key, renamed));
    expect(await screen.findByText(renamed)).toBeTruthy();
  });

  it("renames its view from the tab name", async () => {
    const saved = await createView();
    const { tabKey } = await renderShowing(saved, <TAB.Name />);
    await screen.findByText(saved.name);
    act(() => Panel.editTabName(tabKey));
    const editor = await awaitTextEditing(Panel.tabNameID(tabKey));
    const renamed = uniqueName("renamed");
    act(() => commitTextEdit(editor, renamed));
    await waitFor(async () =>
      expect((await client.views.retrieve({ key: saved.key })).name).toBe(renamed),
    );
  });

  it("restores a deleted view under its original key", async () => {
    const { restore } = TAB;
    assert(restore != null);
    const saved = await createView();
    // Restore rebuilds from the corpse the cache holds, which exists only once the
    // client has seen the view live.
    await client.views.retrieve({ key: saved.key });
    await client.views.delete(saved.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.views.watch(saved.key).getCached())).toBe(
        true,
      ),
    );
    const { key: project } = await client.projects.create({ name: uniqueName("proj") });
    await restore({ client, project, resource: view.resourceID(saved.key) });
    // The restoring client writes through its own cache, so a second client reads it.
    const restored = await createTestClient().views.retrieve({ key: saved.key });
    expect(restored.name).toEqual(saved.name);
  });
});

describe("View.useOpenExplorer", () => {
  const OTHER: panel.NewTab = { variant: "view", type: "other" };

  /**
   * Opens the range explorer from a panel with the other tab focused. A second client
   * opens it, so no view it decides on is in its cache.
   */
  const openFrom = async (beside: panel.NewTab[]): Promise<TabWrapper> => {
    const wrapped = await createTabWrapper({
      client,
      as: createTestClient(),
      tab: OTHER,
      beside,
    });
    const { result } = renderHook(
      () => View.useOpenExplorer({ resourceType: "range", tabType: "explorer" }),
      { wrapper: wrapped.wrapper },
    );
    act(() => result.current());
    return wrapped;
  };

  const explorerOpens = async (panelKey: panel.Key): Promise<void> =>
    await waitFor(async () => {
      const { root } = await client.panels.retrieve(panelKey);
      expect(panel.findTabByType(root, "explorer")).toBeDefined();
    });

  it("focuses a tab that shows a saved view of the type", async () => {
    const saved = await createView();
    const key = uuid.create();
    const { store, panelKey } = await openFrom([{ ...showing(saved), key }]);
    await waitFor(() =>
      expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([
        key,
      ]),
    );
    const { root } = await client.panels.retrieve(panelKey);
    expect(panel.findTabByType(root, "explorer")).toBeUndefined();
  });

  it("opens an explorer when the panel shows only a view of another type", async () => {
    const saved = await createView("status");
    const { panelKey } = await openFrom([showing(saved)]);
    await explorerOpens(panelKey);
  });

  it("opens an explorer when the panel shows only a deleted view", async () => {
    const saved = await createView();
    await client.views.delete(saved.key);
    const { panelKey } = await openFrom([showing(saved)]);
    await explorerOpens(panelKey);
  });
});
