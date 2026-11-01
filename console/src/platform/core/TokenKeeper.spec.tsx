// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, render, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { Session } from "@/session";
import { createCore, createCoreState, keychainAccount } from "@/session/core/testutil";
import { createSessionConsoleWrapper, createStatusConsoleWrapper } from "@/testutil";

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
    const calls = recordKeychain();
    const core = createCore("Local");
    await renderKeeper(core);
    await waitFor(() => expect(calls).toHaveLength(1));
    expect(calls[0]).toMatchObject({
      cmd: "keychain_set",
      args: {
        account: keychainAccount(core),
        secret: expect.stringMatching(/^sys_/),
      },
    });
  });

  it("should forget the token once the Core rejects the session", async () => {
    const calls = recordKeychain();
    const core = createCore("Local", { password: "wrong" });
    await renderKeeper(core);
    await waitFor(() => expect(calls).toHaveLength(1));
    expect(calls[0]).toEqual({
      cmd: "keychain_delete",
      args: { account: keychainAccount(core) },
    });
  });

  it("should leave the keychain alone while the client belongs to another Core", async () => {
    const calls = recordKeychain();
    const client = createTestClient();
    await client.connect();
    const core = createCore("Other", { port: 9095 });
    const { wrapper } = await createStatusConsoleWrapper({
      client,
      status: client.connection.status,
      preloadedState: createCoreState([core], core.key),
    });
    render(<Core.TokenKeeper />, { wrapper });
    await act(async () => {});
    expect(calls).toEqual([]);
  });

  it("should not save a session to another Core at the same address", async () => {
    const calls = recordKeychain();
    const client = createTestClient();
    await client.connect();
    const first = createCore("First");
    const second = createCore("Second");
    const { wrapper, store } = await createStatusConsoleWrapper({
      client,
      status: client.connection.status,
      preloadedState: createCoreState([first, second], first.key),
    });
    render(<Core.TokenKeeper />, { wrapper });
    await waitFor(() => expect(calls).toHaveLength(1));
    await act(async () => {
      store.dispatch(Session.Core.select(second.key));
    });
    expect(calls).toHaveLength(1);
    expect(calls[0]).toMatchObject({
      cmd: "keychain_set",
      args: { account: keychainAccount(first) },
    });
  });
});
