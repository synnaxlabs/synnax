// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Status as LyraStatus } from "@synnaxlabs/lyra/status";
import { save } from "@tauri-apps/plugin-dialog";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  onTestFinished,
  vi,
} from "vitest";

import { Command } from "@/feature/command";
import { Embedded } from "@/feature/embedded";
import {
  clearSupervisor,
  type CommandHandler,
  createDesktopWrapper,
  mockSupervisor,
  RUNNING,
  type Status,
} from "@/feature/embedded/testutil";
import { License } from "@/platform/license";
import { stubHub } from "@/platform/license/testutil";
import { Session } from "@/session";
import { CaptureStatuses } from "@/testutil";

vi.mock("@tauri-apps/plugin-dialog", () => ({ save: vi.fn() }));

const mocks = vi.hoisted((): { engine: "web" | "tauri"; label: string } => ({
  engine: "web",
  label: "main",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/api/window", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  getCurrentWindow: () => ({ label: mocks.label }),
}));

const DIAGNOSTICS = {
  version: "0.58.0",
  history: {
    starts: 2,
    exits: 1,
    failures: 0,
    readies: 2,
    readyAt: 1790014509000,
    timeToReadyMs: 420,
    lastExit: { reason: "crashed", message: "exited with 3", uptimeSeconds: 12 },
  },
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

const LINK = {
  secret: "shh",
  email: "ada@example.com",
  user: "user_a",
  analyticsAllowed: true,
};

const openDiagnostics = async (
  status: Status,
  handlers: Record<string, CommandHandler> = {},
) => {
  const supervisor = mockSupervisor(() => status, {
    supervisor_diagnostics: () => DIAGNOSTICS,
    supervisor_log_tail: () => LOG,
    ...handlers,
  });
  const { wrapper, store } = await createDesktopWrapper();
  let statuses: LyraStatus.NotificationSpec[] = [];
  render(
    <>
      <Command.List
        commands={Embedded.COMMANDS}
        value=">"
        onChange={() => {}}
        inputPlaceholder={<>Commands</>}
      />
      <CaptureStatuses onStatuses={(next) => (statuses = next)} />
    </>,
    { wrapper },
  );
  const command = await screen.findByText("Open diagnostics");
  await act(async () => {
    fireEvent.click(command, { detail: 1 });
  });
  await screen.findByText("Diagnostics");
  return { ...supervisor, store, statuses: () => statuses };
};

/** Opens the erase dialog from the diagnostics and holds its confirm button. */
const holdErase = async (): Promise<void> => {
  fireEvent.click(screen.getByRole("button", { name: "Erase all data" }));
  await screen.findByText(/want to erase all data/);
  const buttons = screen.getAllByRole("button", { name: "Erase all data" });
  fireEvent.mouseDown(buttons[buttons.length - 1]);
};

/** Answers every request to the hub with `answer` until the test ends. */
describe("Embedded diagnostics", () => {
  beforeEach(() => {
    vi.mocked(save).mockReset();
    mocks.engine = "web";
    mocks.label = "main";
  });
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

  it("should erase the data only after a held confirmation", async () => {
    const { commands } = await openDiagnostics(RUNNING);
    fireEvent.click(screen.getByRole("button", { name: "Erase all data" }));
    expect(await screen.findByText(/want to erase all data/)).toBeTruthy();
    const buttons = screen.getAllByRole("button", { name: "Erase all data" });
    const confirm = buttons[buttons.length - 1];
    fireEvent.click(confirm);
    await act(async () => {});
    expect(commands).not.toHaveBeenCalledWith("supervisor_reset");
    fireEvent.mouseDown(confirm);
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_reset"));
  });

  it("should log the machine out before it erases the data", async () => {
    const hub = stubHub(async () => new Response(null, { status: 204 }));
    let atReset: Session.Account.SliceState | undefined;
    const { store, statuses } = await openDiagnostics(RUNNING, {
      supervisor_reset: () => {
        atReset = Session.Account.selectSliceState(store.getState());
      },
    });
    store.dispatch(Session.Account.link(LINK));
    await holdErase();
    await waitFor(() => expect(atReset).toEqual(Session.Account.ZERO_SLICE_STATE));
    // The status list renders after the reset command can already have run.
    await waitFor(() =>
      expect(statuses().map(({ message }) => message)).toContain("Erasing all data"),
    );
    expect(hub).toHaveBeenCalledWith(
      License.UNLINK_URL,
      expect.objectContaining({ headers: { authorization: "Bearer shh" } }),
    );
  });

  it("should erase the data when the hub cannot be reached", async () => {
    const offline = new TypeError("Failed to fetch");
    stubHub(async () => {
      throw offline;
    });
    const logged = vi.spyOn(console, "error").mockImplementation(() => {});
    onTestFinished(() => logged.mockRestore());
    const { commands, store } = await openDiagnostics(RUNNING);
    store.dispatch(Session.Account.link(LINK));
    await holdErase();
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_reset"));
    expect(Session.Account.selectSliceState(store.getState()).link).toBeUndefined();
    expect(logged).toHaveBeenCalledWith(offline);
  });

  it("should offer the erase only in the main window", async () => {
    mocks.engine = "tauri";
    mocks.label = "child";
    await openDiagnostics(RUNNING);
    expect(screen.getByRole("button", { name: "Show logs" })).toBeTruthy();
    expect(screen.queryByText("Erase all data")).toBeNull();
  });

  it("should show the data folder and the logs", async () => {
    const { commands } = await openDiagnostics(RUNNING);
    fireEvent.click(screen.getByRole("button", { name: "Show data folder" }));
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_show_data"));
    fireEvent.click(screen.getByRole("button", { name: "Show logs" }));
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_show_logs"));
  });
});
