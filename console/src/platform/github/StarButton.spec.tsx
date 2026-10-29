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
import { act, screen } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { GitHub } from "@/platform/github";
import { renderWithConsole } from "@/testutil";

describe("GitHub.StarButton", () => {
  it("should not ask for a star outside Synnax Desktop", async () => {
    const commands = vi.fn<(cmd: string) => void>();
    mockIPC((cmd) => {
      commands(cmd);
      return cmd === "github_star_status" ? "not_starred" : undefined;
    });
    onTestFinished(clearMocks);
    await renderWithConsole(<GitHub.StarButton />);
    await act(async () => await scheduler.flushTaskQueue());
    expect(commands).not.toHaveBeenCalledWith("github_star_status");
    expect(screen.queryByRole("button", { name: /star/i })).toBeNull();
  });
});
