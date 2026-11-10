// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  group,
  log,
  NotFoundError,
  project,
  resource,
  schematic,
} from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Project } from "@/feature/project";
import {
  createResource,
  findModalButton,
  findTreeRow,
  renderLiveTree,
  renderTreeContextMenu,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  assertDefined,
  captureBrowserDownloads,
  createTestStore,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const Item = Project.TREE_ITEMS.project;

const createProject = async () =>
  await client.projects.create({ name: uniqueName("project") });

const projectResource = (key: string, name: string) =>
  createResource(project.resourceID(key), name);

// The context menu's hooks require an active project, matching the Project.Guard the
// production tree renders within.
const createStoreWithActive = async (key: string) =>
  await createTestStore({
    preloadedState: {
      [Session.Project.SLICE_NAME]: { version: 0, selected: key },
    },
  });

afterEach(() => {
  vi.restoreAllMocks();
});

describe("project tree item", () => {
  it("should offer Reload Console", async () => {
    const p = await createProject();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [projectResource(p.key, p.name)],
      store: await createStoreWithActive(p.key),
    });
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

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
    expect(screen.getByText("Import components")).toBeTruthy();
    expect(screen.getByText("Export")).toBeTruthy();
    expect(screen.getByText("Copy link")).toBeTruthy();
  });

  it("should offer Copy, and no single-project actions, for a multiple selection", async () => {
    const a = await createProject();
    const b = await createProject();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [projectResource(a.key, a.name), projectResource(b.key, b.name)],
      store: await createStoreWithActive(a.key),
    });
    expect(await screen.findByText("Copy")).toBeTruthy();
    expect(screen.queryByText("Copy link")).toBeNull();
    expect(screen.queryByText("Export")).toBeNull();
  });

  it("should export the clicked project rather than the active one", async () => {
    const p = await createProject();
    const logName = uniqueName("log");
    await client.logs.create({ project: p.key, name: logName });
    const active = await createProject();
    const downloads = captureBrowserDownloads();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client,
      resources: [projectResource(p.key, p.name)],
      store: await createStoreWithActive(active.key),
    });
    fireEvent.click(await screen.findByText("Export"));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    // The zip takes the clicked row's name and carries its members, proving the
    // export took the clicked project's key.
    expect(downloads.anchors[0].download).toBe(`${p.name}.zip`);
    const archive = new TextDecoder().decode(await downloads.blobs[0].arrayBuffer());
    expect(archive).toContain(`${logName}.json`);
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
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("focused tab is not a resource");
    const created = await client.logs.retrieve(tab.resource.key);
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
    const { store } = await renderLiveTree({
      client,
      root: project.TYPE_RESOURCE_ID,
      listRoot: Project.listTreeRoot,
      items: Project.TREE_ITEMS,
    });
    fireEvent.doubleClick(await findTreeRow(p.name));
    await waitFor(() =>
      expect(Session.Project.selectOptionalSelected(store.getState())).toBe(p.key),
    );
  });

  it("should only accept visualizations and groups on drop", () => {
    const source: Haul.Item = { key: "tree", type: PTree.HAUL_TYPE };
    const destination = projectResource("p1", "project");
    const root = project.TYPE_RESOURCE_ID;
    const canDrop = (key: string): boolean =>
      Item.canDrop({
        source,
        items: [PTree.createHaulItem(key)],
        root,
        destination,
        ancestors: [root],
        moved: [{ id: resource.parseID(key), ancestors: [root] }],
      });
    expect(canDrop("log:abc")).toBe(true);
    expect(canDrop("schematic:abc")).toBe(true);
    expect(canDrop("group:abc")).toBe(true);
    expect(canDrop("channel:1")).toBe(false);
  });
});

describe("project tree children", () => {
  it("should list the project's groups and ungrouped visualizations", async () => {
    const p = await createProject();
    const id = project.resourceID(p.key);
    const g = await client.groups.create({ parent: id, name: uniqueName("group") });
    const s = await client.schematics.create({
      project: p.key,
      name: uniqueName("schematic"),
    });
    const l = await client.logs.create({ project: p.key, name: uniqueName("log") });
    await client.batch.send(client.logs.setGroupRequest(l.key, g.key));
    await client.lineplots.create({
      project: (await createProject()).key,
      name: "other",
    });
    assertDefined(Item.children);
    const sources = Item.children({ client, id, root: project.TYPE_RESOURCE_ID });
    const listed = await Promise.all(sources.map(async (src) => await src.retrieve()));
    expect(
      listed.flatMap(({ resources }) => resources.map(({ name }) => name)),
    ).toEqual([g.name, s.name]);
  });
});

describe("project tree drop", () => {
  it("should move visualizations and groups dropped from another project", async () => {
    const src = await createProject();
    const dst = await createProject();
    const srcID = project.resourceID(src.key);
    const ancestors = [srcID, project.TYPE_RESOURCE_ID];
    const s = await client.schematics.create({
      project: src.key,
      name: uniqueName("schematic"),
    });
    const g = await client.groups.create({ parent: srcID, name: uniqueName("group") });
    const l = await client.logs.create({ project: src.key, name: uniqueName("log") });
    await client.batch.send(client.logs.setGroupRequest(l.key, g.key));
    assertDefined(Item.drop);
    await Item.drop({
      client,
      destination: projectResource(dst.key, dst.name),
      items: [
        { id: schematic.resourceID(s.key), ancestors },
        { id: group.resourceID(g.key), ancestors },
      ],
    });
    const [movedSchematic] = await client.schematics.retrieve({ projects: [dst.key] });
    expect(movedSchematic.key).toBe(s.key);
    const [movedLog] = await client.logs.retrieve({ projects: [dst.key] });
    expect(movedLog.key).toBe(l.key);
    expect(movedLog.group).toBe(g.key);
    const [movedGroup] = await client.groups.retrieve({
      parents: [project.resourceID(dst.key)],
    });
    expect(movedGroup?.key).toBe(g.key);
  });

  it("should leave visualizations dropped within their own project there", async () => {
    const p = await createProject();
    const id = project.resourceID(p.key);
    const g = await client.groups.create({ parent: id, name: uniqueName("group") });
    const l = await client.logs.create({ project: p.key, name: uniqueName("log") });
    await client.batch.send(client.logs.setGroupRequest(l.key, g.key));
    assertDefined(Item.drop);
    await Item.drop({
      client,
      destination: projectResource(p.key, p.name),
      items: [
        {
          id: log.resourceID(l.key),
          ancestors: [group.resourceID(g.key), id, project.TYPE_RESOURCE_ID],
        },
      ],
    });
    const [moved] = await client.logs.retrieve({ projects: [p.key] });
    expect(moved.key).toBe(l.key);
    expect(moved.group).toBeUndefined();
  });
});

describe("permission to write the project", () => {
  it("should withhold rename, the create actions, and delete from a viewer", async () => {
    const p = await createProject();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: await roles.get("Viewer"),
      resources: [projectResource(p.key, p.name)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
    expect(screen.queryByText("New Line Plot")).toBeNull();
    expect(screen.queryByText("New Schematic")).toBeNull();
  });
});
