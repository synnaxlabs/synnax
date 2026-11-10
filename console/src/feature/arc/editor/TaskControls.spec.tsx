// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type Synnax as Client, task } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { Arc } from "@synnaxlabs/pluto";
import { crdt } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { describe, expect, it } from "vitest";

import { TaskControls } from "@/feature/arc/editor/TaskControls";
import { Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import {
  awaitCommand,
  clickRedeploy,
  reportTaskRunning,
} from "@/platform/task/testutil";
import {
  collapseSelectTrigger,
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

// The Core rewrites the task on its resync debounce, so the button takes a moment.
const hoverRedeploy = async (): Promise<void> => {
  const button = await waitFor(() => screen.getByRole("button", { name: "Redeploy" }), {
    timeout: 5000,
  });
  fireEvent.pointerOver(button, { pointerType: "mouse" });
};

const searchAndClickRack = async (name: string): Promise<void> => {
  fireEvent.click(await findDialogTrigger());
  const search = await waitFor(() => screen.getByPlaceholderText("Search drivers..."));
  fireEvent.change(search, { target: { value: name } });
  const option = await waitFor(() => screen.getByText(name));
  fireEvent.click(option);
};

describe("TaskControls", () => {
  it("should bind the rack when one is picked", async () => {
    const rck = await createArcRack();
    const a = await client.arcs.create({
      name: uniqueName("arc"),
      type: "automation",
      source: { mode: "text" },
    });
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
    const a = await client.arcs.create({
      name: uniqueName("arc"),
      type: "automation",
      source: { mode: "text" },
    });
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
    const a = await client.arcs.create({
      name: uniqueName("arc"),
      type: "automation",
      source: { mode: "text" },
    });
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
    const a = await client.arcs.create({
      name: uniqueName("arc"),
      type: "automation",
      source: { mode: "text" },
    });
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

  it("should say the automation was edited in the redeploy tooltip", async () => {
    const rck = await createArcRack();
    const a = await client.arcs.create({
      name: uniqueName("arc"),
      type: "automation",
      source: { mode: "text" },
    });
    const tsk = await client.arcs.updateTask(a.key, { rack: rck.key });
    if (tsk == null) throw new Error("expected a deployment task");
    await reportTaskRunning(client, tsk);
    const gen = new crdt.Text(2);
    await client.arcs.dispatch(
      a.key,
      gen.insert(0, "a -> b").map((op) => arc.insertChar(op)),
    );
    await renderControls(a.key);
    await hoverRedeploy();
    await waitFor(async () =>
      expect((await screen.findByRole("tooltip")).textContent).toBe(
        "Automation edited. Redeploy to apply.",
      ),
    );
  });

  it("should name the module that changed in the redeploy tooltip", async () => {
    const rck = await createArcRack();
    const [trig, out] = await client.channels.create([
      { name: uniqueName("trig"), dataType: "uint8", virtual: true },
      { name: uniqueName("out"), dataType: "int64", virtual: true },
    ]);
    const moduleName = uniqueName("mod");
    const flow = `${trig.name} -> ${moduleName}.f() -> ${out.name}`;
    const m = await client.arcs.create({
      name: moduleName,
      type: "module",
      source: {
        mode: "text",
        text: { raw: "export func f() {t u8} i64 { return 1 }" },
      },
    });
    const a = await client.arcs.create({
      name: uniqueName("arc"),
      type: "automation",
      source: {
        mode: "text",
        text: {
          raw: `import ${moduleName}\n\n${flow}`,
        },
      },
    });
    const tsk = await client.arcs.updateTask(a.key, { rack: rck.key });
    if (tsk == null) throw new Error("expected a deployment task");
    await reportTaskRunning(client, tsk);
    const gen = new crdt.Text(2);
    await client.arcs.dispatch(
      m.key,
      gen
        .insert(0, "\nexport func g() i64 { return 2 }\n")
        .map((op) => arc.insertChar(op)),
    );
    await expect
      .poll(async () => await client.arcs.retrieveChanges(a.key), { timeout: 5000 })
      .toEqual([{ type: "module", path: moduleName, kind: "changed" }]);
    await renderControls(a.key);
    await hoverRedeploy();
    await waitFor(async () =>
      expect((await screen.findByRole("tooltip")).textContent).toBe(
        `Module ${moduleName} changed. Redeploy to apply.`,
      ),
    );
  });

  it("should show a module's compile status and no actions", async () => {
    const a = await client.arcs.create({
      name: uniqueName("mod"),
      type: "module",
      source: { mode: "text", text: { raw: "export func f() i64 { return 1 }" } },
    });
    const { container } = await renderControls(a.key);
    await screen.findByText("Compiled");
    expect(queryIconButton(container, "play")).toBeNull();
    expect(queryIcon(container, "refresh")).toBeNull();
  });

  it("should show why a module failed to compile", async () => {
    const a = await client.arcs.create({
      name: uniqueName("mod"),
      type: "module",
      source: { mode: "text", text: { raw: "export func f() i64 { return x }" } },
    });
    await renderControls(a.key);
    await screen.findByText("Failed to compile");
  });

  describe("performance", () => {
    it("should disable the select while no rack is bound", async () => {
      const a = await client.arcs.create({
        name: uniqueName("arc"),
        type: "automation",
        source: { mode: "text" },
      });
      await renderControls(a.key);
      const trigger = await findDialogTriggerByText("Auto");
      expect(trigger.getAttribute("aria-disabled")).toBe("true");
    });

    it("should show the stored level with its glyph as the trigger icon", async () => {
      const rck = await createArcRack();
      const a = await client.arcs.create({
        name: uniqueName("arc"),
        type: "automation",
        source: { mode: "text" },
      });
      await client.arcs.updateTask(a.key, { rack: rck.key, performance: "medium" });
      await renderControls(a.key);
      const trigger = await findDialogTriggerByText("Medium");
      expect(queryIcon(trigger, "meter-medium")).toBe(trigger.firstElementChild);
    });

    it("should list every level with no search", async () => {
      const rck = await createArcRack();
      const a = await client.arcs.create({
        name: uniqueName("arc"),
        type: "automation",
        source: { mode: "text" },
      });
      await client.arcs.updateTask(a.key, { rack: rck.key });
      await renderControls(a.key);
      const trigger = await findDialogTriggerByText("Auto");
      await waitFor(() => expect(trigger.getAttribute("aria-disabled")).toBeNull());
      fireEvent.click(trigger);
      await waitFor(() =>
        expect(screen.getAllByRole("option").map((o) => o.textContent)).toEqual([
          expect.stringMatching(/^Auto/),
          expect.stringMatching(/^Low/),
          expect.stringMatching(/^Medium/),
          expect.stringMatching(/^High/),
        ]),
      );
      expect(screen.queryByPlaceholderText(/Search/)).toBeNull();
    });

    it("should name the level in the tooltip of a collapsed trigger", async () => {
      const rck = await createArcRack();
      const a = await client.arcs.create({
        name: uniqueName("arc"),
        type: "automation",
        source: { mode: "text" },
      });
      await client.arcs.updateTask(a.key, { rack: rck.key, performance: "medium" });
      await renderControls(a.key);
      const trigger = await findDialogTriggerByText("Medium");
      await waitFor(() => expect(trigger.getAttribute("aria-disabled")).toBeNull());
      collapseSelectTrigger(trigger);
      fireEvent.pointerOver(trigger, { pointerType: "mouse" });
      expect((await screen.findByRole("tooltip")).textContent).toBe("Medium");
    });

    it("should set the level of a bound arc", async () => {
      const rck = await createArcRack();
      const a = await client.arcs.create({
        name: uniqueName("arc"),
        type: "automation",
        source: { mode: "text" },
      });
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

  describe("without permission to command the task", () => {
    it("should show the status without any action", async () => {
      const rck = await createArcRack();
      const a = await client.arcs.create({
        name: uniqueName("arc"),
        type: "automation",
        source: { mode: "text" },
      });
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
