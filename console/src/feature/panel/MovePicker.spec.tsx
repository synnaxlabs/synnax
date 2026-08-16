// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { useMovePicker } from "@/feature/panel/MovePicker";
import { Modals } from "@/platform/modals";
import { createPanelWrapper } from "@/platform/panel/testutil";
import { Session } from "@/session";
import { type TestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const OPEN = "open the picker";

interface Harness {
  store: TestStore;
  source: panel.Panel;
  tab: panel.Tab;
  destination: panel.Panel;
  other: panel.Panel;
}

const createTab = (): panel.Tab => ({
  variant: "view",
  key: uuid.create(),
  type: "t",
  args: {},
});

const rowNames = (): (string | null)[] =>
  Array.from(document.querySelectorAll(".console-panel-move-picker__row")).map(
    (row) => row.textContent,
  );

const createPanel = async (
  projectKey: project.Key,
  tabs: panel.Tab[],
): Promise<panel.Panel> =>
  await client.panels.create({
    key: uuid.create(),
    name: uniqueName("panel"),
    parent: project.ontologyID(projectKey),
    root: { variant: "leaf", tabs },
  });

const createHarness = async (): Promise<Harness> => {
  const { key: projectKey } = await client.projects.create({
    name: uniqueName("project"),
    layout: {},
  });
  const tab = createTab();
  const source = await createPanel(projectKey, [tab]);
  const destination = await createPanel(projectKey, [createTab()]);
  const other = await createPanel(projectKey, [createTab()]);
  const { wrapper, store } = await createPanelWrapper({ client, project: projectKey });
  store.dispatch(Session.Panel.select({ key: source.key }));

  const Trigger = (): ReactElement => {
    const open = useMovePicker();
    return (
      <button onClick={() => open({ origin: { panel: source.key, tab } })}>
        {OPEN}
      </button>
    );
  };

  await act(async () => {
    render(
      <>
        <Trigger />
        <Modals.Stack />
      </>,
      { wrapper },
    );
  });
  await act(async () => {
    fireEvent.click(screen.getByText(OPEN));
  });
  await screen.findByText("Move to panel");
  return { store, source, tab, destination, other };
};

describe("Panel.MovePicker", () => {
  it("should move the tab into the panel the user picks", async () => {
    const { store, source, tab, destination } = await createHarness();
    await act(async () => {
      fireEvent.click(await screen.findByText(destination.name));
    });
    await waitFor(async () => {
      const [srcDoc, dstDoc] = await Promise.all([
        client.panels.retrieve(source.key),
        client.panels.retrieve(destination.key),
      ]);
      expect(panel.findTab(dstDoc.root, tab.key)).toEqual(tab);
      expect(panel.findTab(srcDoc.root, tab.key)).toBeUndefined();
    });
    await waitFor(() =>
      expect(Session.Panel.selectSelected(store.getState())).toEqual(destination.key),
    );
  });

  it("should mint a panel holding the tab when New panel is picked", async () => {
    const { store, source, tab } = await createHarness();
    await act(async () => {
      fireEvent.click(await screen.findByText("New panel"));
    });
    await waitFor(() => {
      const selected = Session.Panel.selectSelected(store.getState());
      if (selected == null || selected === source.key)
        throw new Error("no minted panel selected");
    });
    const minted = Session.Panel.selectSelected(store.getState()) as panel.Key;
    await waitFor(async () => {
      const [srcDoc, mintedDoc] = await Promise.all([
        client.panels.retrieve(source.key),
        client.panels.retrieve(minted),
      ]);
      expect(panel.findTab(mintedDoc.root, tab.key)).toEqual(tab);
      expect(panel.findTab(srcDoc.root, tab.key)).toBeUndefined();
    });
  });

  // The picker offers the same panels the strip shows, so it has to list them in the
  // order the user set there.
  it("should list the panels in the strip's order", async () => {
    const { store, source, destination, other } = await createHarness();
    await screen.findByText(destination.name);
    await screen.findByText(other.name);
    const order = [source, destination, other].map(({ key, name }) => ({ key, name }));
    act(() => {
      store.dispatch(Session.Panel.reconcileOrder({ panels: order }));
      store.dispatch(Session.Panel.reorder({ key: destination.key, index: 0 }));
    });
    expect(rowNames()).toEqual([destination.name, other.name, "New panel"]);
    act(() => {
      store.dispatch(Session.Panel.reorder({ key: other.key, index: 0 }));
    });
    expect(rowNames()).toEqual([other.name, destination.name, "New panel"]);
  });

  // Moving a tab to the panel it already sits in is not a move, so the panel it came
  // from is not offered.
  it("should not offer the panel the tab came from", async () => {
    const { source, destination } = await createHarness();
    await screen.findByText(destination.name);
    expect(screen.queryByText(source.name)).toBeNull();
  });
});
