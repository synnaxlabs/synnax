// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Status as LyraStatus } from "@synnaxlabs/lyra/status";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
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
    lastExit: {
      reason: "crashed",
      message: "exited with 3",
      uptimeSeconds: 12,
      exitCode: 3,
    },
    waits: 0,
    lastWaitMs: null,
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
  JSON.stringify({ level: "warn", ts: 1790014510, msg: "slow write", took: "2.1s" }),
  JSON.stringify({
    level: "error",
    ts: 1790014512,
    logger: "api",
    caller: "api/open.go:42",
    msg: "failed to open the store",
    error: "permission denied",
    errorVerbose: "permission denied\n(1) attached stack trace",
    stacktrace: "main.open\n\tapi/open.go:42",
  }),
  "panic: not a structured line",
].join("\n");

const NEW_LINE = JSON.stringify({ level: "info", ts: 1790014520, msg: "a new line" });

// The dialog reads the log tail again every 2 seconds.
const REFRESH_MS = 2000;

const LINK = {
  secret: "shh",
  email: "ada@example.com",
  user: "user_a",
  analyticsAllowed: true,
};

const findLines = async (): Promise<HTMLElement[]> => {
  const log = await screen.findByLabelText("Log");
  await waitFor(() => expect(log.children).toHaveLength(4));
  return Array.from(log.children) as HTMLElement[];
};

/** Gives the log the scroll metrics of a box that holds less than its lines. */
const stubScrolling = (log: HTMLElement): void => {
  let top = 0;
  Object.defineProperty(log, "scrollHeight", { configurable: true, value: 1000 });
  Object.defineProperty(log, "clientHeight", { configurable: true, value: 100 });
  Object.defineProperty(log, "scrollTop", {
    configurable: true,
    get: () => top,
    set: (next: number) => (top = next),
  });
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
    const [info, , , panic] = await findLines();
    expect(info.textContent).toMatch(/INFO storage opened the store$/);
    expect(panic.textContent).toBe("panic: not a structured line");
  });

  it("should show the fields of an entry, but not its caller or stack trace", async () => {
    await openDiagnostics(RUNNING);
    const [, warn, error] = await findLines();
    expect(warn.textContent).toMatch(/WARN slow write took=2.1s$/);
    expect(error.textContent).toMatch(
      /ERROR api failed to open the store error="permission denied"$/,
    );
  });

  it("should quote a field value that a reader could take for another type", async () => {
    const line = JSON.stringify({
      level: "info",
      ts: 1790014509,
      msg: "opened",
      name: "store",
      count: 3,
      digits: "3",
      flag: "true",
      empty: "",
      text: "has space",
      color: "\u001b[31mred\u001b[0m",
      nested: { a: 1 },
    });
    await openDiagnostics(RUNNING, { supervisor_log_tail: () => line });
    const log = await screen.findByLabelText("Log");
    await waitFor(() =>
      expect(log.textContent).toMatch(
        /opened name=store count=3 digits="3" flag="true" empty="" text="has space" color=red nested=\{"a":1\}$/,
      ),
    );
  });

  it("should mark warnings and errors", async () => {
    await openDiagnostics(RUNNING);
    const [info, warn, error, panic] = await findLines();
    expect(warn.classList).toContain("console-diagnostics__line--warn");
    expect(error.classList).toContain("console-diagnostics__line--error");
    for (const line of [info, panic])
      expect(line.className).toBe("console-diagnostics__line");
  });

  it("should copy a header and the recent log with full times", async () => {
    const writeText = stubClipboardWriteText();
    await openDiagnostics(RUNNING);
    await screen.findByText("2 in this session");
    await findLines();
    fireEvent.click(screen.getByRole("button", { name: "Copy logs" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const [text] = writeText.mock.calls[0] as [string];
    const [header, body] = text.split("\n\n");
    expect(header).toMatch(
      /^Synnax Desktop diagnostics\nVersion: 0.58.0\nState: Running\nCopied: \S+Z$/,
    );
    expect(body).toBe(
      [
        "2026-09-21T18:15:09.900Z INFO storage opened the store",
        "2026-09-21T18:15:10.000Z WARN slow write took=2.1s",
        '2026-09-21T18:15:12.000Z ERROR api failed to open the store error="permission denied"',
        "panic: not a structured line",
      ].join("\n"),
    );
  });

  describe("following new lines", () => {
    beforeEach(() => {
      vi.useFakeTimers({ toFake: ["setInterval", "clearInterval"] });
    });
    afterEach(() => {
      vi.useRealTimers();
    });

    const openGrowingLog = async () => {
      let tail = LOG;
      let reads = 0;
      await openDiagnostics(RUNNING, {
        supervisor_log_tail: () => {
          reads++;
          return tail;
        },
      });
      await findLines();
      const log = screen.getByLabelText("Log");
      stubScrolling(log);
      const grow = async (): Promise<void> => {
        tail = `${LOG}\n${NEW_LINE}`;
        const before = reads;
        await act(async () => {
          await vi.advanceTimersByTimeAsync(REFRESH_MS);
        });
        expect(reads).toBe(before + 1);
      };
      return { log, grow };
    };

    /** Selects from before start to after end, as a drag across them does. */
    const select = (start: Node, end: Node): void => {
      const range = document.createRange();
      range.setStartBefore(start);
      range.setEndAfter(end);
      getSelection()?.removeAllRanges();
      getSelection()?.addRange(range);
      onTestFinished(() => getSelection()?.removeAllRanges());
      act(() => {
        document.dispatchEvent(new Event("selectionchange"));
      });
    };

    it("should hold the lines while the person reads above the end", async () => {
      const { log, grow } = await openGrowingLog();
      log.scrollTop = 0;
      fireEvent.scroll(log);
      const latest = await screen.findByRole("button", { name: "Jump to latest" });
      await grow();
      expect(log.textContent).not.toContain("a new line");
      fireEvent.click(latest);
      expect(log.textContent).toContain("a new line");
      expect(log.scrollTop).toBe(1000);
      expect(screen.queryByRole("button", { name: "Jump to latest" })).toBeNull();
    });

    it("should hold the lines while the person selects text in them", async () => {
      const { log, grow } = await openGrowingLog();
      select(log.children[0], log.children[0]);
      const latest = await screen.findByRole("button", { name: "Jump to latest" });
      await grow();
      expect(log.textContent).not.toContain("a new line");
      fireEvent.click(latest);
      expect(log.textContent).toContain("a new line");
    });

    it("should hold the lines for a selection that starts outside the log", async () => {
      const { log, grow } = await openGrowingLog();
      select(screen.getByText("2 in this session"), log.children[0]);
      const latest = await screen.findByRole("button", { name: "Jump to latest" });
      await grow();
      expect(log.textContent).not.toContain("a new line");
      fireEvent.click(latest);
      expect(log.textContent).toContain("a new line");
    });

    it("should show new lines at the end while the view is there", async () => {
      const { log, grow } = await openGrowingLog();
      await grow();
      await waitFor(() => expect(log.textContent).toContain("a new line"));
      expect(log.scrollTop).toBe(1000);
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
    const { commands } = await openDiagnostics({
      state: "failed",
      reason: "crashed",
      message: "gave up",
    });
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
