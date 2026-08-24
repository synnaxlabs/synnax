// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Status } from "@/platform/task/controls/Status";
import { createTaskStatus } from "@/platform/task/testutil";
import { queryIcon, renderWithConsole, stubClipboardWriteText } from "@/testutil";

describe("Controls.Status", () => {
  it("should invoke onToggle when clicked", async () => {
    const onToggle = vi.fn();
    await renderWithConsole(
      <Status status={createTaskStatus()} expanded={false} onToggle={onToggle} />,
    );
    fireEvent.click(screen.getByText("Running smoothly"));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it("should fall back to fallbackMessage when the status message is empty", async () => {
    await renderWithConsole(
      <Status
        status={createTaskStatus({ message: "" })}
        expanded={false}
        onToggle={vi.fn()}
        fallbackMessage="Nothing to report"
      />,
    );
    expect(screen.getByText("Nothing to report")).toBeTruthy();
  });

  it("should reveal the diagnostics and description only while expanded", async () => {
    const stat = createTaskStatus({ description: "detailed diagnostics" });
    const { rerender } = await renderWithConsole(
      <Status status={stat} expanded={false} onToggle={vi.fn()} />,
    );
    expect(screen.queryByText("Copy diagnostics")).toBeNull();
    expect(screen.queryByText("detailed diagnostics")).toBeNull();

    rerender(<Status status={stat} expanded onToggle={vi.fn()} />);
    expect(screen.getByText("Copy diagnostics")).toBeTruthy();
    expect(screen.getByText("detailed diagnostics")).toBeTruthy();

    rerender(<Status status={stat} expanded={false} onToggle={vi.fn()} />);
    expect(screen.queryByText("Copy diagnostics")).toBeNull();
  });

  it("should copy the full status diagnostics to the clipboard", async () => {
    const writeText = stubClipboardWriteText();
    const stat = createTaskStatus({ description: "detailed diagnostics" });
    await renderWithConsole(<Status status={stat} expanded onToggle={vi.fn()} />);
    fireEvent.click(screen.getByText("Copy diagnostics"));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText).toHaveBeenCalledWith(status.toString(stat));
  });

  it("should swap the status icon to a spinner for a loading status", async () => {
    const steady = await renderWithConsole(
      <Status status={createTaskStatus()} expanded={false} onToggle={vi.fn()} />,
    );
    expect(queryIcon(steady.container, "loading")).toBeNull();

    const loading = await renderWithConsole(
      <Status
        status={createTaskStatus({ variant: "loading", message: "Configuring" })}
        expanded={false}
        onToggle={vi.fn()}
      />,
    );
    expect(queryIcon(loading.container, "loading")).toBeTruthy();
  });
});
