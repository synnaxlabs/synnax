// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { createTaskStatus, renderInTaskForm } from "@/platform/task/testutil";
import { getIconButton } from "@/testutil";

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

const CONFIG = { channels: [] };
// The server assigns config hashes; overrides.configHash is what the driver reports
// as deployed, so leaving it unset means the running instance matches the task.
const CONFIG_HASH = "2de66015b3bdded8";

const runningValues = (overrides: { configHash?: string; rack?: number } = {}) => ({
  key: "1",
  rack: 2,
  config: CONFIG,
  configHash: CONFIG_HASH,
  status: createTaskStatus({
    details: {
      task: "1",
      running: true,
      configHash: overrides.configHash ?? CONFIG_HASH,
      rack: overrides.rack ?? 2,
    },
  }),
});

describe("Controls.Controls", () => {
  it("should hide the actions when the task is a snapshot", async () => {
    const { container } = await renderInTaskForm(
      <Task.Controls.Controls
        formStatus={successStatus}
        onDeploy={vi.fn()}
        onStop={vi.fn()}
      />,
      { values: { snapshot: true } },
    );
    await waitFor(() =>
      expect(screen.getByText("Task has not been deployed")).toBeTruthy(),
    );
    expect(container.querySelector("[aria-label='pluto-icon--play']")).toBeNull();
  });

  it("should invoke onDeploy when the start button is pressed", async () => {
    const onDeploy = vi.fn();
    const { container } = await renderInTaskForm(
      <Task.Controls.Controls
        formStatus={successStatus}
        onDeploy={onDeploy}
        onStop={vi.fn()}
      />,
      { values: { key: "1" } },
    );
    fireEvent.click(getIconButton(container, "play"));
    expect(onDeploy).toHaveBeenCalledTimes(1);
  });

  it("should invoke onStop when the task is running", async () => {
    const onDeploy = vi.fn();
    const onStop = vi.fn();
    const { container } = await renderInTaskForm(
      <Task.Controls.Controls
        formStatus={successStatus}
        onDeploy={onDeploy}
        onStop={onStop}
      />,
      { values: runningValues() },
    );
    fireEvent.click(getIconButton(container, "pause"));
    expect(onStop).toHaveBeenCalledTimes(1);
    expect(onDeploy).not.toHaveBeenCalled();
  });

  it("should not invoke onDeploy when the task has no key", async () => {
    const onDeploy = vi.fn();
    const { container } = await renderInTaskForm(
      <Task.Controls.Controls
        formStatus={successStatus}
        onDeploy={onDeploy}
        onStop={vi.fn()}
      />,
      { values: { key: undefined } },
    );
    fireEvent.click(getIconButton(container, "play"));
    expect(onDeploy).not.toHaveBeenCalled();
  });

  it("should surface the form status and disable start/stop when the form is invalid", async () => {
    const onDeploy = vi.fn();
    const { container } = await renderInTaskForm(
      <Task.Controls.Controls
        formStatus={errorStatus}
        onDeploy={onDeploy}
        onStop={vi.fn()}
      />,
      { values: { key: "1" } },
    );
    await waitFor(() => expect(screen.getByText("Config is invalid")).toBeTruthy());
    fireEvent.click(getIconButton(container, "play"));
    expect(onDeploy).not.toHaveBeenCalled();
  });

  it("should not show the redeploy button when the running config matches", async () => {
    await renderInTaskForm(
      <Task.Controls.Controls
        formStatus={successStatus}
        onDeploy={vi.fn()}
        onStop={vi.fn()}
      />,
      { values: runningValues() },
    );
    expect(screen.queryByText("Redeploy")).toBeNull();
  });

  it("should invoke onDeploy from the redeploy button when the task has drifted", async () => {
    const onDeploy = vi.fn();
    await renderInTaskForm(
      <Task.Controls.Controls
        formStatus={successStatus}
        onDeploy={onDeploy}
        onStop={vi.fn()}
      />,
      { values: runningValues({ configHash: "stale" }) },
    );
    fireEvent.click(await screen.findByText("Redeploy"));
    expect(onDeploy).toHaveBeenCalledTimes(1);
  });

  it("should hide the redeploy button when the task is not running", async () => {
    await renderInTaskForm(
      <Task.Controls.Controls
        formStatus={successStatus}
        onDeploy={vi.fn()}
        onStop={vi.fn()}
      />,
      { values: { key: "1", rack: 2, config: CONFIG } },
    );
    expect(screen.queryByText("Redeploy")).toBeNull();
  });

  it("should expand the status on click and contract it on a second click", async () => {
    await renderInTaskForm(
      <Task.Controls.Controls
        formStatus={successStatus}
        onDeploy={vi.fn()}
        onStop={vi.fn()}
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
