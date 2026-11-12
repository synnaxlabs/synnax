// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  arc,
  channel,
  device,
  NotFoundError,
  panel,
  project,
  query,
  rack,
  ranger,
  type resource,
  type Synnax as Client,
  task,
  user,
} from "@synnaxlabs/client";
import {
  type BuiltInRole,
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, assert, describe, expect, it, onTestFinished } from "vitest";

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
import {
  assertDefined,
  awaitTextEditing,
  commitTextEdit,
  createConsoleWrapper,
  getIconButton,
  openContextMenu,
  queryIconButton,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

interface CreateTaskOptions {
  config?: Record<string, unknown>;
  running?: boolean;
  type?: string;
  drifted?: boolean;
}

const createdTasks: task.Key[] = [];
const createdRacks: number[] = [];
const createdArcs: arc.Key[] = [];
// The tasks of the running test, Arc tasks included.
const testTasks: task.Key[] = [];

// Deletion tests already removed their task, so a NotFound here is expected.
const ignoreNotFound = (err: unknown) => {
  if (!NotFoundError.matches(err)) console.error(err);
};

afterEach(async () => {
  testTasks.splice(0);
  await client.arcs.delete(createdArcs.splice(0)).catch(ignoreNotFound);
  await client.tasks.delete(createdTasks.splice(0)).catch(ignoreNotFound);
  await client.racks.delete(createdRacks.splice(0)).catch(ignoreNotFound);
});

const setRunning = async (t: task.Task, running: boolean, drifted = false) =>
  await client.statuses.set(
    createTaskStatus({
      owner: task.resourceID(t.key),
      details: {
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
  const t = await rack.createTask({
    name: uniqueName("task"),
    type,
    config,
  });
  createdTasks.push(t.key);
  testTasks.push(t.key);
  if (running != null) await setRunning(t, running, drifted);
  return t;
};

const isTaskType = ({ type, key }: resource.ID): boolean =>
  type === task.TYPE_RESOURCE_ID.type && key === task.TYPE_RESOURCE_ID.key;

// The toolbar lists every task its user can read, and other spec files leave tasks on
// the Core. The policies of a built-in role, with the reading of tasks narrowed to the
// tasks of the test, show the same rows on any Core.
const createTestTasksClient = async (roleName: BuiltInRole): Promise<Client> => {
  const roles = await client.access.roles.retrieve({});
  const role = roles.find((r) => r.name === roleName);
  assertDefined(role, `no built-in role named ${roleName}`);
  const policies = await client.access.policies.retrieve({ roles: [role.key] });
  return await createTestClientWithPolicy(
    client,
    policies.flatMap(({ objects, actions }) => {
      const name = uniqueName("policy");
      if (!objects.some(isTaskType)) return [{ name, objects, actions }];
      const narrowed = {
        name,
        objects: [
          ...objects.filter((o) => !isTaskType(o)),
          ...task.resourceID(testTasks),
        ],
        actions,
      };
      const writes = actions.filter((a) => a !== "retrieve");
      if (writes.length === 0) return [narrowed];
      const typeWrites = {
        name: uniqueName("policy"),
        objects: [task.TYPE_RESOURCE_ID],
        actions: writes,
      };
      return [narrowed, typeWrites];
    }),
  );
};

interface RenderToolbarOptions {
  as?: Client | BuiltInRole;
  analytics?: Analytics.Sink;
}

// A client keeps its list answer after the list unmounts and serves it without a fetch,
// so each render takes a new client.
const renderToolbar = async ({
  as,
  analytics = Analytics.NOOP,
}: RenderToolbarOptions = {}) => {
  if (typeof as !== "object") as = await createTestTasksClient(as ?? "Owner");
  onTestFinished(async () => await as.close());
  await as.connect();
  const { wrapper, store } = await createConsoleWrapper({ client: as });
  const created = await createSelectedPanel(store, client);
  // The panel's create broadcast may not have reached `as`, and opening a tab reads it.
  onTestFinished(as.panels.watch(created.panelKey).onChange(() => {}));
  await as.panels.retrieve(created.panelKey);
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

const createArc = async (): Promise<arc.Arc> => {
  const a = await client.arcs.create({
    name: uniqueName("arc"),
    type: "automation",
    source: { mode: "text" },
  });
  createdArcs.push(a.key);
  return a;
};

// Binding an Arc to a rack gives it a task.
const createBoundArc = async (): Promise<{ arc: arc.Arc; task: task.Task }> => {
  const a = await createArc();
  const rack = await client.racks.create({ name: uniqueName("rack") });
  createdRacks.push(rack.key);
  const t = await client.arcs.updateTask(a.key, { rack: rack.key });
  assertDefined(t, "binding the Arc to a rack made no task");
  testTasks.push(t.key);
  return { arc: a, task: t };
};

// Returns the key of the first Arc tab in the panel.
const awaitArcTab = async ({ client, panelKey }: CreatedPanel): Promise<arc.Key> =>
  await waitFor(() => {
    const doc = client.panels.watch(panelKey).getCached();
    if (!query.isLive(doc) || doc.root.variant !== "leaf")
      throw new Error("panel doc not found");
    const match = doc.root.tabs.find(
      (t) => t.variant === "resource" && t.resource.type === "arc",
    );
    if (match?.variant !== "resource") throw new Error("Arc tab not found");
    return match.resource.key;
  });

const tabCount = async ({ client, panelKey }: CreatedPanel): Promise<number> => {
  const doc = await client.panels.retrieve(panelKey);
  if (doc.root.variant !== "leaf") throw new Error("expected a leaf panel root");
  return doc.root.tabs.length;
};

describe("task/Toolbar Arc rows", () => {
  it("shows an Arc that has a task with the Arc icon and its type", async () => {
    const { arc: a } = await createBoundArc();
    await renderToolbar();
    const row = (await screen.findByText(a.name)).closest(".lyra-list__item");
    assertDefined(row, `no list item for ${a.name}`);
    expect(row.querySelector(".lyra-icon--arc")).not.toBeNull();
    expect(row.textContent).toContain("Arc automation task");
    expect(row.textContent).not.toContain("Stopped");
  });

  it("leaves out an Arc that has no task", async () => {
    const unbound = await createArc();
    const { arc: bound } = await createBoundArc();
    await renderToolbar();
    await screen.findByText(bound.name);
    expect(screen.queryByText(unbound.name)).toBeNull();
  });

  it("opens the Arc editor on double click", async () => {
    const { arc: a } = await createBoundArc();
    const { created } = await renderToolbar();
    await screen.findByText(a.name);
    fireEvent.doubleClick(screen.getByText(a.name));
    expect(await awaitArcTab(created)).toBe(a.key);
  });

  it("starts the Arc from its start button", async () => {
    const { arc: a, task: t } = await createBoundArc();
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    try {
      await renderToolbar();
      await screen.findByText(a.name);
      const start = await waitFor(() => {
        const row = document.getElementById(t.key);
        if (row == null) throw new Error(`no list item for ${a.name}`);
        return getIconButton(row, "play");
      });
      fireEvent.click(start);
      expect((await awaitCommand(streamer, t.key)).type).toBe("start");
    } finally {
      streamer.close();
    }
  });

  it("keeps the editor closed when the start button is double clicked", async () => {
    const { arc: started, task: t } = await createBoundArc();
    const { arc: opened } = await createBoundArc();
    const { created } = await renderToolbar();
    await screen.findByText(started.name);
    await screen.findByText(opened.name);
    // The start button mounts only once the update permission resolves, so poll the
    // row for it instead of reading it off the first render.
    const start = await waitFor(() => {
      const row = document.getElementById(t.key);
      if (row == null) throw new Error(`no list item for ${started.name}`);
      return getIconButton(row, "play");
    });
    fireEvent.doubleClick(start);
    // Opening the other Arc gives a leaked double click time to land, so the tab count
    // below is not read before the editor would have appeared.
    fireEvent.doubleClick(screen.getByText(opened.name));
    expect(await awaitArcTab(created)).toBe(opened.key);
    expect(await tabCount(created)).toBe(1);
  });

  describe("context menu", () => {
    it("opens the Arc editor from Edit", async () => {
      const { arc: a } = await createBoundArc();
      const { created } = await renderToolbar();
      await openContextMenuUntil(a.name, "Edit");
      fireEvent.click(screen.getByText("Edit"));
      expect(await awaitArcTab(created)).toBe(a.key);
    });

    it("renames the Arc through the inline editor", async () => {
      const { arc: a } = await createBoundArc();
      await renderToolbar();
      await openContextMenuUntil(a.name, "Rename");
      fireEvent.click(screen.getByText("Rename"));
      const editor = await awaitTextEditing(`text-${a.key}`);
      const renamed = uniqueName("renamed");
      commitTextEdit(editor, renamed);
      await waitFor(async () =>
        expect((await client.arcs.retrieve({ key: a.key })).name).toBe(renamed),
      );
    });

    it("deletes the Arc after confirmation", async () => {
      const { arc: a } = await createBoundArc();
      await renderToolbar();
      await openContextMenuUntil(a.name, "Delete");
      fireEvent.click(screen.getByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${a.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(async () => {
        await expect(client.arcs.retrieve({ key: a.key })).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
    });
  });

  describe("selection with a task", () => {
    // A click can land before the list knows the row, so it repeats until the row
    // reads as selected. Control adds the row to the selection.
    const selectRow = async (name: string, control: boolean): Promise<void> => {
      await screen.findByText(name);
      await waitFor(() => {
        const row = screen.getByText(name).closest(".lyra-list__item");
        assertDefined(row, `no list item for ${name}`);
        if (row.getAttribute("aria-selected") !== "true") {
          if (control) fireEvent.keyDown(document.body, { code: "Control" });
          fireEvent.click(screen.getByText(name));
          if (control) fireEvent.keyUp(document.body, { code: "Control" });
        }
        expect(row.getAttribute("aria-selected")).toBe("true");
      });
    };

    const selectRows = async (first: string, ...others: string[]): Promise<void> => {
      await selectRow(first, false);
      for (const name of others) await selectRow(name, true);
    };

    it("starts the Arc and the task, reporting each", async () => {
      const { arc: a } = await createBoundArc();
      const t = await createTask({ running: false });
      const analytics = createTestSink();
      await renderToolbar({ analytics });
      await selectRows(t.name, a.name);
      await openContextMenuUntil(a.name, "Start");
      fireEvent.click(screen.getByText("Start"));
      await waitFor(() => {
        expect(analytics.capture).toHaveBeenCalledWith("task_started", {
          type: NI.Task.ANALOG_READ_TYPE,
        });
        expect(analytics.capture).toHaveBeenCalledWith("automation_deployed", {});
      });
    });

    it("deletes the Arc and the task after one confirmation", async () => {
      const { arc: a } = await createBoundArc();
      const t = await createTask();
      await renderToolbar();
      await selectRows(t.name, a.name);
      await openContextMenuUntil(a.name, "Delete");
      fireEvent.click(screen.getByText("Delete"));
      await screen.findByRole("dialog");
      fireEvent.click(findButton("Delete"));
      await waitFor(async () => {
        await expect(client.arcs.retrieve({ key: a.key })).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
        await expect(client.tasks.retrieve(t.key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
    });

    it("hides Delete when the Arc cannot be deleted", async () => {
      const { arc: a } = await createBoundArc();
      const t = await createTask();
      const taskDeleter = await createTestClientWithPolicy(client, [
        {
          name: uniqueName("retrieve"),
          objects: [
            ...task.resourceID(testTasks),
            arc.TYPE_RESOURCE_ID,
            arc.directory.TYPE_RESOURCE_ID,
            rack.TYPE_RESOURCE_ID,
            device.TYPE_RESOURCE_ID,
            channel.TYPE_RESOURCE_ID,
            ranger.TYPE_RESOURCE_ID,
            panel.TYPE_RESOURCE_ID,
            project.TYPE_RESOURCE_ID,
            user.TYPE_RESOURCE_ID,
            access.role.TYPE_RESOURCE_ID,
            access.policy.TYPE_RESOURCE_ID,
          ],
          actions: ["retrieve"],
        },
        {
          name: uniqueName("delete"),
          objects: [task.TYPE_RESOURCE_ID],
          actions: ["delete"],
        },
      ]);
      await renderToolbar({ as: taskDeleter });
      // The client sees Delete on the task alone, so its grants have loaded.
      await openContextMenuUntil(t.name, "Delete");
      fireEvent.keyDown(document.body, { code: "Escape" });
      await waitFor(() =>
        expect(document.querySelector(".lyra-menu-context")).toBeNull(),
      );
      await selectRows(t.name, a.name);
      await openContextMenuUntil(a.name, "Reload Console");
      expect(screen.queryByText("Delete")).toBeNull();
    });

    it("leaves the Arc out of a snapshot to the active range", async () => {
      const { arc: a } = await createBoundArc();
      const t = await createTask();
      const start = TimeStamp.now();
      const rng = await client.ranges.create({
        name: uniqueName("range"),
        timeRange: { start, end: start.add(TimeSpan.seconds(1)) },
      });
      const { store } = await renderToolbar();
      store.dispatch(Session.Range.add(Session.Range.fromClient(rng.payload)));
      store.dispatch(Session.Range.select(rng.key));
      await selectRows(t.name, a.name);
      await openContextMenuUntil(a.name, `Snapshot to ${rng.name}`);
      fireEvent.click(screen.getByText(`Snapshot to ${rng.name}`));
      const snapshots = await waitFor(async () => {
        const tasks = await client.tasks.retrieve({ ranges: [rng.key] });
        expect(tasks).not.toHaveLength(0);
        return tasks;
      });
      createdTasks.push(...snapshots.map((s) => s.key));
      expect(snapshots.map(({ type }) => type)).toEqual([NI.Task.ANALOG_READ_TYPE]);
    });
  });
});

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
    await renderToolbar();
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
    const getRow = (name: string): HTMLElement => {
      const row = screen.getByText(name).closest<HTMLElement>(".lyra-list__item");
      assertDefined(row, `no list item for ${name}`);
      return row;
    };

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
      await renderToolbar({ as: "Viewer" });
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
      await renderToolbar({ analytics });
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
        const tasks = await client.tasks.retrieve({ ranges: [rng.key] });
        expect(tasks).not.toHaveLength(0);
        return tasks;
      });
      // The snapshot is a task on the same rack, which cannot go while it remains.
      createdTasks.push(...snapshots.map((s) => s.key));
    });
  });

  describe("permission to command tasks", () => {
    const createStoppedTask = async () => await createTask({ running: false });

    it("withholds the start/stop button and its items from a viewer", async () => {
      const t = await createStoppedTask();
      await renderToolbar({ as: "Viewer" });
      await screen.findByText(t.name);
      expect(queryIconButton(document.body, "play")).toBeNull();
      await openContextMenu(t.name);
      expect(screen.queryByText("Start")).toBeNull();
      expect(screen.queryByText("Stop")).toBeNull();
    });

    it("withholds Edit configuration from an operator", async () => {
      const t = await createStoppedTask();
      await renderToolbar({ as: "Operator" });
      await screen.findByText(t.name);
      await openContextMenuUntil(t.name, "Start");
      expect(screen.queryByText("Edit configuration")).toBeNull();
    });

    it("offers the start/stop button and its items to an operator", async () => {
      const t = await createStoppedTask();
      await renderToolbar({ as: "Operator" });
      await screen.findByText(t.name);
      await waitFor(() =>
        expect(queryIconButton(document.body, "play")).not.toBeNull(),
      );
      await openContextMenu(t.name);
      expect(await screen.findByText("Start")).toBeTruthy();
    });
  });
});
