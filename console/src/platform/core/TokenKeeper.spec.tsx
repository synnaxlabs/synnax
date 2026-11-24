// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createServer } from "node:http";
import type { AddressInfo } from "node:net";

import { connection, Synnax as Client } from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  createTestUser,
} from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, render, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime", async (importOriginal) => {
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
    // a status update can retry before the record changes
    await waitFor(() => expect(calls.length).toBeGreaterThan(0));
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

  it("should save the new token after the user sets its own password", async () => {
    const calls = recordKeychain();
    const username = `keeper-${id.create()}`;
    const admin = createTestClient();
    const { key: role } = await createEmptyRole(admin);
    await createTestUser(admin, { role, username, password: "old", name: "Keeper" });
    const core = createCore("Local", { username, password: "old" });
    const grabbed: { client?: Client | null } = {};
    const Grab = (): null => {
      grabbed.client = Synnax.use();
      return null;
    };
    const { wrapper } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: createCoreState([core], core.key),
    });
    render(
      <>
        <Core.TokenKeeper />
        <Grab />
      </>,
      { wrapper },
    );
    await waitFor(() => expect(calls).toHaveLength(1));
    await act(async () => {
      await grabbed.client?.auth.passwords.set({
        password: "new",
        currentPassword: "old",
      });
    });
    await waitFor(() => expect(calls).toHaveLength(2));
    expect(calls[1]).toMatchObject({
      cmd: "keychain_set",
      args: { secret: grabbed.client?.auth.token },
    });
    expect(calls[1]).not.toEqual(calls[0]);
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

  const SESSION = {
    key: "8b7c3e52-6f0a-4c1d-9e2b-3a4f5d6e7f80",
    subject: { type: "user", key: "1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f" },
    credential: "2d3e4f5a-6b7c-4d8e-9f0a-1b2c3d4e5f60",
    lastActive: 0,
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

  // A local Core that opens a session for any login and hands out token. An empty
  // token is how a Core answers a client whose session lives in a cookie.
  const renderCodeLogin = async (token: string) => {
    const server = createServer((req, res) => {
      req.resume();
      req.on("end", () => {
        res.setHeader("Content-Type", "application/json");
        res.end(
          JSON.stringify(
            req.url?.endsWith("/auth/session/create") === true
              ? { token, session: SESSION }
              : { sessions: [] },
          ),
        );
      });
    });
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    onTestFinished(
      async () => await new Promise<void>((resolve) => server.close(() => resolve())),
    );
    const { port } = server.address() as AddressInfo;
    const client = new Client({
      host: "127.0.0.1",
      port,
      secure: false,
      oidc: CODE,
      sessionCookie: token === "",
    });
    onTestFinished(async () => await client.close());
    await client.auth.sessions.retrieve();
    const core = createCore("Local", {
      host: "127.0.0.1",
      port,
      secure: false,
      username: "",
      password: "",
      oidc: CODE,
    });
    const { wrapper, store } = await createStatusConsoleWrapper({
      client,
      status: {
        ...connection.DEFAULT_STATUS,
        variant: "success",
        message: "Connected",
        details: { ...connection.DEFAULT_STATUS.details, authenticated: true },
      },
      preloadedState: createCoreState([core], core.key),
    });
    render(<Core.TokenKeeper />, { wrapper });
    return { core, store };
  };

  it("should forget a spent code on the web once a cookie holds the session", async () => {
    mocks.engine = "web";
    onTestFinished(() => {
      mocks.engine = "tauri";
    });
    const { core, store } = await renderCodeLogin("");
    await waitFor(() =>
      expect(store.getState().core.cores[core.key].oidc).toBeUndefined(),
    );
  });

  it("should keep a spent code on the web while a token holds the session", async () => {
    mocks.engine = "web";
    onTestFinished(() => {
      mocks.engine = "tauri";
    });
    const { core, store } = await renderCodeLogin("sys_token");
    await act(async () => {});
    expect(store.getState().core.cores[core.key].oidc).toEqual(CODE);
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
