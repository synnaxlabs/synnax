// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Selector } from "@/feature/panel/Selector";
import { Modals } from "@/platform/modals";
import { createPanelWrapper } from "@/platform/panel/testutil";
import { findModalButton } from "@/platform/tree/menuTestutil";
import { Session } from "@/session";
import { type TestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const createProjectPanel = async (projectKey: project.Key): Promise<panel.Panel> =>
  await client.panels.create({
    key: uuid.create(),
    name: uniqueName("panel"),
    parent: project.ontologyID(projectKey),
    root: {
      variant: "leaf",
      tabs: [{ variant: "view", key: uuid.create(), type: "t", args: {} }],
    },
  });

// The strip's own order decides which panel is a neighbor, so the row is read back
// from the rendered pills instead of assumed from creation order.
const renderStrip = async (
  panels: panel.Panel[],
  projectKey: project.Key,
): Promise<{ store: TestStore; row: panel.Panel[] }> => {
  const { wrapper, store } = await createPanelWrapper({ client, project: projectKey });
  await act(async () => {
    render(
      <>
        <Selector />
        <Modals.Stack />
      </>,
      { wrapper },
    );
  });
  await waitFor(() =>
    panels.forEach(({ name }) => expect(screen.getByText(name)).toBeTruthy()),
  );
  const row = screen
    .getAllByRole("tab")
    .map((pill) => panels.find(({ name }) => pill.textContent?.includes(name)))
    .filter((pan) => pan != null);
  expect(row).toHaveLength(panels.length);
  return { store, row };
};

const deletePanel = async (pan: panel.Panel): Promise<void> => {
  fireEvent.contextMenu(screen.getByText(pan.name));
  fireEvent.click(await screen.findByText("Delete"));
  await screen.findByText(`Are you sure you want to delete ${pan.name}?`);
  await act(async () => {
    fireEvent.click(findModalButton("Delete"));
  });
};

describe("Panel.Selector", () => {
  it("should select a newly created panel", async () => {
    const { wrapper, store } = await createPanelWrapper({ client });
    await act(async () => {
      render(<Selector />, { wrapper });
    });
    // The strip suspends on the project's panel list, so nothing renders until it
    // resolves.
    const create = await screen.findByRole("button");
    expect(Session.Panel.selectSelected(store.getState())).toBeUndefined();
    await act(async () => {
      fireEvent.click(create);
    });
    const selected = Session.Panel.selectSelected(store.getState());
    expect(selected).toBeDefined();
    await waitFor(() => expect(screen.getByText("New Panel")).toBeTruthy());
  });

  // Deleting the selected panel hands the window to the panel beside it in the
  // strip, the way closing a browser tab does.
  describe("delete", () => {
    const createStrip = async (): Promise<{ store: TestStore; row: panel.Panel[] }> => {
      const { key: projectKey } = await client.projects.create({
        name: uniqueName("project"),
        layout: {},
      });
      const panels = [
        await createProjectPanel(projectKey),
        await createProjectPanel(projectKey),
        await createProjectPanel(projectKey),
      ];
      return await renderStrip(panels, projectKey);
    };

    const select = async (store: TestStore, pan: panel.Panel): Promise<void> => {
      await act(async () => {
        fireEvent.click(screen.getByText(pan.name));
      });
      expect(Session.Panel.selectSelected(store.getState())).toEqual(pan.key);
    };

    // Each test visits the far end of the strip first, so the neighbor and the
    // most recently used panel are different panels.
    it("should select the panel to the right of the deleted one", async () => {
      const { store, row } = await createStrip();
      await select(store, row[2]);
      await select(store, row[0]);
      await deletePanel(row[0]);
      await waitFor(() =>
        expect(Session.Panel.selectSelected(store.getState())).toEqual(row[1].key),
      );
    });

    it("should select the panel to the left when the deleted one was last", async () => {
      const { store, row } = await createStrip();
      await select(store, row[0]);
      await select(store, row[2]);
      await deletePanel(row[2]);
      await waitFor(() =>
        expect(Session.Panel.selectSelected(store.getState())).toEqual(row[1].key),
      );
    });
  });
});
