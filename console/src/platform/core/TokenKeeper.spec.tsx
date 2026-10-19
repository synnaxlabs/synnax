// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { createCore, createCoreState } from "@/session/core/testutil";
import { createSessionConsoleWrapper } from "@/testutil";

/** Records the keychain writes and deletes the keeper issues. */
const recordKeychain = () => {
  const calls: Array<{ cmd: string; args: unknown }> = [];
  mockIPC((cmd, args) => {
    if (cmd.startsWith("keychain_")) calls.push({ cmd, args });
    return undefined;
  });
  onTestFinished(clearMocks);
  return calls;
};

const renderKeeper = async (core: ReturnType<typeof createCore>) => {
  const { wrapper } = await createSessionConsoleWrapper({
    client: null,
    preloadedState: createCoreState([core], core.key),
  });
  return render(<Core.TokenKeeper />, { wrapper });
};

describe("Core.TokenKeeper", () => {
  it("should save the token once the Core accepts the session", async () => {
    mocks.engine = "tauri";
    const calls = recordKeychain();
    await renderKeeper(createCore("Local"));
    await waitFor(() => expect(calls).toHaveLength(1));
    expect(calls[0]).toMatchObject({
      cmd: "keychain_set",
      args: { account: "localhost:9090", secret: expect.stringMatching(/^sys_/) },
    });
  });

  it("should forget the token once the Core rejects the session", async () => {
    mocks.engine = "tauri";
    const calls = recordKeychain();
    await renderKeeper(createCore("Local", { password: "wrong" }));
    await waitFor(() => expect(calls).toHaveLength(1));
    expect(calls[0]).toEqual({
      cmd: "keychain_delete",
      args: { account: "localhost:9090" },
    });
  });
});
