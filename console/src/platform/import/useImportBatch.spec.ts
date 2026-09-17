// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type ontology, type panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/pluto";
import { act, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Import } from "@/platform/import";
import { Session } from "@/session";
import { renderHookWithConsole, type TestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const createLogID = async (): Promise<ontology.ID> => {
  const proj = await client.projects.create({
    name: uniqueName("project"),
    layout: {},
  });
  const created = await client.logs.create(proj.key, { name: uniqueName("log") });
  return log.ontologyID(created.key);
};

const renderImportBatch = async () => {
  const proj = await client.projects.create({
    name: uniqueName("project"),
    layout: {},
  });
  const { result, store } = await renderHookWithConsole(
    () => ({
      importBatch: Import.useImportBatch(),
      notifications: Status.useNotifications(),
    }),
    { client, preloadedState: { project: { version: 0, selected: proj.key } } },
  );
  return { result, store };
};

const awaitPanel = async (store: TestStore): Promise<panel.Key> =>
  await waitFor(() => {
    const selected = Session.Panel.selectSelected(store.getState());
    if (selected == null) throw new Error("no panel opened for the batch");
    return selected;
  });

const resourceKeys = async (panelKey: panel.Key): Promise<string[]> => {
  const { root } = await client.panels.retrieve(panelKey);
  if (root.variant !== "leaf") throw new Error("expected a leaf");
  return root.tabs.map((tab) => {
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    return tab.resource.key;
  });
};

describe("Import.useImportBatch", () => {
  it("opens the created resources as one batch of tabs in item order", async () => {
    const [first, second] = [await createLogID(), await createLogID()];
    const { result, store } = await renderImportBatch();
    await act(
      async () =>
        await result.current.importBatch({
          items: [{ name: "a.json" }, { name: "b.json" }],
          importItem: async ({ name }) => (name === "a.json" ? first : second),
        }),
    );
    const panelKey = await awaitPanel(store);
    await waitFor(async () =>
      expect(await resourceKeys(panelKey)).toEqual([first.key, second.key]),
    );
  });

  // A bundle import brings its own panels and returns no resource, so a batch where
  // every item does that must not open an empty panel.
  it("opens no panel when no item created a resource", async () => {
    const onSuccess = vi.fn();
    const { result, store } = await renderImportBatch();
    await act(
      async () =>
        await result.current.importBatch({
          items: [{ name: "a.zip" }, { name: "b.zip" }],
          importItem: async () => undefined,
          onSuccess,
        }),
    );
    expect(onSuccess).toHaveBeenCalledTimes(2);
    expect(Session.Panel.selectSelected(store.getState())).toBeUndefined();
  });

  it("reports a failed item by name and still opens tabs for the rest", async () => {
    const id = await createLogID();
    const onSuccess = vi.fn();
    const { result, store } = await renderImportBatch();
    await act(
      async () =>
        await result.current.importBatch({
          items: [{ name: "notes.txt" }, { name: "a.json" }],
          importItem: async ({ name }) => {
            if (name === "notes.txt") throw new Error("not a JSON file");
            return id;
          },
          onSuccess,
        }),
    );
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.map((status) => status.message),
      ).toContain("Failed to import notes.txt"),
    );
    expect(onSuccess).toHaveBeenCalledTimes(1);
    expect(onSuccess).toHaveBeenCalledWith({ name: "a.json" });
    const panelKey = await awaitPanel(store);
    await waitFor(async () => expect(await resourceKeys(panelKey)).toEqual([id.key]));
  });

  it("imports concurrently and orders tabs by item, not by finish order", async () => {
    let release = (): void => {};
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    const [slow, fast] = [await createLogID(), await createLogID()];
    const finished: string[] = [];
    const { result, store } = await renderImportBatch();
    let batch!: Promise<void>;
    act(() => {
      batch = result.current.importBatch({
        items: [{ name: "slow.json" }, { name: "fast.json" }],
        importItem: async ({ name }) => {
          const isSlow = name === "slow.json";
          if (isSlow) await held;
          finished.push(name);
          return isSlow ? slow : fast;
        },
      });
    });
    await waitFor(() => expect(finished).toEqual(["fast.json"]));
    expect(Session.Panel.selectSelected(store.getState())).toBeUndefined();
    release();
    await act(async () => await batch);
    const panelKey = await awaitPanel(store);
    await waitFor(async () =>
      expect(await resourceKeys(panelKey)).toEqual([slow.key, fast.key]),
    );
  });
});
