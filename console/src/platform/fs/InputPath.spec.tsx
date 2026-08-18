// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({
  engine: "web",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/plugin-dialog", () => ({ open: vi.fn() }));
vi.mock("@tauri-apps/plugin-fs", () => ({
  readDir: vi.fn(),
  readFile: vi.fn(),
  readTextFile: vi.fn(),
}));

import { type Status } from "@synnaxlabs/pluto";
import { open } from "@tauri-apps/plugin-dialog";

import { FS } from "@/platform/fs";
import { CaptureStatuses, renderWithConsole } from "@/testutil";

const openMock = vi.mocked(open);

describe("FS.InputPath", () => {
  beforeEach(() => {
    mocks.engine = "web";
    openMock.mockReset();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should show a placeholder without a path and the path once one is set", async () => {
    const { rerender } = await renderWithConsole(
      <FS.InputPath value="" onChange={vi.fn()} />,
    );
    expect(screen.getByText("No file selected")).toBeTruthy();
    rerender(<FS.InputPath value="/tmp/config.json" onChange={vi.fn()} />);
    expect(screen.getByText("/tmp/config.json")).toBeTruthy();
    expect(screen.queryByText("No file selected")).toBeNull();
  });

  it("should surface an error and not call onChange when clicked in the browser", async () => {
    const onChange = vi.fn();
    let statuses: Status.NotificationSpec[] = [];
    await renderWithConsole(
      <>
        <FS.InputPath value="" onChange={onChange} />
        <CaptureStatuses onStatuses={(s) => (statuses = s)} />
      </>,
    );
    fireEvent.click(screen.getByText("Select file"));
    await waitFor(() =>
      expect(
        statuses.some(
          (s) => s.variant === "error" && s.message === "Failed to open file",
        ),
      ).toBe(true),
    );
    expect(openMock).not.toHaveBeenCalled();
    expect(onChange).not.toHaveBeenCalled();
  });

  describe("tauri engine", () => {
    beforeEach(() => {
      mocks.engine = "tauri";
    });

    it("should open a dialog and report the chosen path", async () => {
      openMock.mockResolvedValue("/tmp/picked.json");
      const onChange = vi.fn();
      await renderWithConsole(
        <FS.InputPath value="" onChange={onChange} extension="json" />,
      );
      fireEvent.click(screen.getByText("Select file"));
      await waitFor(() => expect(onChange).toHaveBeenCalledWith("/tmp/picked.json"));
      expect(openMock).toHaveBeenCalledWith({
        title: undefined,
        directory: false,
        multiple: false,
        filters: [{ name: "JSON", extensions: ["json"] }],
      });
    });

    it("should not call onChange when the dialog is cancelled", async () => {
      openMock.mockResolvedValue(null);
      const onChange = vi.fn();
      await renderWithConsole(<FS.InputPath value="" onChange={onChange} />);
      fireEvent.click(screen.getByText("Select file"));
      await waitFor(() => expect(openMock).toHaveBeenCalled());
      expect(onChange).not.toHaveBeenCalled();
    });
  });
});
