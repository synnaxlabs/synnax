// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { id, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { renderWithConsole } from "@/testutil";

const makeStatus = (overrides: Partial<status.Status> = {}): status.Status => ({
  key: id.create(),
  name: "Task Status",
  variant: "success",
  message: "Running smoothly",
  description: "",
  time: TimeStamp.now(),
  ...overrides,
});

describe("Controls.Status", () => {
  it("should render the status message", async () => {
    await renderWithConsole(
      <Task.Controls.Status
        status={makeStatus({ message: "All good" })}
        expanded={false}
        onToggle={vi.fn()}
      />,
    );
    expect(screen.getByText("All good")).toBeTruthy();
  });

  it("should render the fallback message when message is empty", async () => {
    await renderWithConsole(
      <Task.Controls.Status
        status={makeStatus({ message: "" })}
        expanded={false}
        onToggle={vi.fn()}
        fallbackMessage="Nothing to report"
      />,
    );
    expect(screen.getByText("Nothing to report")).toBeTruthy();
  });

  it("should invoke onToggle when clicked", async () => {
    const onToggle = vi.fn();
    await renderWithConsole(
      <Task.Controls.Status
        status={makeStatus()}
        expanded={false}
        onToggle={onToggle}
      />,
    );
    fireEvent.click(screen.getByText("Running smoothly"));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it("should reveal the copy-diagnostics button and description when expanded", async () => {
    await renderWithConsole(
      <Task.Controls.Status
        status={makeStatus({ description: "detailed diagnostics" })}
        expanded
        onToggle={vi.fn()}
      />,
    );
    expect(screen.getByText("Copy diagnostics")).toBeTruthy();
    expect(screen.getByText("detailed diagnostics")).toBeTruthy();
  });

  it("should not render the copy-diagnostics button when collapsed", async () => {
    await renderWithConsole(
      <Task.Controls.Status
        status={makeStatus({ description: "detailed diagnostics" })}
        expanded={false}
        onToggle={vi.fn()}
      />,
    );
    expect(screen.queryByText("Copy diagnostics")).toBeNull();
    expect(screen.queryByText("detailed diagnostics")).toBeNull();
  });

  it("should render the loading icon for a loading status", async () => {
    const { container } = await renderWithConsole(
      <Task.Controls.Status
        status={makeStatus({ variant: "loading", message: "Configuring" })}
        expanded={false}
        onToggle={vi.fn()}
      />,
    );
    expect(container.querySelector('[aria-label="pluto-icon--loading"]')).toBeTruthy();
  });
});
