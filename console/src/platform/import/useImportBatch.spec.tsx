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
import { type Status } from "@synnaxlabs/pluto";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Import } from "@/platform/import";
import { createPanelWrapper } from "@/platform/panel/testutil";
import { Session } from "@/session";
import { CaptureStatuses, type TestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const createLogID = async (): Promise<ontology.ID> => {
  const proj = await client.projects.create({
    name: uniqueName("project"),
    layout: {},
  });
  const created = await client.logs.create(proj.key, { name: uniqueName("log") });
  return log.ontologyID(created.key);
};

interface RenderResult {
  importBatch: Import.ImportBatch;
  store: TestStore;
}

const renderImportBatch = async (
  onStatuses?: (statuses: Status.NotificationSpec[]) => void,
): Promise<RenderResult> => {
  const { wrapper: Panel, store } = await createPanelWrapper({ client });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Panel>
      {children}
      {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
    </Panel>
  );
  const { result } = renderHook(() => Import.useImportBatch(), { wrapper });
  return { importBatch: result.current, store };
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
    const { importBatch, store } = await renderImportBatch();
    await act(
      async () =>
        await importBatch({
          items: [{ name: "a.json" }, { name: "b.json" }],
          importItem: async ({ name }) => (name === "a.json" ? first : second),
        }),
    );
    const panelKey = await awaitPanel(store);
    await waitFor(async () =>
      expect(await resourceKeys(panelKey)).toEqual([first.key, second.key]),
    );
  });

  // A dropped directory imports a whole project, which brings its own panels, so it
  // creates no resource for the batch to open.
  it("opens no tab for an item that created no resource", async () => {
    const id = await createLogID();
    const { importBatch, store } = await renderImportBatch();
    await act(
      async () =>
        await importBatch({
          items: [{ name: "my-project" }, { name: "a.json" }],
          importItem: async ({ name }) => (name === "a.json" ? id : undefined),
        }),
    );
    const panelKey = await awaitPanel(store);
    await waitFor(async () => expect(await resourceKeys(panelKey)).toEqual([id.key]));
  });

  it("reports a failed item by name and still opens tabs for the rest", async () => {
    const id = await createLogID();
    let statuses: Status.NotificationSpec[] = [];
    const { importBatch, store } = await renderImportBatch((s) => (statuses = s));
    await act(
      async () =>
        await importBatch({
          items: [{ name: "notes.txt" }, { name: "a.json" }],
          importItem: async ({ name }) => {
            if (name === "notes.txt") throw new Error("not a JSON file");
            return id;
          },
        }),
    );
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to import notes.txt"),
    );
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
    const { importBatch, store } = await renderImportBatch();
    let batch!: Promise<void>;
    act(() => {
      batch = importBatch({
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
