// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { createResourceTab } from "@/platform/panel/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  getIconButton,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const createGraphArc = async (graph: Partial<arc.Arc["graph"]> = {}) =>
  await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    graph: { nodes: [], edges: [], ...graph },
  });

const renderToolbar = async (arcKey: string): Promise<{ store: TestStore }> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const { panelKey, tabKey } = await createResourceTab(client, arc.ontologyID(arcKey));
  await act(async () => {
    render(
      <PlutoPanel.Scope.Provider value={panelKey}>
        <PlutoPanel.TabScope.Provider value={tabKey}>
          <Suspense fallback={null}>
            <Arc.Editor.Toolbar />
          </Suspense>
        </PlutoPanel.TabScope.Provider>
      </PlutoPanel.Scope.Provider>,
      { wrapper },
    );
  });
  return { store };
};

describe("arc editor toolbar", () => {
  it("shows the arc name and the stage groups once editable", async () => {
    const arc = await createGraphArc();
    await renderToolbar(arc.key);
    expect(await screen.findByText(arc.name)).toBeTruthy();
    expect(await screen.findByText("Basic")).toBeTruthy();
    expect(screen.getByText("Telemetry")).toBeTruthy();
    expect(screen.getByText("Operators")).toBeTruthy();
    expect(screen.getByText("Flow Control")).toBeTruthy();
    expect(screen.getAllByText("Constant").length).toBeGreaterThan(0);
  });

  it("switches stage groups when a group is clicked", async () => {
    const arc = await createGraphArc();
    await renderToolbar(arc.key);
    await waitFor(() =>
      expect(screen.getAllByText("Constant").length).toBeGreaterThan(0),
    );
    fireEvent.click(screen.getByText("Telemetry"));
    await waitFor(() => expect(screen.queryAllByText("Constant")).toHaveLength(0));
  });

  it("prompts for a selection on the properties tab", async () => {
    const arc = await createGraphArc();
    await renderToolbar(arc.key);
    await screen.findByText("Basic");
    fireEvent.click(screen.getByText("Properties"));
    expect(
      await screen.findByText("Select an Arc element to configure its properties."),
    ).toBeTruthy();
  });

  it("shows the selected node's form and breadcrumb on the properties tab", async () => {
    const arc = await createGraphArc({
      nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
      inputs: { n1: { type: "constant", value: 42 } },
    });
    const { store } = await renderToolbar(arc.key);
    await screen.findByText("Basic");
    store.dispatch(Session.Arc.setSelected({ key: arc.key, selected: ["n1"] }));
    fireEvent.click(screen.getByText("Properties"));
    const input = await screen.findByDisplayValue("42");
    expect(input).toBeTruthy();
    expect(screen.getAllByText("Constant").length).toBeGreaterThan(0);
  });

  it("offers alignment controls when multiple nodes are selected", async () => {
    const arc = await createGraphArc({
      nodes: [
        { key: "n1", position: { x: 0, y: 0 } },
        { key: "n2", position: { x: 10, y: 10 } },
      ],
      inputs: {
        n1: { type: "constant", value: 1 },
        n2: { type: "constant", value: 2 },
      },
    });
    const { store } = await renderToolbar(arc.key);
    await screen.findByText("Basic");
    store.dispatch(Session.Arc.setSelected({ key: arc.key, selected: ["n1", "n2"] }));
    fireEvent.click(screen.getByText("Properties"));
    expect(await screen.findByText("Align")).toBeTruthy();
  });

  it("keeps the alignment controls alive when the diagram is not mounted", async () => {
    const arc = await createGraphArc({
      nodes: [
        { key: "n1", position: { x: 0, y: 0 } },
        { key: "n2", position: { x: 10, y: 10 } },
      ],
      inputs: {
        n1: { type: "constant", value: 1 },
        n2: { type: "constant", value: 2 },
      },
    });
    const { store } = await renderToolbar(arc.key);
    await screen.findByText("Basic");
    store.dispatch(Session.Arc.setSelected({ key: arc.key, selected: ["n1", "n2"] }));
    fireEvent.click(screen.getByText("Properties"));
    await screen.findByText("Align");
    fireEvent.click(getIconButton(document.body, "align-y-center"));
    fireEvent.click(getIconButton(document.body, "align-x-center"));
    expect(screen.getByText("Align")).toBeTruthy();
  });

  it("offers to enable editing when the arc is not editable", async () => {
    const arc = await createGraphArc();
    const { store } = await renderToolbar(arc.key);
    await screen.findByText("Basic");
    store.dispatch(Session.Arc.setEditable({ key: arc.key, editable: false }));
    expect(await screen.findByText(/is not editable/)).toBeTruthy();
    fireEvent.click(await screen.findByText("enable editing."));
    await waitFor(() =>
      expect(
        Session.Arc.selectState({ state: store.getState(), key: arc.key }).graph
          .editable,
      ).toBe(true),
    );
  });

  it("renders the text toolbar for a text-mode arc", async () => {
    const arc = await client.arcs.create({
      name: uniqueName("arc"),
      mode: "text",
      text: { raw: "x = 1" },
    });
    await renderToolbar(arc.key);
    expect(await screen.findByText(arc.name)).toBeTruthy();
    expect(await screen.findByText("No Content")).toBeTruthy();
    expect(screen.queryByText("Basic")).toBeNull();
  });
});
