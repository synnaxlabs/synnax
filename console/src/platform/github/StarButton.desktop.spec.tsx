// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scheduler } from "@synnaxlabs/x";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, type Mock, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { GitHub } from "@/platform/github";
import { Session } from "@/session";
import { renderWithConsole, type TestStore } from "@/testutil";

const REPOSITORY = "https://github.com/synnaxlabs/synnax";

type Command = (cmd: string, args: unknown) => void;

/**
 * Stands in for the GitHub CLI behind the app's commands.
 * @returns A spy that receives every command the app invokes.
 */
const mockCLI = (status: string, starred = true): Mock<Command> => {
  const commands = vi.fn<Command>();
  mockIPC((cmd, args) => {
    commands(cmd, args);
    if (cmd === "github_star_status") return status;
    if (cmd === "github_star") return starred;
    return undefined;
  });
  onTestFinished(clearMocks);
  return commands;
};

const selectStarred = (store: TestStore): boolean =>
  store.getState()[Session.GitHub.SLICE_NAME].starred;

const clickStar = async (): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: /star/i }));
};

describe("GitHub.StarButton", () => {
  it("should star Synnax through the GitHub CLI and celebrate it", async () => {
    const commands = mockCLI("not_starred");
    const { store } = await renderWithConsole(<GitHub.StarButton />);
    await clickStar();
    expect(await screen.findByRole("button", { name: /starred/i })).toBeTruthy();
    expect(selectStarred(store)).toBe(true);
    expect(commands).toHaveBeenCalledWith("github_star", expect.anything());
    expect(commands).not.toHaveBeenCalledWith("plugin:shell|open", expect.anything());
  });

  it("should open the repository page when the GitHub CLI fails to star", async () => {
    const commands = mockCLI("not_starred", false);
    const { store } = await renderWithConsole(<GitHub.StarButton />);
    await clickStar();
    await waitFor(() => expect(selectStarred(store)).toBe(true));
    expect(commands).toHaveBeenCalledWith(
      "plugin:shell|open",
      expect.objectContaining({ path: REPOSITORY }),
    );
  });

  it("should open the repository page when the GitHub CLI cannot answer", async () => {
    const commands = mockCLI("unknown");
    const { store } = await renderWithConsole(<GitHub.StarButton />);
    await clickStar();
    await waitFor(() => expect(selectStarred(store)).toBe(true));
    expect(commands).not.toHaveBeenCalledWith("github_star", expect.anything());
    expect(commands).toHaveBeenCalledWith(
      "plugin:shell|open",
      expect.objectContaining({ path: REPOSITORY }),
    );
  });

  it("should remember a star the GitHub CLI reports without asking", async () => {
    mockCLI("starred");
    const { store } = await renderWithConsole(<GitHub.StarButton />);
    await waitFor(() => expect(selectStarred(store)).toBe(true));
    expect(screen.queryByRole("button", { name: /star/i })).toBeNull();
  });

  it("should not ask the GitHub CLI again once the user starred Synnax", async () => {
    const commands = mockCLI("not_starred");
    await renderWithConsole(<GitHub.StarButton />, {
      preloadedState: {
        [Session.GitHub.SLICE_NAME]: {
          ...Session.GitHub.ZERO_SLICE_STATE,
          starred: true,
        },
      },
    });
    await act(async () => await scheduler.flushTaskQueue());
    expect(commands).not.toHaveBeenCalledWith("github_star_status", expect.anything());
  });
});
