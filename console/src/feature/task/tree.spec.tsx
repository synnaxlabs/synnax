// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, NotFoundError, ontology, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List, Text } from "@synnaxlabs/pluto";
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
} from "@/platform/tree/testutil";
import { findTreeRow, renderOntologyTree } from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import {
  assertDefined,
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  createTestStore,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

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
  const [parent] = await client.ontology.retrieveParents(t.ontologyID);
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

interface CreateMenuPropsArgs {
  tasks: task.Task[];
  overrides?: Partial<Tree.BaseProps>;
  snapshot?: boolean;
}

const createMenuProps = async ({
  tasks,
  overrides,
  snapshot = false,
}: CreateMenuPropsArgs): Promise<Tree.ContextMenuProps> => {
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
  render(
    <>
      <Menu {...props} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { store };
};

describe("task ontology", () => {
  describe("onSelect", () => {
    it("should open the task's configuration view when double-clicked", async () => {
      const t = await createTask();
      const { store } = await renderTaskTree(t);
      fireEvent.doubleClick(await findTreeRow(t.name));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "view") throw new Error("expected a view tab");
      expect(tab.type).toBe(NI.Task.ANALOG_READ_TYPE);
      expect(tab.args).toEqual({ taskKey: t.key });
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

    it("should open the configuration view from Edit configuration", async () => {
      const t = await createTask();
      const openTab = vi.fn();
      const props = await createMenuProps({
        tasks: [t],
        overrides: { openTab, handleError: createExecutingHandleError() },
      });
      await renderMenu(props);
      fireEvent.click(await screen.findByText("Edit configuration"));
      await waitFor(() => expect(openTab).toHaveBeenCalledTimes(1));
      expect(openTab.mock.calls[0][0]).toEqual({
        variant: "view",
        type: NI.Task.ANALOG_READ_TYPE,
        args: { taskKey: t.key },
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
        await expect(client.tasks.retrieve({ key: t.key })).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
    });

    it("should rename the task through the inline editor", async () => {
      const t = await createTask();
      const store = await createTestStore();
      const props: Tree.ContextMenuProps = {
        ...createBaseProps({ client, store }),
        selection: createSelection({ ids: [t.ontologyID] }),
        state: createState([createResource(t.ontologyID, t.name, { snapshot: false })]),
      };
      const itemID = List.itemNameID(ontology.idToString(t.ontologyID));
      const RenameHarness = () => {
        const rename = Task.useRename(props);
        return (
          <>
            <button onClick={rename}>trigger rename</button>
            <Text.MaybeEditable id={itemID} value={t.name} onChange={() => {}} />
          </>
        );
      };
      const { wrapper } = await createConsoleWrapper({ client, store });
      render(<RenameHarness />, { wrapper });
      fireEvent.click(screen.getByText("trigger rename"));
      const editor = await awaitTextEditing(itemID);
      const renamed = uniqueName("renamed");
      await act(async () => {
        commitTextEdit(editor, renamed);
      });
      await waitFor(async () =>
        expect((await client.tasks.retrieve({ key: t.key })).name).toBe(renamed),
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
      const { store } = await renderMenu(props);
      store.dispatch(Session.Range.add(Session.Range.fromClient(rng.payload)));
      store.dispatch(Session.Range.select(rng.key));
      fireEvent.click(await screen.findByText(`Snapshot to ${rng.name}`));
      await waitFor(async () => {
        const children = await client.ontology.retrieveChildren(rng.ontologyID);
        expect(children.some((c) => c.id.type === "task")).toBe(true);
      });
    });
  });
});
