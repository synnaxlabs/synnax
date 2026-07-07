// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, createTestClient, panel } from "@synnaxlabs/client";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { renderPalette } from "@/platform/palette/testutil";
import { Session } from "@/session";
import { assertDefined, stubGeometry, uniqueName, waitForFocusedTab } from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("arc palette", () => {
  it("opens the arc explorer view from the open explorer command", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const { store, openCommandPalette } = await renderPalette({
      commands: Arc.COMMANDS,
      client,
    });
    store.dispatch(Session.Project.select(proj.key));
    await openCommandPalette();
    const item = await screen.findByText("Open the Arc Explorer");
    await act(async () => {
      fireEvent.click(item);
    });
    const focused = await waitForFocusedTab(store);
    const panelKey = Session.Panel.selectSelected(store.getState());
    assertDefined(panelKey);
    await waitFor(async () => {
      const doc = await client.panels.retrieve(panelKey);
      const tab = panel.findTab(doc.root, focused);
      if (tab?.variant !== "view") throw new Error("expected a view tab");
      expect(tab.type).toBe(Arc.Explorer.TAB_TYPE);
    });
  });

  it("creates an arc through the create command's modal", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const { store, openCommandPalette } = await renderPalette({
      commands: Arc.COMMANDS,
      client,
    });
    store.dispatch(Session.Project.select(proj.key));
    await openCommandPalette();
    const item = await screen.findByText("Create an Arc Automation");
    await act(async () => {
      fireEvent.click(item);
    });
    const inputs = await screen.findAllByPlaceholderText("Automation Name");
    const input = inputs[inputs.length - 1];
    const name = uniqueName("arc");
    fireEvent.change(input, { target: { value: name } });
    const buttons = screen.getAllByRole("button", { name: "Create" });
    const create = buttons[buttons.length - 1];
    await waitFor(() => expect(create.className).not.toContain("pluto--disabled"));
    await act(async () => {
      fireEvent.click(create);
    });
    const focused = await waitForFocusedTab(store);
    const panelKey = Session.Panel.selectSelected(store.getState());
    assertDefined(panelKey);
    const key = await waitFor(async () => {
      const doc = await client.panels.retrieve(panelKey);
      const tab = panel.findTab(doc.root, focused);
      if (tab?.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.type).toBe(arc.TYPE_ONTOLOGY_ID.type);
      return tab.resource.key;
    });
    const created = await client.arcs.retrieve({ key });
    expect(created.name).toBe(name);
  });
});
