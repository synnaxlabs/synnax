// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  arc,
  NotFoundError,
  type panel,
  type Synnax as Client,
  task,
} from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, assert, describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import {
  awaitCommand,
  awaitTaskResourceTab,
  type CreatedPanel,
  createSelectedPanel,
  createTaskStatus,
} from "@/platform/task/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import {
  assertDefined,
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  createKeyEndingIn,
  getIconButton,
  openContextMenu,
  queryIcon,
  queryIconButton,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

interface CreateTaskOptions {
  config?: Record<string, unknown>;
  running?: boolean;
  type?: string;
  drifted?: boolean;
}

// The toolbar lists every task on the Core, so leaked tasks from prior tests slow each
// render. Track created resources and remove them after each test.
const createdTasks: task.Key[] = [];
const createdRacks: number[] = [];
const createdArcs: arc.Key[] = [];

// Deletion tests already removed their task, so a NotFound here is expected.
const ignoreNotFound = (err: unknown) => {
  if (!NotFoundError.matches(err)) console.error(err);
};

afterEach(async () => {
  // An Arc takes its task with it.
  await client.arcs.delete(createdArcs.splice(0)).catch(ignoreNotFound);
  await client.tasks.delete(createdTasks.splice(0)).catch(ignoreNotFound);
  await client.racks.delete(createdRacks.splice(0)).catch(ignoreNotFound);
});

const setRunning = async (t: task.Task, running: boolean, drifted = false) =>
  await client.statuses.set(
    createTaskStatus({
      key: task.statusKey(t.key),
      details: {
        task: t.key,
        running,
        configHash: drifted ? "stale" : t.configHash,
        rack: t.rack,
      },
    }),
  );

const createTask = async ({
  config = {},
  running,
  type = NI.Task.ANALOG_READ_TYPE,
  drifted,
}: CreateTaskOptions = {}) => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  createdRacks.push(rack.key);
  // The toolbar renders only the first rows of every task on the Core, so the key
  // sorts this task first.
  const t = await rack.createTask({
    key: createKeyEndingIn("0"),
    name: uniqueName("task"),
    type,
    config,
  });
  createdTasks.push(t.key);
  if (running != null) await setRunning(t, running, drifted);
  return t;
};

// Binding an Arc to a rack creates its task, named after the Arc.
const createArcTask = async () => {
  const rack = await client.racks.create({
    name: uniqueName("rack"),
    integrations: ["arc"],
  });
  createdRacks.push(rack.key);
  const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
  createdArcs.push(a.key);
  const t = await client.arcs.updateTask(a.key, { rack: rack.key });
  assertDefined(t, "binding the Arc created no task");
  return { arc: a, task: t };
};

const getRow = (name: string): HTMLElement => {
  const row = screen.getByText(name).closest<HTMLElement>(".lyra-list__item");
  assertDefined(row, `no list item for ${name}`);
  return row;
};

// A connected Core, so links to its cluster can be copied.
const CORE = createCore("Core");

const renderToolbar = async (
  as: Client = client,
  analytics: Analytics.Sink = Analytics.NOOP,
) => {
  const { wrapper, store } = await createConsoleWrapper({
    client: as,
    preloadedState: createCoreState([CORE], CORE.key),
  });
  const created = await createSelectedPanel(store, client);
  render(
    <Analytics.Provider sink={analytics}>
      <Task.RegistryProvider registry={Task.REGISTRY}>
        {Task.TOOLBAR.content}
        <Modals.Stack />
      </Task.RegistryProvider>
    </Analytics.Provider>,
    { wrapper },
  );
  return { created, store };
};

const awaitTab = async (created: CreatedPanel, type: string): Promise<void> =>
  await waitFor(async () => {
    const doc = await client.panels.retrieve(created.panelKey);
    assert(doc.root.variant === "leaf", "panel root is not a leaf");
    const tab = doc.root.tabs.find(
      (t): t is panel.ViewTab => t.variant === "view" && t.type === type,
    );
    assertDefined(tab, `no ${type} tab was opened`);
  });

// The task's status streams in after the list fetch, so wait for the item it gates.
const openContextMenuUntil = async (name: string, text: string): Promise<void> => {
  await openContextMenu(name);
  await screen.findByText(text);
};

describe("task/Toolbar", () => {
  it("renders a created task with its parsed type", async () => {
    const t = await createTask();
    await renderToolbar();
    expect(await screen.findByText(t.name)).toBeTruthy();
    expect((await screen.findAllByText("NI analog read task")).length).toBeGreaterThan(
      0,
    );
  });

  it("withholds the empty message until the task list answers", async () => {
    const t = await createTask();
    // The task cache hangs off the client, so a client an earlier test left holding a
    // live empty list would answer the toolbar before the fetch. A fresh client is
    // cold, which is the cold-start state this guard exists for.
    await renderToolbar(createTestClient());
    expect(screen.queryByText("No tasks")).toBeNull();
    await screen.findByText(t.name);
  });

  it("opens the task selector tab from the create action", async () => {
    const { created } = await renderToolbar();
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    await awaitTab(created, "taskSelector");
  });

  it("opens the task's resource tab on double click", async () => {
    const t = await createTask();
    const { created } = await renderToolbar();
    await screen.findByText(t.name);
    // Re-query synchronously: async status/permission resolutions can replace the
    // text node, detaching a match held across an await before the event lands.
    fireEvent.doubleClick(screen.getByText(t.name));
    expect(await awaitTaskResourceTab(created)).toBe(t.key);
  });

  describe("data saving button", () => {
    it("disables data saving for a task whose config has it enabled", async () => {
      const t = await createTask({ config: { dataSavingDisabled: false } });
      await renderToolbar();
      await screen.findByText(t.name);
      fireEvent.click(
        await within(getRow(t.name)).findByRole("button", {
          name: "Data saving enabled. Click to disable.",
        }),
      );
      await waitFor(async () => {
        const updated = await client.tasks.retrieve(t.key);
        expect(updated.config).toMatchObject({ dataSavingDisabled: true });
      });
    });

    it("enables data saving for a task whose config has it disabled", async () => {
      const t = await createTask({ config: { dataSavingDisabled: true } });
      await renderToolbar();
      await screen.findByText(t.name);
      fireEvent.click(
        await within(getRow(t.name)).findByRole("button", {
          name: "Data saving disabled. Click to enable.",
        }),
      );
      await waitFor(async () => {
        const updated = await client.tasks.retrieve(t.key);
        expect(updated.config).toMatchObject({ dataSavingDisabled: false });
      });
    });

    it("hides the button for a task without the config field", async () => {
      const t = await createTask({ type: "pagerduty_alert" });
      await renderToolbar();
      await screen.findByText(t.name);
      expect(
        within(getRow(t.name)).queryByRole("button", { name: /^Data saving/ }),
      ).toBeNull();
    });

    it("leaves the config unchanged when a viewer clicks the button", async () => {
      const t = await createTask({ config: { dataSavingDisabled: false } });
      const viewer = await roles.get("Viewer");
      await renderToolbar(viewer);
      await screen.findByText(t.name);
      fireEvent.click(
        await within(getRow(t.name)).findByRole("button", {
          name: "Data saving enabled",
        }),
      );
      const updated = await client.tasks.retrieve(t.key);
      expect(updated.config).toMatchObject({ dataSavingDisabled: false });
    });
  });

  describe("context menu", () => {
    it("opens the resource tab from Edit configuration", async () => {
      const t = await createTask();
      const { created } = await renderToolbar();
      await openContextMenu(t.name);
      fireEvent.click(await screen.findByText("Edit configuration"));
      expect(await awaitTaskResourceTab(created)).toBe(t.key);
    });

    it("offers Reload Console", async () => {
      const t = await createTask();
      await renderToolbar();
      await openContextMenu(t.name);
      expect(await screen.findByText("Reload Console")).toBeTruthy();
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

    it("reports the type of a task it started", async () => {
      const t = await createTask({ running: false });
      const analytics = createTestSink();
      await renderToolbar(client, analytics);
      await openContextMenuUntil(t.name, "Start");
      fireEvent.click(screen.getByText("Start"));
      await waitFor(() =>
        expect(analytics.capture).toHaveBeenCalledWith("task_started", {
          type: NI.Task.ANALOG_READ_TYPE,
        }),
      );
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

    it("updates an open menu when the task starts running", async () => {
      const t = await createTask({ running: false });
      await renderToolbar();
      await openContextMenuUntil(t.name, "Start");
      await setRunning(t, true);
      expect(await screen.findByText("Stop")).toBeTruthy();
      expect(screen.queryByText("Start")).toBeNull();
    });

    it("redeploys a drifted running task from the context menu", async () => {
      const t = await createTask({ running: true, drifted: true });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        await renderToolbar();
        await openContextMenuUntil(t.name, "Redeploy");
        fireEvent.click(screen.getByText("Redeploy"));
        const cmd = await awaitCommand(streamer, t.key);
        expect(cmd.type).toBe("start");
      } finally {
        streamer.close();
      }
    });

    it("hides Redeploy when the running instance matches the stored config", async () => {
      const t = await createTask({ running: true });
      await renderToolbar();
      await openContextMenuUntil(t.name, "Stop");
      expect(screen.queryByText("Redeploy")).toBeNull();
    });

    it("enables data saving for a task whose config has it disabled", async () => {
      const t = await createTask({ config: { dataSavingDisabled: true } });
      await renderToolbar();
      await openContextMenuUntil(t.name, "Enable data saving");
      fireEvent.click(screen.getByText("Enable data saving"));
      await waitFor(async () => {
        const updated = await client.tasks.retrieve(t.key);
        expect(updated.config).toMatchObject({ dataSavingDisabled: false });
      });
    });

    it("disables data saving for a task whose config has it enabled", async () => {
      const t = await createTask({ config: { dataSavingDisabled: false } });
      await renderToolbar();
      await openContextMenuUntil(t.name, "Disable data saving");
      fireEvent.click(screen.getByText("Disable data saving"));
      await waitFor(async () => {
        const updated = await client.tasks.retrieve(t.key);
        expect(updated.config).toMatchObject({ dataSavingDisabled: true });
      });
    });

    it("hides the data saving items for a task without the config field", async () => {
      const t = await createTask({ type: "pagerduty_alert" });
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
        expect((await client.tasks.retrieve(t.key)).name).toBe(renamed),
      );
    });

    it("renames a running task without asking for confirmation", async () => {
      const t = await createTask({ running: true });
      await renderToolbar();
      await openContextMenuUntil(t.name, "Stop");
      fireEvent.click(screen.getByText("Rename"));
      const editor = await awaitTextEditing(`text-${t.key}`);
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.tasks.retrieve(t.key)).name).toBe(renamed),
      );
      expect(screen.queryByText(/Are you sure you want to rename/)).toBeNull();
    });

    it("deletes a task after confirmation", async () => {
      const t = await createTask();
      await renderToolbar();
      await openContextMenuUntil(t.name, "Delete");
      fireEvent.click(screen.getByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${t.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(async () => {
        await expect(client.tasks.retrieve(t.key)).rejects.toSatisfy((e) =>
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
      const snapshots = await waitFor(async () => {
        const children = await client.ontology.children.retrieve({
          ids: rng.ontologyID,
        });
        const tasks = children.filter((c) => c.id.type === "task");
        expect(tasks).not.toHaveLength(0);
        return tasks;
      });
      // The snapshot is a task on the same rack, which cannot go while it remains.
      createdTasks.push(...snapshots.map((c) => c.id.key));
    });
  });

  describe("Arc tasks", () => {
    it("lists an Arc bound to a rack under the Arc icon", async () => {
      const { arc: a } = await createArcTask();
      await renderToolbar();
      await screen.findByText(a.name);
      const row = getRow(a.name);
      expect(within(row).getByText("Arc task")).toBeTruthy();
      expect(queryIcon(row, "arc")).not.toBeNull();
    });

    it("withholds the data saving button and items from an Arc task", async () => {
      const { arc: a } = await createArcTask();
      await renderToolbar();
      await screen.findByText(a.name);
      expect(
        within(getRow(a.name)).queryByRole("button", { name: /^Data saving/ }),
      ).toBeNull();
      await openContextMenu(a.name);
      await screen.findByText("Edit configuration");
      expect(screen.queryByText("Enable data saving")).toBeNull();
      expect(screen.queryByText("Disable data saving")).toBeNull();
    });

    it("opens the Arc editor on double click", async () => {
      const { arc: a } = await createArcTask();
      const { store } = await renderToolbar();
      await screen.findByText(a.name);
      fireEvent.doubleClick(screen.getByText(a.name));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("focused tab is not a resource");
      expect(tab.resource).toEqual(arc.ontologyID(a.key));
    });

    it("starts the automation from the row's play button", async () => {
      const { arc: a, task: t } = await createArcTask();
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const analytics = createTestSink();
        await renderToolbar(client, analytics);
        await screen.findByText(a.name);
        const play = await waitFor(() => getIconButton(getRow(a.name), "play"));
        fireEvent.click(play);
        const cmd = await awaitCommand(streamer, t.key);
        expect(cmd.type).toBe("start");
        await waitFor(() =>
          expect(analytics.capture).toHaveBeenCalledWith("automation_deployed", {}),
        );
      } finally {
        streamer.close();
      }
    });

    it("stops the automation from the row's stop button", async () => {
      const { arc: a, task: t } = await createArcTask();
      await setRunning(t, true);
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        await renderToolbar();
        await screen.findByText(a.name);
        const stop = await waitFor(() => getIconButton(getRow(a.name), "stop"));
        fireEvent.click(stop);
        const cmd = await awaitCommand(streamer, t.key);
        expect(cmd.type).toBe("stop");
      } finally {
        streamer.close();
      }
    });

    it("copies a link to the Arc from the context menu", async () => {
      const writeText = stubClipboardWriteText();
      const { arc: a } = await createArcTask();
      await renderToolbar();
      await openContextMenu(a.name);
      fireEvent.click(await screen.findByText("Copy link"));
      await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
      expect(writeText.mock.calls[0][0]).toBe(
        `synnax://cluster/${CORE.clusterKey}/arc/${a.key}`,
      );
    });

    it("renames the Arc, and so its task, through the inline editor", async () => {
      const { arc: a, task: t } = await createArcTask();
      await renderToolbar();
      await openContextMenu(a.name);
      fireEvent.click(await screen.findByText("Rename"));
      const editor = await awaitTextEditing(`text-${t.key}`);
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.arcs.retrieve(a.key)).name).toBe(renamed),
      );
      await waitFor(async () =>
        expect((await client.tasks.retrieve(t.key)).name).toBe(renamed),
      );
    });
  });

  describe("permission to command tasks", () => {
    const createStoppedTask = async () => await createTask({ running: false });

    it("withholds the start/stop button and its items from a viewer", async () => {
      const t = await createStoppedTask();
      const viewer = await roles.get("Viewer");
      await renderToolbar(viewer);
      await screen.findByText(t.name);
      expect(queryIconButton(document.body, "play")).toBeNull();
      await openContextMenu(t.name);
      expect(screen.queryByText("Start")).toBeNull();
      expect(screen.queryByText("Stop")).toBeNull();
    });

    it("withholds Edit configuration from an operator", async () => {
      const t = await createStoppedTask();
      const operator = await roles.get("Operator");
      await renderToolbar(operator);
      await screen.findByText(t.name);
      await openContextMenuUntil(t.name, "Start");
      expect(screen.queryByText("Edit configuration")).toBeNull();
    });

    it("offers the start/stop button and its items to an operator", async () => {
      const t = await createStoppedTask();
      const operator = await roles.get("Operator");
      await renderToolbar(operator);
      await screen.findByText(t.name);
      await waitFor(() =>
        expect(queryIconButton(document.body, "play")).not.toBeNull(),
      );
      await openContextMenu(t.name);
      expect(await screen.findByText("Start")).toBeTruthy();
    });
  });
});
