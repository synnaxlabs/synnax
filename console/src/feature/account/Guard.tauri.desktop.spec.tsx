// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { scheduler } from "@synnaxlabs/x";
import { act, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  engine: "tauri" as const,
  label: "main",
  onOpenUrl: vi.fn(async () => () => {}),
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/api/window", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  getCurrentWindow: () => ({ label: mocks.label }),
}));

vi.mock("@tauri-apps/plugin-deep-link", () => ({
  getCurrent: async () => null,
  onOpenUrl: mocks.onOpenUrl,
}));

import { Account } from "@/feature/account";
import { createStatusConsoleWrapper } from "@/testutil";

const renderGuard = async (): Promise<void> => {
  const { wrapper } = await createStatusConsoleWrapper({ client: createTestClient() });
  render(
    <Account.Guard>
      <span>licensed content</span>
    </Account.Guard>,
    { wrapper },
  );
};

describe("Account.Guard in the tauri engine", () => {
  beforeEach(() => {
    mocks.label = "main";
    mocks.onOpenUrl.mockClear();
  });

  it("should listen for login links in the main window", async () => {
    await renderGuard();
    await waitFor(() => expect(mocks.onOpenUrl).toHaveBeenCalledOnce());
  });

  it("should not listen for login links in another window", async () => {
    mocks.label = "some-child-window";
    await renderGuard();
    // Async effects start after a task queue flush, so wait one out before asserting.
    await act(async () => await scheduler.flushTaskQueue());
    expect(screen.getByText("licensed content")).toBeTruthy();
    expect(mocks.onOpenUrl).not.toHaveBeenCalled();
  });
});
