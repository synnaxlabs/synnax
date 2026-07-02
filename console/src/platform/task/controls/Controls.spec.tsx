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
import { renderInTaskForm } from "@/platform/task/testutil";

type FormStatus = ComponentProps<typeof Task.Controls.Controls>["formStatus"];

const successStatus: FormStatus = {
  key: id.create(),
  name: "form",
  variant: "success",
  message: "",
  description: "",
  time: TimeStamp.now(),
};

describe("Controls.Controls", () => {
  it("should render the Configure and start/stop actions when not a snapshot", async () => {
    await renderInTaskForm(
      <Task.Controls.Controls
        layoutKey="l1"
        formStatus={successStatus}
        onConfigure={vi.fn()}
      />,
      { values: { snapshot: false } },
    );
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /Configure/ })).toBeTruthy(),
    );
  });

  it("should hide the actions when the task is a snapshot", async () => {
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

  it("should issue a start/stop command when the start button is pressed", async () => {
    const { container } = await renderInTaskForm(
      <Task.Controls.Controls
        layoutKey="l1"
        formStatus={successStatus}
        onConfigure={vi.fn()}
      />,
      { values: { snapshot: false, key: "task-1" } },
    );
    const startButton = container
      .querySelector('[aria-label="pluto-icon--play"]')
      ?.closest("button");
    expect(startButton).toBeTruthy();
    fireEvent.click(startButton as HTMLButtonElement);
    await waitFor(() =>
      expect(screen.getByRole("button", { name: /Configure/ })).toBeTruthy(),
    );
  });

  it("should surface the form status when it is not successful", async () => {
    const errStatus: FormStatus = {
      key: id.create(),
      name: "form",
      variant: "error",
      message: "Config is invalid",
      description: "",
      time: TimeStamp.now(),
      details: { stack: "", error: new Error("Config is invalid") },
    };
    await renderInTaskForm(
      <Task.Controls.Controls
        layoutKey="l1"
        formStatus={errStatus}
        onConfigure={vi.fn()}
      />,
      { values: { snapshot: false } },
    );
    await waitFor(() => expect(screen.getByText("Config is invalid")).toBeTruthy());
  });
});
