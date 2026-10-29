// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { type Mock, onTestFinished, vi } from "vitest";

import { GitHub } from "@/platform/github";
import {
  type ConsolePreloadedState,
  renderHookWithConsole,
  renderWithConsole,
  type TestStore,
} from "@/testutil";

export type Command = (cmd: string, args: unknown) => void;

/**
 * Stands in for the GitHub CLI behind the app's commands.
 * @returns A spy that receives every command the app invokes.
 */
export const mockCLI = (
  status: string | Promise<string>,
  starred = true,
): Mock<Command> => {
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

/** Mounts the status check and a star button over one store, as the app does. */
export const renderStar = async (
  preloadedState?: ConsolePreloadedState,
): Promise<TestStore> => {
  const { store } = await renderHookWithConsole(GitHub.useSyncStatus, {
    preloadedState,
  });
  await renderWithConsole(<GitHub.StarButton />, { store });
  return store;
};
