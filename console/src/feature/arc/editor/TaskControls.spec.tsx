// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, status, type Synnax as Client, task } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/lyra/status";
import { Arc } from "@synnaxlabs/pluto";
import { crdt, uuid } from "@synnaxlabs/x";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, Suspense } from "react";
import { describe, expect, it } from "vitest";

import { TaskControls } from "@/feature/arc/editor/TaskControls";
import { Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { createPanelWrapper } from "@/platform/panel/testutil";
import {
  awaitCommand,
  clickRedeploy,
  reportTaskRunning,
} from "@/platform/task/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  findDialogTrigger,
  findDialogTriggerByText,
  getIconButton,
  queryIcon,
  queryIconButton,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const createArcRack = async () =>
  await client.racks.create({ name: uniqueName("rack"), integrations: ["arc"] });

const renderControls = async (
  arcKey: arc.Key,
  as: Client = client,
  analytics: Analytics.Sink = Analytics.NOOP,
) => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  return await renderSuspended(
    <Analytics.Provider sink={analytics}>
      <Suspense fallback={null}>
        <Arc.Suspended arcKey={arcKey}>
          <TaskControls />
        </Arc.Suspended>
      </Suspense>
    </Analytics.Provider>,
    { wrapper },
  );
};

const searchAndClickRack = async (name: string): Promise<void> => {
  fireEvent.click(await findDialogTrigger());
  const search = await waitFor(() => screen.getByPlaceholderText("Search Drivers..."));
  fireEvent.change(search, { target: { value: name } });
  const option = await waitFor(() => screen.getByText(name));
  fireEvent.click(option);
};

// Renders the controls of an arc whose task has a status, in a tab that is visible or
// hidden, next to a hook that adds statuses and reads the notifications.
const renderInTab = async (visible: boolean) => {
  const rck = await createArcRack();
  const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
  const tsk = await client.arcs.updateTask(a.key, { rack: rck.key });
  if (tsk == null) throw new Error("expected a deployment task");
  const key = task.statusKey(tsk.key);
  await client.statuses.set(
    status.create<ReturnType<typeof task.statusDetailsZ>>({
      key,
      variant: "error",
      message: "Task failed",
      details: {
        task: tsk.key,
        running: false,
        cmd: "",
        configHash: "",
        rack: rck.key,
        data: {},
      },
    }),
  );
  const { wrapper: Panel, store } = await createPanelWrapper({
    client,
    panelKey: uuid.create(),
    tabKey: uuid.create(),
  });
  if (!visible) store.dispatch(Session.Panel.select({ key: uuid.create() }));
  const wrapper = ({ children }: PropsWithChildren) => (
    <Panel>
      <Suspense fallback={null}>
        <Arc.Suspended arcKey={a.key}>
          <TaskControls />
        </Arc.Suspended>
      </Suspense>
      {children}
    </Panel>
  );
  const { result } = renderHook(
    () => ({
      add: Status.useAdder(),
      notifications: Status.useNotifications().statuses,
    }),
    { wrapper },
  );
  await waitFor(() => expect(screen.getByText("Task failed")).toBeTruthy());
  const reply = () =>
    act(() =>
      result.current.add({ key, variant: "error", message: "Failed to start task" }),
    );
  return { result, reply };
};

describe("TaskControls", () => {
  it("should bind the rack when one is picked", async () => {
    const rck = await createArcRack();
    const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
    const { container } = await renderControls(a.key);
    await waitFor(() =>
      expect(getIconButton(container, "play").getAttribute("aria-disabled")).toBe(
        "true",
      ),
    );
    await searchAndClickRack(rck.name);
    await waitFor(async () => {
      const tsk = await client.arcs.task.retrieve(a.key);
      expect(tsk).not.toBeNull();
      expect(tsk?.rack).toBe(rck.key);
      expect(tsk?.config.arcKey).toBe(a.key);
    });
    await waitFor(() =>
      expect(getIconButton(container, "play").getAttribute("aria-disabled")).toBeNull(),
    );
  });

  it("should start the arc from the play button", async () => {
    const rck = await createArcRack();
    const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
    const tsk = await client.arcs.updateTask(a.key, { rack: rck.key });
    if (tsk == null) throw new Error("expected a deployment task");
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    try {
      const { container } = await renderControls(a.key);
      await waitFor(() =>
        expect(
          getIconButton(container, "play").getAttribute("aria-disabled"),
        ).toBeNull(),
      );
      fireEvent.click(getIconButton(container, "play"));
      const cmd = await awaitCommand(streamer, tsk.key);
      expect(cmd.type).toBe("start");
    } finally {
      streamer.close();
    }
  });

  it("should report the automation it deployed", async () => {
    const rck = await createArcRack();
    const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
    await client.arcs.updateTask(a.key, { rack: rck.key });
    const analytics = createTestSink();
    const { container } = await renderControls(a.key, client, analytics);
    await waitFor(() =>
      expect(getIconButton(container, "play").getAttribute("aria-disabled")).toBeNull(),
    );
    fireEvent.click(getIconButton(container, "play"));
    await waitFor(() =>
      expect(analytics.capture).toHaveBeenCalledWith("automation_deployed", {}),
    );
  });

  it("should offer a start that picks up the synced config when the arc drifts", async () => {
    const rck = await createArcRack();
    const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
    const tsk = await client.arcs.updateTask(a.key, { rack: rck.key });
    if (tsk == null) throw new Error("expected a deployment task");
    await reportTaskRunning(client, tsk);
    const gen = new crdt.Text(2);
    await client.arcs.dispatch(
      a.key,
      gen.insert(0, "a -> b").map((op) => arc.insertChar(op)),
    );
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    try {
      await renderControls(a.key);
      await clickRedeploy();
      const cmd = await awaitCommand(streamer, tsk.key);
      expect(cmd.type).toBe("start");
    } finally {
      streamer.close();
    }
    await waitFor(async () => {
      const restamped = await client.tasks.retrieve(tsk.key);
      expect(restamped.config.hash).not.toEqual(tsk.config.hash);
    });
  });

  describe("performance", () => {
    it("should disable the select while no rack is bound", async () => {
      const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
      await renderControls(a.key);
      const trigger = await findDialogTriggerByText("Auto");
      expect(trigger.getAttribute("aria-disabled")).toBe("true");
    });

    it("should show the stored level with its glyph as the trigger icon", async () => {
      const rck = await createArcRack();
      const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
      await client.arcs.updateTask(a.key, { rack: rck.key, performance: "medium" });
      await renderControls(a.key);
      const trigger = await findDialogTriggerByText("Medium");
      expect(queryIcon(trigger, "performance-medium")).toBe(trigger.firstElementChild);
    });

    it("should set the level of a bound arc", async () => {
      const rck = await createArcRack();
      const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
      await client.arcs.updateTask(a.key, { rack: rck.key });
      await renderControls(a.key);
      const trigger = await findDialogTriggerByText("Auto");
      await waitFor(() => expect(trigger.getAttribute("aria-disabled")).toBeNull());
      fireEvent.click(trigger);
      fireEvent.click(await screen.findByText("High"));
      await waitFor(async () => {
        const tsk = await client.arcs.task.retrieve(a.key);
        expect(tsk?.config.performance).toBe("high");
        expect(tsk?.rack).toBe(rck.key);
      });
    });
  });

  describe("notifications", () => {
    it("should not notify a reply that the controls of a visible tab show", async () => {
      const { result, reply } = await renderInTab(true);
      reply();
      expect(result.current.notifications).toEqual([]);
    });

    it("should notify a reply on an arc whose tab is not visible", async () => {
      const { result, reply } = await renderInTab(false);
      reply();
      await waitFor(() =>
        expect(result.current.notifications).toEqual([
          expect.objectContaining({ message: "Failed to start task" }),
        ]),
      );
    });
  });

  describe("without permission to command the task", () => {
    it("should show the status without any action", async () => {
      const rck = await createArcRack();
      const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
      await client.arcs.updateTask(a.key, { rack: rck.key });
      const viewer = await createTestClientWithRole(client, "Viewer");
      const { container } = await renderControls(a.key, viewer);
      await waitFor(() =>
        expect(container.querySelector(".console-arc-editor__controls")).not.toBeNull(),
      );
      expect(queryIconButton(container, "play")).toBeNull();
      expect(queryIconButton(container, "stop")).toBeNull();
    });
  });
});
