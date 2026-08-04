// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, status, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Arc } from "@synnaxlabs/pluto";
import { crdt } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { describe, expect, it } from "vitest";

import { TaskControls } from "@/feature/arc/editor/TaskControls";
import { awaitCommand } from "@/platform/task/testutil";
import {
  createConsoleWrapper,
  findDialogTrigger,
  getIconButton,
  stubGeometry,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

stubGeometry();

const createArcRack = async () =>
  await client.racks.create({ name: uniqueName("rack"), integrations: ["arc"] });

const renderControls = async (arcKey: arc.Key) => {
  const { wrapper } = await createConsoleWrapper({ client });
  return render(
    <Suspense fallback={null}>
      <Arc.Suspended arcKey={arcKey}>
        <TaskControls />
      </Arc.Suspended>
    </Suspense>,
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

const setRunning = async (tsk: task.Task): Promise<void> => {
  await client.statuses.set(
    status.create<ReturnType<typeof task.statusDetailsZ>>({
      key: task.statusKey(tsk.key),
      variant: "success",
      message: "Running",
      details: {
        task: tsk.key,
        running: true,
        cmd: "",
        configHash: tsk.configHash,
        rack: tsk.rack,
        data: {},
      },
    }),
  );
};

describe("TaskControls", () => {
  it("should deploy the arc when a rack is picked", async () => {
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

  it("should redeploy and start the arc from the play button", async () => {
    const rck = await createArcRack();
    const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
    const tsk = await client.arcs.deploy(a.key, rck.key);
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

  it("should offer a redeploy that restamps and starts when the arc drifts", async () => {
    const rck = await createArcRack();
    const a = await client.arcs.create({ name: uniqueName("arc"), mode: "text" });
    const tsk = await client.arcs.deploy(a.key, rck.key);
    if (tsk == null) throw new Error("expected a deployment task");
    await setRunning(tsk);
    const gen = new crdt.Text(2);
    await client.arcs.dispatch(
      a.key,
      gen.insert(0, "a -> b").map((op) => arc.insertChar(op)),
    );
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    try {
      await renderControls(a.key);
      fireEvent.click(await screen.findByText("Redeploy"));
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
});
