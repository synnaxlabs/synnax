// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { createTaskStatus, getControlsFrame } from "@/platform/task/testutil";
import { getIconButton, queryIcon, renderWithConsole } from "@/testutil";

describe("Controls.Bar", () => {
  it("should deploy on play and stop on stop", async () => {
    const onDeploy = vi.fn();
    const onStop = vi.fn();
    const { container, rerender } = await renderWithConsole(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running={false}
        onDeploy={onDeploy}
        onStop={onStop}
      />,
    );
    fireEvent.click(getIconButton(container, "play"));
    expect(onDeploy).toHaveBeenCalledTimes(1);
    expect(onStop).not.toHaveBeenCalled();

    rerender(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running
        onDeploy={onDeploy}
        onStop={onStop}
      />,
    );
    fireEvent.click(getIconButton(container, "stop"));
    expect(onStop).toHaveBeenCalledTimes(1);
    expect(onDeploy).toHaveBeenCalledTimes(1);
  });

  it("should stop a task whose start is loading", async () => {
    const onDeploy = vi.fn();
    const onStop = vi.fn();
    const { container } = await renderWithConsole(
      <Task.Controls.Bar
        status={createTaskStatus({ variant: "loading", message: "Starting task" })}
        running={false}
        startStopVariant="loading"
        onDeploy={onDeploy}
        onStop={onStop}
      />,
    );
    const button = getIconButton(container, "stop");
    expect(button.getAttribute("aria-disabled")).not.toBe("true");
    fireEvent.click(button);
    expect(onStop).toHaveBeenCalledTimes(1);
    expect(onDeploy).not.toHaveBeenCalled();
  });

  it("should invoke onDeploy from the redeploy button when drifted", async () => {
    const onDeploy = vi.fn();
    await renderWithConsole(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running
        drifted
        onDeploy={onDeploy}
        onStop={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByText("Redeploy"));
    expect(onDeploy).toHaveBeenCalledTimes(1);
  });

  it("should render extra actions in the action row and hide them with it", async () => {
    const { rerender } = await renderWithConsole(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running={false}
        extraActions={<button>Pick rack</button>}
        onDeploy={vi.fn()}
        onStop={vi.fn()}
      />,
    );
    expect(screen.getByText("Pick rack")).toBeTruthy();

    rerender(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running={false}
        hideActions
        extraActions={<button>Pick rack</button>}
        onDeploy={vi.fn()}
        onStop={vi.fn()}
      />,
    );
    expect(screen.queryByText("Pick rack")).toBeNull();
  });

  it("should keep the settings when the actions hide", async () => {
    const { container } = await renderWithConsole(
      <Task.Controls.Bar
        status={createTaskStatus()}
        running={false}
        hideActions
        settings={<button>Auto start</button>}
        onDeploy={vi.fn()}
        onStop={vi.fn()}
      />,
    );
    expect(screen.getByText("Auto start")).toBeTruthy();
    expect(container.querySelector("[aria-label='lyra-icon--play']")).toBeNull();
  });

  it("should deploy a task with no status", async () => {
    const onDeploy = vi.fn();
    const { container } = await renderWithConsole(
      <Task.Controls.Bar running={false} onDeploy={onDeploy} onStop={vi.fn()} />,
    );
    expect(screen.getByText("No status")).toBeTruthy();
    fireEvent.click(getIconButton(container, "play"));
    expect(onDeploy).toHaveBeenCalledTimes(1);
  });

  describe("status", () => {
    it("should expand on a click and show the diagnostics only while expanded", async () => {
      const stat = createTaskStatus({ description: "detailed diagnostics" });
      await renderWithConsole(
        <Task.Controls.Bar status={stat} running={false} onDeploy={vi.fn()} />,
      );
      expect(screen.queryByText("Copy diagnostics")).toBeNull();
      expect(screen.getAllByText("detailed diagnostics")).toHaveLength(1);
      fireEvent.click(screen.getByText("Running smoothly"));
      expect(screen.getByText("Copy diagnostics")).toBeTruthy();
      expect(screen.getAllByText("detailed diagnostics")).toHaveLength(1);
    });

    it("should fall back to fallbackMessage when the status message is empty", async () => {
      await renderWithConsole(
        <Task.Controls.Bar
          status={createTaskStatus({ message: "" })}
          running={false}
          fallbackMessage="Nothing to report"
          onDeploy={vi.fn()}
        />,
      );
      expect(screen.getByText("Nothing to report")).toBeTruthy();
    });

    it("should copy the full status diagnostics to the clipboard", async () => {
      const writeText = stubClipboardWriteText();
      const stat = createTaskStatus({ description: "detailed diagnostics" });
      await renderWithConsole(
        <Task.Controls.Bar status={stat} running={false} onDeploy={vi.fn()} />,
      );
      fireEvent.click(screen.getByText("Running smoothly"));
      fireEvent.click(screen.getByText("Copy diagnostics"));
      await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
      expect(writeText).toHaveBeenCalledWith(status.toString(stat));
    });

    it("should swap the status icon to a spinner for a loading status", async () => {
      const steady = await renderWithConsole(
        <Task.Controls.Bar
          status={createTaskStatus()}
          running={false}
          onDeploy={vi.fn()}
        />,
      );
      expect(queryIcon(steady.container, "loading")).toBeNull();

      const loading = await renderWithConsole(
        <Task.Controls.Bar
          status={createTaskStatus({ variant: "loading", message: "Configuring" })}
          running={false}
          onDeploy={vi.fn()}
        />,
      );
      expect(queryIcon(loading.container, "loading")).toBeTruthy();
    });

    it("should stay collapsed when a missing status is clicked", async () => {
      const { container, rerender } = await renderWithConsole(
        <Task.Controls.Bar running={false} onDeploy={vi.fn()} />,
      );
      expect(queryIcon(container, "status-concentric")).toBeTruthy();
      fireEvent.click(screen.getByText("No status"));
      rerender(
        <Task.Controls.Bar
          status={createTaskStatus()}
          running={false}
          onDeploy={vi.fn()}
        />,
      );
      expect(screen.queryByText("Copy diagnostics")).toBeNull();
    });

    it("should contract the expanded status on a click on the bar", async () => {
      const { container } = await renderWithConsole(
        <Task.Controls.Bar
          status={createTaskStatus()}
          running={false}
          onDeploy={vi.fn()}
        />,
      );
      fireEvent.click(screen.getByText("Running smoothly"));
      expect(screen.getByText("Copy diagnostics")).toBeTruthy();
      fireEvent.click(getControlsFrame(container));
      expect(screen.queryByText("Copy diagnostics")).toBeNull();
    });
  });

  describe("start and stop button", () => {
    it("should style the stop state as an error and keep the start state neutral", async () => {
      const { container, rerender } = await renderWithConsole(
        <Task.Controls.Bar
          status={createTaskStatus()}
          running={false}
          onDeploy={vi.fn()}
        />,
      );
      expect(getIconButton(container, "play").className).not.toContain(
        "lyra--status-error",
      );
      rerender(
        <Task.Controls.Bar status={createTaskStatus()} running onStop={vi.fn()} />,
      );
      expect(getIconButton(container, "stop").className).toContain(
        "lyra--status-error",
      );
    });

    it("should let the start and stop variant override the running error styling", async () => {
      const { container } = await renderWithConsole(
        <Task.Controls.Bar
          status={createTaskStatus()}
          running
          startStopVariant="warning"
          onStop={vi.fn()}
        />,
      );
      const button = getIconButton(container, "stop");
      expect(button.className).toContain("lyra--status-warning");
      expect(button.className).not.toContain("lyra--status-error");
    });

    it("should ignore the second click of a double click", async () => {
      const onDeploy = vi.fn();
      const { container } = await renderWithConsole(
        <Task.Controls.Bar
          status={createTaskStatus()}
          running={false}
          onDeploy={onDeploy}
        />,
      );
      const button = getIconButton(container, "play");
      fireEvent.click(button, { detail: 1 });
      fireEvent.click(button, { detail: 2 });
      expect(onDeploy).toHaveBeenCalledTimes(1);
    });

    it("should not deploy while disabled", async () => {
      const onDeploy = vi.fn();
      const { container } = await renderWithConsole(
        <Task.Controls.Bar
          status={createTaskStatus()}
          running={false}
          disabled
          onDeploy={onDeploy}
        />,
      );
      fireEvent.click(getIconButton(container, "play"));
      expect(onDeploy).not.toHaveBeenCalled();
    });

    it("should keep a click in the action row from contracting the status", async () => {
      const parentClick = vi.fn();
      await renderWithConsole(
        <div onClick={parentClick}>
          <Task.Controls.Bar
            status={createTaskStatus()}
            running={false}
            extraActions={<span>extra</span>}
            onDeploy={vi.fn()}
          />
        </div>,
      );
      fireEvent.click(screen.getByText("Running smoothly"));
      fireEvent.click(screen.getByText("extra"));
      expect(screen.getByText("Copy diagnostics")).toBeTruthy();
      expect(parentClick).not.toHaveBeenCalled();
    });
  });
});
