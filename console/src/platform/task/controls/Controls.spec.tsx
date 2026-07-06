// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, task } from "@synnaxlabs/client";
import { id, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import {
  awaitCommand,
  createTaskStatus,
  renderInTaskForm,
  renderInTaskFormWithClient,
} from "@/platform/task/testutil";
import { getIconButton, uniqueName } from "@/testutil";

type FormStatus = ComponentProps<typeof Task.Controls.Controls>["formStatus"];

const successStatus: FormStatus = {
  key: id.create(),
  name: "form",
  variant: "success",
  message: "",
  description: "",
  time: TimeStamp.now(),
};

const errorStatus: FormStatus = {
  key: id.create(),
  name: "form",
  variant: "error",
  message: "Config is invalid",
  description: "",
  time: TimeStamp.now(),
  details: { stack: "", error: new Error("Config is invalid") },
};

const client = createTestClient();
const rack = await client.racks.create({ name: uniqueName("rack") });

const createTask = async () =>
  await rack.createTask({ name: uniqueName("tsk"), type: "test_type", config: {} });

describe("Controls.Controls", () => {
  it("should hide the configure and start/stop actions when the task is a snapshot", async () => {
    await renderInTaskForm(
      <Task.Controls.Controls
        layoutKey="l1"
        formStatus={successStatus}
        onConfigure={vi.fn()}
      />,
      { values: { snapshot: true } },
    );
    await waitFor(() =>
      expect(screen.getByText("Task has not been configured")).toBeTruthy(),
    );
    expect(screen.queryByRole("button", { name: /Configure/ })).toBeNull();
  });

  it("should invoke onConfigure when the Configure button is pressed", async () => {
    const onConfigure = vi.fn();
    await renderInTaskForm(
      <Task.Controls.Controls
        layoutKey="l1"
        formStatus={successStatus}
        onConfigure={onConfigure}
      />,
      { values: { snapshot: false } },
    );
    fireEvent.click(await screen.findByRole("button", { name: /Configure/ }));
    expect(onConfigure).toHaveBeenCalledTimes(1);
  });

  it("should issue a start command for the task when not running", async () => {
    const tsk = await createTask();
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    try {
      const { container } = await renderInTaskFormWithClient(
        <Task.Controls.Controls
          layoutKey="l1"
          formStatus={successStatus}
          onConfigure={vi.fn()}
        />,
        { client, values: { key: tsk.key } },
      );
      fireEvent.click(getIconButton(container, "play"));
      const cmd = await awaitCommand(streamer, tsk.key);
      expect(cmd.type).toBe("start");
    } finally {
      streamer.close();
    }
  });

  it("should issue a stop command for the task when running", async () => {
    const tsk = await createTask();
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    try {
      const { container } = await renderInTaskFormWithClient(
        <Task.Controls.Controls
          layoutKey="l1"
          formStatus={successStatus}
          onConfigure={vi.fn()}
        />,
        {
          client,
          values: {
            key: tsk.key,
            status: createTaskStatus({ details: { task: tsk.key, running: true } }),
          },
        },
      );
      fireEvent.click(getIconButton(container, "pause"));
      const cmd = await awaitCommand(streamer, tsk.key);
      expect(cmd.type).toBe("stop");
    } finally {
      streamer.close();
    }
  });

  it("should not issue a command when the task has no key", async () => {
    const executeCommand = vi.spyOn(client.tasks, "executeCommand");
    const { container } = await renderInTaskFormWithClient(
      <Task.Controls.Controls
        layoutKey="l1"
        formStatus={successStatus}
        onConfigure={vi.fn()}
      />,
      { client, values: { key: undefined } },
    );
    fireEvent.click(getIconButton(container, "play"));
    expect(executeCommand).not.toHaveBeenCalled();
    executeCommand.mockRestore();
  });

  it("should surface the form status and disable start/stop when the form is invalid", async () => {
    const executeCommand = vi.spyOn(client.tasks, "executeCommand");
    const tsk = await createTask();
    const { container } = await renderInTaskFormWithClient(
      <Task.Controls.Controls
        layoutKey="l1"
        formStatus={errorStatus}
        onConfigure={vi.fn()}
      />,
      { client, values: { key: tsk.key } },
    );
    await waitFor(() => expect(screen.getByText("Config is invalid")).toBeTruthy());
    fireEvent.click(getIconButton(container, "play"));
    expect(executeCommand).not.toHaveBeenCalled();
    executeCommand.mockRestore();
  });

  it("should expand the status on click and contract it on a second click", async () => {
    await renderInTaskForm(
      <Task.Controls.Controls
        layoutKey="l1"
        formStatus={successStatus}
        onConfigure={vi.fn()}
      />,
      { values: { status: createTaskStatus({ message: "Running smoothly" }) } },
    );
    expect(screen.queryByText("Copy diagnostics")).toBeNull();
    fireEvent.click(await screen.findByText("Running smoothly"));
    expect(await screen.findByText("Copy diagnostics")).toBeTruthy();
    fireEvent.click(screen.getByText("Running smoothly"));
    await waitFor(() => expect(screen.queryByText("Copy diagnostics")).toBeNull());
  });
});
