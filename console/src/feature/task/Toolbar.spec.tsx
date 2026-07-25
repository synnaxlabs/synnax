// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type panel, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type record, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import {
  awaitCommand,
  type CreatedPanel,
  createSelectedPanel,
  createTaskStatus,
} from "@/platform/task/testutil";
import { Session } from "@/session";
import {
  assertDefined,
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  getIconButton,
  openContextMenu,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

interface CreateTaskOptions {
  config?: Record<string, unknown>;
  running?: boolean;
}

// The toolbar lists every task on the cluster, so leaked tasks from prior tests slow
// each render. Track created resources and remove them after each test.
const createdTasks: task.Key[] = [];
const createdRacks: number[] = [];

// Deletion tests already removed their task, so a NotFound here is expected.
const ignoreNotFound = (err: unknown) => {
  if (!NotFoundError.matches(err)) console.error(err);
};

afterEach(async () => {
  await client.tasks.delete(createdTasks.splice(0)).catch(ignoreNotFound);
  await client.racks.delete(createdRacks.splice(0)).catch(ignoreNotFound);
});

const createTask = async ({ config = {}, running }: CreateTaskOptions = {}) => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  createdRacks.push(rack.key);
  const t = await rack.createTask({
    name: uniqueName("task"),
    type: NI.Task.ANALOG_READ_TYPE,
    config,
  });
  createdTasks.push(t.key);
  if (running != null)
    await client.statuses.set(
      createTaskStatus({
        key: task.statusKey(t.key),
        details: { task: t.key, running },
      }),
    );
  return t;
};

const renderToolbar = async () => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const created = await createSelectedPanel(store, client);
  // useOpenTab reads the panel query cache; warm it and keep it subscribed
  // so dispatches stay visible.
  await client.panels.retrieve({ key: created.panelKey });
  render(
    <Task.RegistryProvider registry={Task.REGISTRY}>
      {Task.TOOLBAR.content}
      <Modals.Stack />
    </Task.RegistryProvider>,
    { wrapper },
  );
  return { created, store };
};

const awaitTab = async (created: CreatedPanel, type: string): Promise<record.Unknown> =>
  await waitFor(async () => {
    const doc = await client.panels.retrieve({ key: created.panelKey });
    if (doc.root.variant !== "leaf") throw new Error("panel root is not a leaf");
    const tab = doc.root.tabs.find(
      (t): t is panel.TabView => t.variant === "view" && t.type === type,
    );
    assertDefined(tab, `no ${type} tab was opened`);
    return tab.args;
  });

// The task's reported status streams in after the initial list fetch. Reopen the menu
// until the status-dependent item is present so status-gated flows are deterministic.
const openContextMenuUntil = async (name: string, text: string): Promise<void> => {
  await expect
    .poll(async () => {
      await openContextMenu(name);
      return screen.queryByText(text) != null;
    })
    .toBe(true);
};

describe("task/Toolbar", () => {
  it("renders a created task with its parsed type", async () => {
    const t = await createTask();
    await renderToolbar();
    expect(await screen.findByText(t.name)).toBeTruthy();
    expect((await screen.findAllByText("NI Analog Read Task")).length).toBeGreaterThan(
      0,
    );
  });

  it("opens the task selector tab from the create action", async () => {
    const { created } = await renderToolbar();
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    const args = await awaitTab(created, "taskSelector");
    expect(args).toEqual({});
  });

  it("opens the task's configuration tab on double click", async () => {
    const t = await createTask();
    const { created } = await renderToolbar();
    await screen.findByText(t.name);
    // Re-query synchronously: async status/permission resolutions can replace the
    // text node, detaching a match held across an await before the event lands.
    fireEvent.doubleClick(screen.getByText(t.name));
    const args = await awaitTab(created, NI.Task.ANALOG_READ_TYPE);
    expect(args).toEqual({ taskKey: t.key });
  });

  describe("context menu", () => {
    it("opens the configuration tab from Edit configuration", async () => {
      const t = await createTask();
      const { created } = await renderToolbar();
      await openContextMenu(t.name);
      fireEvent.click(await screen.findByText("Edit configuration"));
      const args = await awaitTab(created, NI.Task.ANALOG_READ_TYPE);
      expect(args).toEqual({ taskKey: t.key });
    });

    it("issues a start command for a stopped task", async () => {
      const t = await createTask({ running: false });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        await renderToolbar();
        await openContextMenuUntil(t.name, "Start");
        fireEvent.click(screen.getByText("Start"));
        const cmd = await awaitCommand(streamer, t.key);
        expect(cmd.type).toBe("start");
      } finally {
        streamer.close();
      }
    });

    it("issues a stop command for a running task", async () => {
      const t = await createTask({ running: true });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        await renderToolbar();
        await openContextMenuUntil(t.name, "Stop");
        fireEvent.click(screen.getByText("Stop"));
        const cmd = await awaitCommand(streamer, t.key);
        expect(cmd.type).toBe("stop");
      } finally {
        streamer.close();
      }
    });

    it("enables data saving for a task whose config has it disabled", async () => {
      const t = await createTask({ config: { dataSaving: false } });
      await renderToolbar();
      await openContextMenuUntil(t.name, "Enable data saving");
      fireEvent.click(screen.getByText("Enable data saving"));
      await waitFor(async () => {
        const updated = await client.tasks.retrieve({ key: t.key });
        expect(updated.config).toEqual({ dataSaving: true });
      });
    });

    it("disables data saving for a task whose config has it enabled", async () => {
      const t = await createTask({ config: { dataSaving: true } });
      await renderToolbar();
      await openContextMenuUntil(t.name, "Disable data saving");
      fireEvent.click(screen.getByText("Disable data saving"));
      await waitFor(async () => {
        const updated = await client.tasks.retrieve({ key: t.key });
        expect(updated.config).toEqual({ dataSaving: false });
      });
    });

    it("hides the data saving items for a task without the config field", async () => {
      const t = await createTask();
      await renderToolbar();
      await openContextMenu(t.name);
      await screen.findByText("Edit configuration");
      expect(screen.queryByText("Enable data saving")).toBeNull();
      expect(screen.queryByText("Disable data saving")).toBeNull();
    });

    it("renames a stopped task through the inline editor", async () => {
      const t = await createTask();
      await renderToolbar();
      await openContextMenu(t.name);
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditing(`text-${t.key}`);
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.tasks.retrieve({ key: t.key })).name).toBe(renamed),
      );
    });

    it("asks for confirmation before renaming a running task", async () => {
      const t = await createTask({ running: true });
      await renderToolbar();
      await openContextMenuUntil(t.name, "Stop");
      fireEvent.click(screen.getByText("Rename"));
      const editor = await awaitTextEditing(`text-${t.key}`);
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await screen.findByText(
        `Are you sure you want to rename ${t.name} to ${renamed}?`,
      );
      fireEvent.click(findButton("Rename"));
      await waitFor(async () =>
        expect((await client.tasks.retrieve({ key: t.key })).name).toBe(renamed),
      );
    });

    it("deletes a task after confirmation", async () => {
      const t = await createTask();
      await renderToolbar();
      await openContextMenuUntil(t.name, "Delete");
      fireEvent.click(screen.getByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${t.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(async () => {
        await expect(client.tasks.retrieve({ key: t.key })).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
    });

    it("snapshots the task to the active range", async () => {
      const t = await createTask();
      const start = TimeStamp.now();
      const rng = await client.ranges.create({
        name: uniqueName("range"),
        timeRange: { start, end: start.add(TimeSpan.seconds(1)) },
      });
      const { store } = await renderToolbar();
      store.dispatch(Session.Range.add(Session.Range.fromClient(rng.payload)));
      store.dispatch(Session.Range.select(rng.key));
      await openContextMenu(t.name);
      fireEvent.click(await screen.findByText(`Snapshot to ${rng.name}`));
      await waitFor(async () => {
        const children = await client.ontology.children.retrieve({
          ids: rng.ontologyID,
        });
        expect(children.some((c) => c.id.type === "task")).toBe(true);
      });
    });
  });
});
