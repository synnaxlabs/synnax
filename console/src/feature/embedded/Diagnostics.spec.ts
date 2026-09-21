// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { save } from "@tauri-apps/plugin-dialog";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Embedded } from "@/feature/embedded";
import {
  clearSupervisor,
  type CommandHandler,
  createDesktopWrapper,
  mockSupervisor,
  RUNNING,
} from "@/feature/embedded/testutil";

vi.mock("@tauri-apps/plugin-dialog", () => ({ save: vi.fn() }));

const DIAGNOSTICS = {
  version: "0.58.0",
  history: { starts: 2, readyAt: 1790014509000, lastExit: "exited with 3" },
  dataDir: "/data/synnax",
  logDir: "/logs/synnax",
  dataSize: 3_000_000,
};

const LOG = [
  JSON.stringify({
    level: "info",
    ts: 1790014509.9,
    logger: "storage",
    msg: "\u001b[32mopened the store\u001b[0m",
  }),
  "panic: not a structured line",
].join("\n");

const openDiagnostics = async (
  status: Embedded.Status,
  handlers: Record<string, CommandHandler> = {},
): Promise<ReturnType<typeof mockSupervisor>> => {
  const supervisor = mockSupervisor(() => status, {
    supervisor_diagnostics: () => DIAGNOSTICS,
    supervisor_log_tail: () => LOG,
    ...handlers,
  });
  const { wrapper } = await createDesktopWrapper();
  const { result } = renderHook(() => Embedded.useDiagnosticsModal(), { wrapper });
  await act(async () => result.current());
  await screen.findByText("Diagnostics");
  return supervisor;
};

describe("Embedded.useDiagnosticsModal", () => {
  beforeEach(() => vi.mocked(save).mockReset());
  afterEach(clearSupervisor);

  it("should show the health, the data, and a readable log", async () => {
    await openDiagnostics(RUNNING);
    expect(await screen.findByText("Running")).toBeTruthy();
    expect(await screen.findByText("2 in this session")).toBeTruthy();
    expect(screen.getByText("exited with 3")).toBeTruthy();
    expect(screen.getByText("3.0 MB in /data/synnax")).toBeTruthy();
    const log = await screen.findByLabelText("Log");
    await waitFor(() => {
      expect(log.textContent).toMatch(/INFO storage opened the store$/m);
      expect(log.textContent).toContain("panic: not a structured line");
    });
  });

  it("should restart a running Synnax only after a confirmation", async () => {
    const { commands } = await openDiagnostics(RUNNING);
    await screen.findByText("Running");
    fireEvent.click(screen.getByRole("button", { name: "Restart" }));
    expect(await screen.findByText(/want to restart Synnax/)).toBeTruthy();
    expect(commands).not.toHaveBeenCalledWith("supervisor_restart");
    const buttons = screen.getAllByRole("button", { name: "Restart" });
    await act(async () => {
      fireEvent.click(buttons[buttons.length - 1]);
    });
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_restart"));
  });

  it("should restart a stopped Synnax without a confirmation", async () => {
    const { commands } = await openDiagnostics({ state: "failed", message: "gave up" });
    expect(await screen.findByText("gave up")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Restart" }));
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_restart"));
  });

  it("should export an archive to the path the person picks", async () => {
    vi.mocked(save).mockResolvedValue("/home/me/diagnostics.zip");
    const exported = vi.fn<CommandHandler>();
    await openDiagnostics(RUNNING, { supervisor_export_diagnostics: exported });
    fireEvent.click(screen.getByRole("button", { name: "Export" }));
    await waitFor(() =>
      expect(exported).toHaveBeenCalledWith({ path: "/home/me/diagnostics.zip" }),
    );
  });

  it("should export nothing when the person cancels the save dialog", async () => {
    vi.mocked(save).mockResolvedValue(null);
    const { commands } = await openDiagnostics(RUNNING);
    fireEvent.click(screen.getByRole("button", { name: "Export" }));
    await waitFor(() => expect(save).toHaveBeenCalled());
    await act(async () => {});
    expect(commands).not.toHaveBeenCalledWith("supervisor_export_diagnostics");
  });

  it("should show the data folder and the logs", async () => {
    const { commands } = await openDiagnostics(RUNNING);
    fireEvent.click(screen.getByRole("button", { name: "Show data folder" }));
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_show_data"));
    fireEvent.click(screen.getByRole("button", { name: "Show logs" }));
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_show_logs"));
  });
});
