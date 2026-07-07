// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError, ontology, task } from "@synnaxlabs/client";
import { List, Text } from "@synnaxlabs/pluto";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { Modals } from "@/platform/modals";
import { findLastButton } from "@/platform/modals/testutil";
import { type Ontology } from "@/platform/ontology";
import {
  createBaseProps,
  createExecutingHandleError,
  createResource,
  createSelection,
  createState,
} from "@/platform/ontology/testutil";
import { Session } from "@/session";
import {
  assertDefined,
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  createTestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const { onSelect, TreeContextMenu: Menu } = Task.ONTOLOGY_SERVICE;
assertDefined(onSelect, "task ontology service has no onSelect");
assertDefined(Menu, "task ontology service has no TreeContextMenu");

const createTask = async () => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await rack.createTask({
    name: uniqueName("task"),
    type: NI.Task.ANALOG_READ_TYPE,
    config: {},
  });
};

interface CreateMenuPropsArgs {
  tasks: task.Task[];
  overrides?: Partial<Ontology.BaseProps>;
  snapshot?: boolean;
}

const createMenuProps = async ({
  tasks,
  overrides,
  snapshot = false,
}: CreateMenuPropsArgs): Promise<Ontology.TreeContextMenuProps> => {
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

const renderMenu = async (props: Ontology.TreeContextMenuProps) => {
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
    it("should open the task's configuration view", async () => {
      const t = await createTask();
      const openTab = vi.fn();
      const store = await createTestStore();
      onSelect({
        ...createBaseProps({
          client,
          store,
          overrides: { openTab, handleError: createExecutingHandleError() },
        }),
        selection: [createResource(t.ontologyID, t.name)],
      });
      await waitFor(() => expect(openTab).toHaveBeenCalledTimes(1));
      const opened = openTab.mock.calls[0][0];
      expect(opened.view.type).toBe(NI.Task.ANALOG_READ_TYPE);
      expect(opened.view.args).toEqual({ taskKey: t.key });
    });

    it("should report an error for a task that does not exist", async () => {
      const errors: string[] = [];
      const store = await createTestStore();
      onSelect({
        ...createBaseProps({
          client,
          store,
          overrides: {
            handleError: createExecutingHandleError((message) => errors.push(message)),
          },
        }),
        selection: [createResource(task.ontologyID("999999999999"), "Ghost Task")],
      });
      await waitFor(() => expect(errors).toContain("Could not open Ghost Task"));
    });

    it("should do nothing for an empty selection", async () => {
      const openTab = vi.fn();
      const store = await createTestStore();
      onSelect({
        ...createBaseProps({
          client,
          store,
          overrides: { openTab, handleError: createExecutingHandleError() },
        }),
        selection: [],
      });
      expect(openTab).not.toHaveBeenCalled();
    });
  });

  describe("TreeContextMenu", () => {
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
      expect(openTab.mock.calls[0][0].view.args).toEqual({ taskKey: t.key });
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
      const props: Ontology.TreeContextMenuProps = {
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
