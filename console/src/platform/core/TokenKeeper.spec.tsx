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
import { createCore, createCoreState } from "@/session/core/testutil";
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
        account: `http://synnax@localhost:9090#${core.key}`,
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
      args: { account: `http://synnax@localhost:9090#${core.key}` },
    });
  });

  it("should not save its token again over a later login's token", async () => {
    const calls = recordKeychain();
    const core = createCore("Local");
    const { wrapper, store } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: createCoreState([core], core.key),
    });
    render(<Core.TokenKeeper />, { wrapper });
    await waitFor(() => expect(calls).toHaveLength(1));
    await act(async () => {
      store.dispatch(Session.Core.set({ ...core, lastLogin: Date.now() }));
    });
    expect(calls).toHaveLength(1);
    expect(calls[0]).toMatchObject({ cmd: "keychain_set" });
  });

  it("should retry a save the keychain rejected", async () => {
    const calls: Array<{ cmd: string }> = [];
    let failures = 1;
    mockIPC((cmd) => {
      if (!cmd.startsWith("keychain_")) return undefined;
      calls.push({ cmd });
      if (cmd === "keychain_set" && failures-- > 0) throw new Error("locked");
      return undefined;
    });
    onTestFinished(clearMocks);
    const core = createCore("Local");
    const { wrapper, store } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: createCoreState([core], core.key),
    });
    render(<Core.TokenKeeper />, { wrapper });
    await waitFor(() => expect(calls).toHaveLength(1));
    await act(async () => {
      store.dispatch(Session.Core.set({ ...core, lastLogin: Date.now() }));
    });
    await waitFor(() => expect(calls).toHaveLength(2));
    expect(calls).toEqual([{ cmd: "keychain_set" }, { cmd: "keychain_set" }]);
  });

  it("should not replay a rejection onto a later login's token", async () => {
    const calls = recordKeychain();
    const core = createCore("Local", { password: "wrong" });
    const { wrapper, store } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: createCoreState([core], core.key),
    });
    render(<Core.TokenKeeper />, { wrapper });
    await waitFor(() => expect(calls).toHaveLength(1));
    await act(async () => {
      store.dispatch(Session.Core.set({ ...core, lastLogin: Date.now() }));
    });
    expect(calls).toEqual([
      {
        cmd: "keychain_delete",
        args: { account: `http://synnax@localhost:9090#${core.key}` },
      },
    ]);
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

  const CODE = {
    provider: "idp",
    code: "code",
    verifier: "verifier",
    redirectUri: "http://127.0.0.1:4242/callback",
  };

  // a client logged in to the same Core the record names, which holds a spent code
  const renderWithCode = async () => {
    const client = createTestClient();
    await client.connect();
    const core = createCore("Local", { oidc: CODE });
    const { wrapper, store } = await createStatusConsoleWrapper({
      client,
      status: client.connection.status,
      preloadedState: createCoreState([core], core.key),
    });
    render(<Core.TokenKeeper />, { wrapper });
    return { core, store };
  };

  it("should keep a spent code on the web while no cookie holds the session", async () => {
    mocks.engine = "web";
    onTestFinished(() => {
      mocks.engine = "tauri";
    });
    const { core, store } = await renderWithCode();
    await act(async () => {});
    expect(store.getState().core.cores[core.key].oidc).toEqual(CODE);
  });
});
