// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project, ranger, type Synnax as Client } from "@synnaxlabs/client";
import {
  type BuiltInRole,
  createTestClient,
  RoleClients,
} from "@synnaxlabs/client/testutil";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Panel } from "@/platform/panel";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { Session } from "@/session";
import {
  createTestClientWithGrants,
  resolveFocusedTab,
  uniqueName,
  waitForFocusedTab,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

let projectKey: string | undefined;
const testProject = async (): Promise<string> =>
  (projectKey ??= (
    await client.projects.create({ name: uniqueName("project"), layout: {} })
  ).key);

const viewLeaf = (key: string, type: string): panel.New["root"] => ({
  variant: "leaf",
  tabs: [{ variant: "view", key, type, args: {} }],
});

// A node key no tree in these specs reaches: the root's own children stop at 3.
const STALE_LEAF = 99;

const leafTabs = (root: panel.Node): panel.Tab[] => {
  if (root.variant !== "leaf") throw new Error("expected a single-leaf root");
  return root.tabs;
};

describe("Panel.useOpenTab", () => {
  describe("inserting into an existing panel", () => {
    it("inserts the tab into the scoped panel on the Core and focuses it", async () => {
      const existing = await createServerPanel(client, viewLeaf(uuid.create(), "seed"));
      const { wrapper, store } = await createPanelWrapper({
        client,
        panelKey: existing.key,
      });
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      const newTabKey = uuid.create();
      await act(async () => {
        result.current({ variant: "view", key: newTabKey, type: "added", args: {} });
      });
      await waitFor(async () => {
        const doc = await client.panels.retrieve(existing.key);
        expect(panel.findTab(doc.root, newTabKey)).toBeDefined();
      });
      await waitFor(() =>
        expect(
          Session.Panel.selectSelectedTabs(store.getState(), existing.key)[0],
        ).toBe(newTabKey),
      );
      const doc = await client.panels.retrieve(existing.key);
      expect(leafTabs(doc.root)).toHaveLength(2);
    });

    it("inserts into the session-selected panel when no panel scope is present", async () => {
      const existing = await createServerPanel(client, viewLeaf(uuid.create(), "seed"));
      const { wrapper, store } = await createPanelWrapper({ client });
      store.dispatch(
        Session.Panel.select({ key: existing.key, windowKey: MAIN_WINDOW }),
      );
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      const newTabKey = uuid.create();
      await act(async () => {
        result.current({ variant: "view", key: newTabKey, type: "added", args: {} });
      });
      await waitFor(async () => {
        const doc = await client.panels.retrieve(existing.key);
        expect(panel.findTab(doc.root, newTabKey)).toBeDefined();
      });
      await waitFor(() =>
        expect(
          Session.Panel.selectSelectedTabs(store.getState(), existing.key)[0],
        ).toBe(newTabKey),
      );
    });

    it("prefers the surrounding panel scope over the session-selected panel", async () => {
      const scoped = await createServerPanel(client, viewLeaf(uuid.create(), "seed"));
      const selected = await createServerPanel(client, viewLeaf(uuid.create(), "seed"));
      const { wrapper, store } = await createPanelWrapper({
        client,
        panelKey: scoped.key,
      });
      store.dispatch(
        Session.Panel.select({ key: selected.key, windowKey: MAIN_WINDOW }),
      );
      await primePanel(wrapper, scoped.key);
      await primePanel(wrapper, selected.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      const newTabKey = uuid.create();
      await act(async () => {
        result.current({ variant: "view", key: newTabKey, type: "added", args: {} });
      });
      await waitFor(async () => {
        const doc = await client.panels.retrieve(scoped.key);
        expect(panel.findTab(doc.root, newTabKey)).toBeDefined();
      });
      const selectedDoc = await client.panels.retrieve(selected.key);
      expect(panel.findTab(selectedDoc.root, newTabKey)).toBeUndefined();
    });

    it("focuses the existing tab when the resource already backs one", async () => {
      const seedTabKey = uuid.create();
      const resource = ranger.ontologyID(uuid.create());
      const existing = await createServerPanel(client, {
        variant: "leaf",
        tabs: [{ variant: "resource", key: seedTabKey, resource }],
      });
      const { wrapper, store } = await createPanelWrapper({
        client,
        panelKey: existing.key,
      });
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      await act(async () => {
        result.current({ variant: "resource", key: uuid.create(), resource });
      });
      await waitFor(() =>
        expect(
          Session.Panel.selectSelectedTabs(store.getState(), existing.key)[0],
        ).toBe(seedTabKey),
      );
      const doc = await client.panels.retrieve(existing.key);
      expect(leafTabs(doc.root)).toHaveLength(1);
    });

    it("opens a new tab beside the surrounding one when the params omit a key", async () => {
      const seedTabKey = uuid.create();
      const existing = await createServerPanel(client, viewLeaf(seedTabKey, "seed"));
      const { wrapper } = await createPanelWrapper({
        client,
        panelKey: existing.key,
        tabKey: seedTabKey,
      });
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      await act(async () => {
        result.current({ variant: "view", type: "added", args: {} });
      });
      await waitFor(async () => {
        const doc = await client.panels.retrieve(existing.key);
        expect(leafTabs(doc.root)).toHaveLength(2);
      });
      const doc = await client.panels.retrieve(existing.key);
      const seed = panel.findTab(doc.root, seedTabKey);
      if (seed == null || seed.variant !== "view") throw new Error("seed tab missing");
      expect(seed.type).toBe("seed");
    });

    it("replaces the tab in place when the params carry that tab's key", async () => {
      const seedTabKey = uuid.create();
      const existing = await createServerPanel(client, viewLeaf(seedTabKey, "seed"));
      const { wrapper } = await createPanelWrapper({
        client,
        panelKey: existing.key,
        tabKey: seedTabKey,
      });
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      await act(async () => {
        result.current({
          variant: "view",
          key: seedTabKey,
          type: "replaced",
          args: {},
        });
      });
      await waitFor(async () => {
        const doc = await client.panels.retrieve(existing.key);
        const tab = panel.findTab(doc.root, seedTabKey);
        if (tab == null || tab.variant !== "view")
          throw new Error("tab not replaced yet");
        expect(tab.type).toBe("replaced");
      });
      const doc = await client.panels.retrieve(existing.key);
      expect(leafTabs(doc.root)).toHaveLength(1);
    });

    it("focuses the existing view when a singleton view of the same type is reopened", async () => {
      const seedTabKey = uuid.create();
      const existing = await createServerPanel(
        client,
        viewLeaf(seedTabKey, "range_explorer"),
      );
      const { wrapper, store } = await createPanelWrapper({
        client,
        panelKey: existing.key,
      });
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      await act(async () => {
        result.current(
          { variant: "view", type: "range_explorer", args: {} },
          { singleton: true },
        );
      });
      await waitFor(() =>
        expect(
          Session.Panel.selectSelectedTabs(store.getState(), existing.key)[0],
        ).toBe(seedTabKey),
      );
      const doc = await client.panels.retrieve(existing.key);
      expect(leafTabs(doc.root)).toHaveLength(1);
    });

    it("focuses a new keyless resource tab opened beside the current tab", async () => {
      const curTabKey = uuid.create();
      const existing = await createServerPanel(client, viewLeaf(curTabKey, "current"));
      const { wrapper, store } = await createPanelWrapper({
        client,
        panelKey: existing.key,
        tabKey: curTabKey,
      });
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      const resource = ranger.ontologyID(uuid.create());
      await act(async () => {
        result.current({ variant: "resource", resource });
      });
      await waitFor(async () => {
        const doc = await client.panels.retrieve(existing.key);
        expect(leafTabs(doc.root)).toHaveLength(2);
      });
      const doc = await client.panels.retrieve(existing.key);
      const resourceTab = leafTabs(doc.root).find((t) => t.variant === "resource");
      if (resourceTab == null) throw new Error("resource tab missing");
      await waitFor(() =>
        expect(
          Session.Panel.selectSelectedTabs(store.getState(), existing.key)[0],
        ).toBe(resourceTab.key),
      );
    });

    it("splits the placed leaf and puts the tab in the half the split creates", async () => {
      const seedTabKey = uuid.create();
      const existing = await createServerPanel(client, viewLeaf(seedTabKey, "seed"));
      const { wrapper } = await createPanelWrapper({
        client,
        panelKey: existing.key,
      });
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      const newTabKey = uuid.create();
      await act(async () => {
        result.current(
          { variant: "view", key: newTabKey, type: "dropped", args: {} },
          { placement: { leaf: panel.ROOT_NODE_KEY, location: "left" } },
        );
      });
      await waitFor(async () => {
        const { root } = await client.panels.retrieve(existing.key);
        if (root.variant !== "split") throw new Error("root did not split");
        expect(panel.findTab(root.first, newTabKey)).toBeDefined();
        expect(panel.findTab(root.last, seedTabKey)).toBeDefined();
      });
    });

    it("falls back to an unplaced open when the placed leaf is gone", async () => {
      const seedTabKey = uuid.create();
      const existing = await createServerPanel(client, viewLeaf(seedTabKey, "seed"));
      const { wrapper } = await createPanelWrapper({
        client,
        panelKey: existing.key,
      });
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      const newTabKey = uuid.create();
      await act(async () => {
        result.current(
          { variant: "view", key: newTabKey, type: "dropped", args: {} },
          { placement: { leaf: STALE_LEAF, location: "left" } },
        );
      });
      await waitFor(async () => {
        const doc = await client.panels.retrieve(existing.key);
        expect(panel.findTab(doc.root, newTabKey)).toBeDefined();
      });
      const doc = await client.panels.retrieve(existing.key);
      expect(leafTabs(doc.root)).toHaveLength(2);
    });

    it("opens a second view of the same type when singleton is not set", async () => {
      const seedTabKey = uuid.create();
      const existing = await createServerPanel(
        client,
        viewLeaf(seedTabKey, "range_explorer"),
      );
      const { wrapper } = await createPanelWrapper({
        client,
        panelKey: existing.key,
        tabKey: seedTabKey,
      });
      await primePanel(wrapper, existing.key);
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      await act(async () => {
        result.current({ variant: "view", type: "range_explorer", args: {} });
      });
      await waitFor(async () => {
        const doc = await client.panels.retrieve(existing.key);
        expect(leafTabs(doc.root)).toHaveLength(2);
      });
    });
  });

  describe("creating a new panel", () => {
    it("creates a panel on the Core seeding the opened tab", async () => {
      const { wrapper, store } = await createPanelWrapper({ client });
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      const tabKey = uuid.create();
      await act(async () => {
        result.current({ variant: "view", key: tabKey, type: "created", args: {} });
      });
      const tab = await resolveFocusedTab(store, client, (t) => t.key === tabKey);
      expect(tab.variant).toBe("view");
      const panelKey = Session.Panel.selectSelected(store.getState());
      if (panelKey == null) throw new Error("no panel selected after create");
      const doc = await client.panels.retrieve(panelKey);
      expect(doc.name).toBe("New panel");
      expect(panel.findTab(doc.root, tabKey)).toBeDefined();
    });

    it("selects the new panel and focuses the seeded tab in the session", async () => {
      const { wrapper, store } = await createPanelWrapper({ client });
      const { result } = renderHook(() => Panel.useOpenTab(), { wrapper });
      const tabKey = uuid.create();
      await act(async () => {
        result.current({ variant: "view", key: tabKey, type: "created", args: {} });
      });
      const focused = await waitForFocusedTab(store);
      expect(focused).toBe(tabKey);
      expect(Session.Panel.selectSelected(store.getState())).toBeDefined();
    });
  });
});

describe("Panel.useOpenTabs", () => {
  it("splits the placed leaf once and fills the new half with the whole batch", async () => {
    const seedTabKey = uuid.create();
    const existing = await createServerPanel(client, viewLeaf(seedTabKey, "seed"));
    const { wrapper } = await createPanelWrapper({ client, panelKey: existing.key });
    await primePanel(wrapper, existing.key);
    const { result } = renderHook(() => Panel.useOpenTabs(), { wrapper });
    const keys = [uuid.create(), uuid.create(), uuid.create()];
    await act(async () => {
      result.current(
        keys.map((key) => ({ variant: "view", key, type: "dropped", args: {} })),
        { placement: { leaf: panel.ROOT_NODE_KEY, location: "left" } },
      );
    });
    await waitFor(async () => {
      const { root } = await client.panels.retrieve(existing.key);
      if (root.variant !== "split") throw new Error("root did not split");
      expect(leafTabs(root.first).map((t) => t.key)).toEqual(keys);
      expect(leafTabs(root.last).map((t) => t.key)).toEqual([seedTabKey]);
    });
  });

  it("opens the rest of the batch when one tab's resource is already open", async () => {
    const seedTabKey = uuid.create();
    const resource = ranger.ontologyID(uuid.create());
    const existing = await createServerPanel(client, {
      variant: "leaf",
      tabs: [{ variant: "resource", key: seedTabKey, resource }],
    });
    const { wrapper } = await createPanelWrapper({ client, panelKey: existing.key });
    await primePanel(wrapper, existing.key);
    const { result } = renderHook(() => Panel.useOpenTabs(), { wrapper });
    const freshKey = uuid.create();
    await act(async () => {
      result.current([
        { variant: "resource", key: uuid.create(), resource },
        { variant: "view", key: freshKey, type: "added", args: {} },
      ]);
    });
    await waitFor(async () => {
      const doc = await client.panels.retrieve(existing.key);
      expect(leafTabs(doc.root).map((t) => t.key)).toEqual([seedTabKey, freshKey]);
    });
  });

  // Focus follows the last requested tab. When that tab is a duplicate the reducer
  // skips it, so focus has to resolve to the tab already holding the resource.
  it("focuses the tab already holding the resource when the batch ends on a duplicate", async () => {
    const seedTabKey = uuid.create();
    const resource = ranger.ontologyID(uuid.create());
    const existing = await createServerPanel(client, {
      variant: "leaf",
      tabs: [{ variant: "resource", key: seedTabKey, resource }],
    });
    const { wrapper, store } = await createPanelWrapper({
      client,
      panelKey: existing.key,
    });
    await primePanel(wrapper, existing.key);
    const { result } = renderHook(() => Panel.useOpenTabs(), { wrapper });
    await act(async () => {
      result.current([
        { variant: "view", key: uuid.create(), type: "added", args: {} },
        { variant: "resource", key: uuid.create(), resource },
      ]);
    });
    await waitFor(() =>
      expect(Session.Panel.selectSelectedTabs(store.getState(), existing.key)[0]).toBe(
        seedTabKey,
      ),
    );
  });
});

describe("Panel.useCanOpenTab", () => {
  const renderCan = async (as: Client, panelKey?: panel.Key) => {
    const { wrapper } = await createPanelWrapper({
      client: as,
      project: await testProject(),
      panelKey,
    });
    if (panelKey != null) await primePanel(wrapper, panelKey);
    return renderHook(
      () => ({
        open: Panel.useCanOpenTab(),
        edit: Panel.useCanEditActive(),
      }),
      { wrapper },
    );
  };

  it.each<BuiltInRole>(["Viewer", "Operator"])(
    "should refuse both to a %s",
    async (role) => {
      const existing = await createServerPanel(client, viewLeaf(uuid.create(), "seed"));
      const { result } = await renderCan(await roles.get(role), existing.key);
      await waitFor(() => expect(result.current.edit).toBe(false));
      expect(result.current.open).toBe(false);
    },
  );

  it("should allow both for an engineer", async () => {
    const existing = await createServerPanel(client, viewLeaf(uuid.create(), "seed"));
    const { result } = await renderCan(await roles.get("Engineer"), existing.key);
    await waitFor(() => expect(result.current.open).toBe(true));
    expect(result.current.edit).toBe(true);
  });

  it("should fall back to the create grant when no panel is in scope", async () => {
    const creator = await createTestClientWithGrants(client, {
      retrieve: [panel.TYPE_ONTOLOGY_ID, project.TYPE_ONTOLOGY_ID],
      create: [panel.TYPE_ONTOLOGY_ID],
    });
    const { result } = await renderCan(creator);
    await waitFor(() => expect(result.current.open).toBe(true));
    expect(result.current.edit).toBe(false);
  });
});
