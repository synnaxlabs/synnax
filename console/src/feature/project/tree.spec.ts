// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, panel, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { findModalButton, renderTreeContextMenu } from "@/platform/tree/menuTestutil";
import { createResource } from "@/platform/tree/testutil";
import { findTreeRow, renderOntologyTree } from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import {
  assertDefined,
  createTestStore,
  uniqueName,
  waitForFocusedTab,
} from "@/testutil";

const client = createTestClient();

const Item = Project.TREE_ITEMS.project;

const createProject = async () =>
  await client.projects.create({ name: uniqueName("project"), layout: {} });

const projectResource = (key: string, name: string) =>
  createResource(project.ontologyID(key), name);

// The context menu's hooks require an active project, matching the Project.Guard the
// production tree renders within.
const createStoreWithActive = async (key: string) =>
  await createTestStore({
    preloadedState: {
      [Session.Project.SLICE_NAME]: { version: 0, selected: key },
    },
  });

describe("project ontology service", () => {
  it("should expose creation, import, export, and link actions", async () => {
    const p = await createProject();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [projectResource(p.key, p.name)],
      store: await createStoreWithActive(p.key),
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.getByText("Create line plot")).toBeTruthy();
    expect(screen.getByText("Create log")).toBeTruthy();
    expect(screen.getByText("Create table")).toBeTruthy();
    expect(screen.getByText("Create schematic")).toBeTruthy();
    expect(screen.getByText("Import component(s)")).toBeTruthy();
    expect(screen.getByText("Export")).toBeTruthy();
    expect(screen.getByText("Copy link")).toBeTruthy();
  });

  it("should create a log inside the project from the context menu", async () => {
    const p = await createProject();
    assertDefined(Item.ContextMenu);
    const { store } = await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [projectResource(p.key, p.name)],
      store: await createStoreWithActive(p.key),
    });
    fireEvent.click(await screen.findByText("Create log"));
    const focusedTab = await waitForFocusedTab(store);
    const panelKey = Session.Panel.selectSelected(store.getState());
    assertDefined(panelKey, "no panel selected");
    const doc = await client.panels.retrieve(panelKey);
    const tab = panel.findTab(doc.root, focusedTab);
    if (tab?.variant !== "resource") throw new Error("focused tab is not a resource");
    const created = await client.logs.retrieve({ key: tab.resource.key });
    expect(created.name).toBe("Log");
    expect(Session.Project.selectOptionalSelected(store.getState())).toBe(p.key);
  });

  it("should clear the active project when it is deleted", async () => {
    const p = await createProject();
    const store = await createStoreWithActive(p.key);
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [projectResource(p.key, p.name)],
      store,
    });
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${p.name}?`);
    fireEvent.click(findModalButton("Delete"));
    await waitFor(() =>
      expect(Session.Project.selectOptionalSelected(store.getState())).toBeUndefined(),
    );
    await waitFor(async () => {
      await expect(client.projects.retrieve(p.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("should select the project when the row is double-clicked", async () => {
    const p = await createProject();
    const [parent] = await client.ontology.retrieveParents(project.ontologyID(p.key));
    const { store } = await renderOntologyTree({
      client,
      root: parent.id,
      items: Project.TREE_ITEMS,
    });
    fireEvent.doubleClick(await findTreeRow(p.name));
    await waitFor(() =>
      expect(Session.Project.selectOptionalSelected(store.getState())).toBe(p.key),
    );
  });

  it("should only accept mosaic-compatible children on drop", () => {
    const { canDrop } = Item;
    const source = { key: "s", type: "t" };
    expect(canDrop({ source, items: [{ key: "log:abc", type: "log" }] })).toBe(true);
    expect(canDrop({ source, items: [{ key: "schematic:abc", type: "x" }] })).toBe(
      true,
    );
    expect(canDrop({ source, items: [{ key: "channel:1", type: "channel" }] })).toBe(
      false,
    );
  });
});
