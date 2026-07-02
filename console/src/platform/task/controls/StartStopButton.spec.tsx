// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { renderWithConsole } from "@/testutil";

describe("Controls.StartStopButton", () => {
  it("should show the play icon when not running", async () => {
    const { container } = await renderWithConsole(
      <Task.Controls.StartStopButton running={false} onClick={vi.fn()} />,
    );
    expect(container.querySelector('[aria-label="pluto-icon--play"]')).toBeTruthy();
    expect(container.querySelector('[aria-label="pluto-icon--pause"]')).toBeNull();
  });

  it("should show the pause icon when running", async () => {
    const { container } = await renderWithConsole(
      <Task.Controls.StartStopButton running onClick={vi.fn()} />,
    );
    expect(container.querySelector('[aria-label="pluto-icon--pause"]')).toBeTruthy();
    expect(container.querySelector('[aria-label="pluto-icon--play"]')).toBeNull();
  });

  it("should invoke onClick when pressed", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Task.Controls.StartStopButton running={false} onClick={onClick} />,
    );
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("should not invoke onClick when disabled", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Task.Controls.StartStopButton running={false} onClick={onClick} disabled />,
    );
    fireEvent.click(screen.getByRole("button"));
    expect(onClick).not.toHaveBeenCalled();
  });
});
