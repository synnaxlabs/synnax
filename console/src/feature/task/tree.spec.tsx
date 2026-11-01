// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, NotFoundError, ontology, type task } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { Modals } from "@/platform/modals";
import { findLastButton } from "@/platform/modals/testutil";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createExecutingHandleError,
  createResource,
  createSelection,
  createState,
  findTreeRow,
  renderOntologyTree,
  renderTreeContextMenu,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  assertDefined,
  awaitTextEditingElement,
  CaptureStatuses,
  commitTextEdit,
  createConsoleWrapper,
  createTestStore,
  openContextMenu,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const Item = Task.TREE_ITEMS.task;
const { ContextMenu: Menu } = Item;
assertDefined(Menu, "task ontology service has no ContextMenu");

const createTask = async () => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await rack.createTask({
    name: uniqueName("task"),
    type: NI.Task.ANALOG_READ_TYPE,
    config: {},
  });
};

const renderTaskTree = async (t: task.Task) => {
  const [parent] = await client.ontology.parents.retrieve({ ids: t.ontologyID });
  const grp = await client.groups.create({
    parent: ontology.ROOT_ID,
    name: uniqueName("taskgrp"),
  });
  await client.ontology.moveChildren(
    parent.id,
    group.ontologyID(grp.key),
    t.ontologyID,
  );
  return await renderOntologyTree({
    client,
    root: group.ontologyID(grp.key),
    items: Task.TREE_ITEMS,
  });
};

interface CreateMenuPropsParams {
  tasks: task.Task[];
  overrides?: Partial<Tree.BaseProps>;
  snapshot?: boolean;
}

const createMenuProps = async ({
  tasks,
  overrides,
  snapshot = false,
}: CreateMenuPropsParams): Promise<Tree.ContextMenuProps> => {
  const ids = tasks.map((t) => t.ontologyID);
  const store = await createTestStore();
  return {
    ...createBaseProps({ client, store, overrides }),
    selection: createSelection({ ids }),
    state: createState(
      ids.map((id, i) => createResource(id, tasks[i].name, { snapshot })),
    ),
  };
};

const renderMenu = async (props: Tree.ContextMenuProps) => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const statuses: Status.NotificationSpec[] = [];
  render(
    <>
      <Menu {...props} />
      <Modals.Stack />
      <CaptureStatuses
        onStatuses={(next) => statuses.splice(0, statuses.length, ...next)}
      />
    </>,
    { wrapper },
  );
  return { store, statuses };
};

describe("task ontology", () => {
  describe("onSelect", () => {
    it("should open the task's configuration tab when double-clicked", async () => {
      const t = await createTask();
      const { store } = await renderTaskTree(t);
      fireEvent.doubleClick(await findTreeRow(t.name));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.type).toBe("task");
      expect(tab.resource.key).toBe(t.key);
    });
  });

  describe("ContextMenu", () => {
    it("should render the single-selection menu items", async () => {
      const t = await createTask();
      const props = await createMenuProps({ tasks: [t] });
      await renderMenu(props);
      expect(await screen.findByText("Rename")).toBeTruthy();
      expect(screen.getByText("Edit configuration")).toBeTruthy();
      expect(screen.getByText("Copy link")).toBeTruthy();
      expect(screen.getByText("Export")).toBeTruthy();
      expect(screen.getByText("Delete")).toBeTruthy();
      expect(screen.queryByText(/Snapshot to/)).toBeNull();
    });

    it("should label the edit item View configuration for snapshots", async () => {
      const t = await createTask();
      const props = await createMenuProps({ tasks: [t], snapshot: true });
      await renderMenu(props);
      expect(await screen.findByText("View configuration")).toBeTruthy();
    });

    it("should open the configuration tab from Edit configuration", async () => {
      const t = await createTask();
      const openTab = vi.fn();
      const props = await createMenuProps({
        tasks: [t],
        overrides: { openTab, handleError: createExecutingHandleError() },
      });
      await renderMenu(props);
      fireEvent.click(await screen.findByText("Edit configuration"));
      await waitFor(() => expect(openTab).toHaveBeenCalledTimes(1));
      expect(openTab.mock.calls[0][0]).toMatchObject({
        variant: "resource",
        resource: { type: "task", key: t.key },
      });
    });

    it("should delete the task after confirmation", async () => {
      const t = await createTask();
      const props = await createMenuProps({ tasks: [t] });
      await renderMenu(props);
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${t.name}?`);
      fireEvent.click(findLastButton("Delete"));
      await waitFor(async () => {
        await expect(client.tasks.retrieve(t.key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
    });

    it("should rename the task through the inline editor", async () => {
      const t = await createTask();
      await renderTaskTree(t);
      await openContextMenu(t.name);
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditingElement();
      const renamed = uniqueName("renamed");
      await act(async () => {
        commitTextEdit(editor, renamed);
      });
      await waitFor(async () =>
        expect((await client.tasks.retrieve(t.key)).name).toBe(renamed),
      );
    });

    it("should snapshot the selection to the active range", async () => {
      const t = await createTask();
      const start = TimeStamp.now();
      const rng = await client.ranges.create({
        name: uniqueName("range"),
        timeRange: { start, end: start.add(TimeSpan.seconds(1)) },
      });
      const props = await createMenuProps({ tasks: [t] });
      const { store, statuses } = await renderMenu(props);
      store.dispatch(Session.Range.add(Session.Range.fromClient(rng.payload)));
      store.dispatch(Session.Range.select(rng.key));
      fireEvent.click(await screen.findByText(`Snapshot to ${rng.name}`));
      await waitFor(async () => {
        const children = await client.ontology.children.retrieve({
          ids: rng.ontologyID,
        });
        expect(children.some((c) => c.id.type === "task")).toBe(true);
      });
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            variant: "success",
            name: rng.name,
            message: "Snapshotted task",
            description: `Created a snapshot of ${t.name}.`,
          }),
        ),
      );
    });
  });
});

describe("permission to write the task", () => {
  it("should offer Reload Console", async () => {
    const t = await createTask();
    await renderTreeContextMenu(Menu, {
      client,
      resources: [createResource(t.ontologyID, t.name, { snapshot: false })],
    });
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should withhold rename, grouping, and delete from a viewer", async () => {
    const t = await createTask();
    await renderTreeContextMenu(Menu, {
      client: await roles.get("Viewer"),
      resources: [createResource(t.ontologyID, t.name, { snapshot: false })],
    });
    expect(await screen.findByText("Export")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Group selection")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });
});
