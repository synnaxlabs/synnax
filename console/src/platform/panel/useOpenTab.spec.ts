// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
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
import { resolveFocusedTab, waitForFocusedTab } from "@/testutil";

const client = createTestClient();

const viewLeaf = (key: string, type: string): panel.New["root"] => ({
  variant: "leaf",
  tabs: [{ variant: "view", key, type, args: {} }],
});

const leafTabs = (root: panel.Node): panel.Tab[] => {
  if (root.variant !== "leaf") throw new Error("expected a single-leaf root");
  return root.tabs;
};

describe("Panel.useOpenTab", () => {
  describe("inserting into an existing panel", () => {
    it("inserts the tab into the scoped panel on the cluster and focuses it", async () => {
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
  });

  describe("creating a new panel", () => {
    it("creates a panel on the cluster seeding the opened tab", async () => {
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
      expect(doc.name).toBe("New Panel");
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
