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

import { auth, connection, Synnax as Client, user } from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  createTestUser,
} from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { expectAlways } from "@synnaxlabs/x/testutil";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, render, waitFor } from "@testing-library/react";
import type { PropsWithChildren, ReactElement } from "react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { Session } from "@/session";
import { createCore, createCoreState, keychainAccount } from "@/session/core/testutil";
import {
  createConsoleWrapper,
  createSessionConsoleWrapper,
  createStatusConsoleWrapper,
  disableActEnvironment,
} from "@/testutil";

/** Records the keychain calls the keeper issues. Every read answers with `stored`. */
const recordKeychain = (stored?: string) => {
  const calls: Array<{ cmd: string; args: unknown }> = [];
  mockIPC((cmd, args) => {
    if (cmd.startsWith("keychain_")) calls.push({ cmd, args });
    return cmd === "keychain_get" ? (stored ?? null) : undefined;
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

  // the client resumes the saved token once, and forgets it when the Core rejects it
  const REJECTED = (core: ReturnType<typeof createCore>) => {
    const account = keychainAccount(core);
    return [
      { cmd: "keychain_get", args: { account } },
      { cmd: "keychain_get", args: { account } },
      { cmd: "keychain_delete", args: { account, expected: "sys_dead" } },
    ];
  };

  it("should forget the token once the Core rejects the session", async () => {
    const calls = recordKeychain("sys_dead");
    const core = createCore("Local", { password: "wrong" });
    await renderKeeper(core);
    await waitFor(() => expect(calls).toEqual(REJECTED(core)));
  });

  it("should leave the keychain alone when it holds no token", async () => {
    const calls = recordKeychain();
    const core = createCore("Local", { password: "wrong" });
    await renderKeeper(core);
    await waitFor(() => expect(calls).toHaveLength(1));
    await expectAlways(() =>
      expect(calls).toEqual([
        { cmd: "keychain_get", args: { account: keychainAccount(core) } },
      ]),
    );
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
    const calls = recordKeychain("sys_dead");
    const core = createCore("Local", { password: "wrong" });
    const { wrapper, store } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: createCoreState([core], core.key),
    });
    render(<Core.TokenKeeper />, { wrapper });
    await waitFor(() => expect(calls).toEqual(REJECTED(core)));
    await act(async () => {
      store.dispatch(Session.Core.set({ ...core, lastLogin: Date.now() }));
    });
    expect(calls).toEqual(REJECTED(core));
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

  describe("with two windows", () => {
    interface Window {
      client?: Client | null;
      status?: connection.Status;
      setUsername?: (username: string) => void;
    }

    const Observe = ({ window }: { window: Window }): null => {
      window.client = Synnax.use();
      window.status = Synnax.useConnectionStatus();
      window.setUsername = Core.useSetUsername();
      return null;
    };

    const Connect = ({ children }: PropsWithChildren): ReactElement => (
      <Synnax.Provider connParams={Core.useConnParams()}>{children}</Synnax.Provider>
    );

    // Two windows of one desktop Console that resume one saved session from a shared
    // keychain, as after a relaunch, when no record holds the password.
    const openWindows = async () => {
      disableActEnvironment();
      const keychain = new Map<string, string>();
      // a write held back lands right after the next read, as a write from window A
      // would between window B's read and B's delete
      const race: { held: boolean; pending?: [string, string] } = { held: false };
      mockIPC((cmd, args) => {
        const { account, secret, expected } = args as {
          account: string;
          secret?: string;
          expected?: string;
        };
        if (cmd === "keychain_get") {
          const token = keychain.get(account) ?? null;
          if (race.pending != null) keychain.set(...race.pending);
          race.pending = undefined;
          return token;
        }
        if (cmd === "keychain_set" && secret != null)
          if (race.held) {
            race.held = false;
            race.pending = [account, secret];
          } else keychain.set(account, secret);
        if (
          cmd === "keychain_delete" &&
          (expected == null || keychain.get(account) === expected)
        )
          keychain.delete(account);
        return undefined;
      });
      onTestFinished(clearMocks);
      const admin = createTestClient();
      const roles = await admin.access.roles.retrieve({});
      const role = roles.find((r) => r.name === "Owner")?.key;
      if (role == null) throw new Error("no Owner role");
      const username = `windows-${id.create()}`;
      const { user: u } = await createTestUser(admin, {
        role,
        username,
        password: "old",
      });
      const opener = createTestClient({ username, password: "old" });
      await opener.auth.sessions.retrieve();
      const core = createCore("Local", { username, password: "" });
      const token = opener.auth.token;
      if (token == null) throw new Error("not logged in");
      keychain.set(keychainAccount(core), token);
      const { wrapper, store } = await createConsoleWrapper({
        client: null,
        preloadedState: createCoreState([core], core.key),
      });
      const windows: [Window, Window] = [{}, {}];
      for (const window of windows)
        render(
          <Connect>
            <Core.TokenKeeper />
            <Observe window={window} />
          </Connect>,
          { wrapper },
        );
      await waitFor(() =>
        windows.forEach((w) => expect(w.status?.details.authenticated).toBe(true)),
      );
      const clients = windows.map((w) => w.client);
      const [a, b] = clients;
      if (a == null || b == null) throw new Error("no client");
      return { admin, a, b, windows, keychain, race, store, core, u, clients };
    };

    // window A sets its own password, which ends the session both windows resumed
    const setPassword = async (a: Client, b: Client): Promise<string> => {
      await a.auth.passwords.set({ password: "new", currentPassword: "old" });
      const fresh = a.auth.token;
      if (fresh == null) throw new Error("no new session");
      // window B's next request finds its session ended
      await b.auth.sessions.retrieve().catch(() => undefined);
      return fresh;
    };

    it("should resume the session another window saved after a password change", async () => {
      const { a, b, windows, keychain, core, clients } = await openWindows();
      const fresh = await setPassword(a, b);
      await waitFor(() => {
        expect(b.auth.token).toBe(fresh);
        expect(windows[1].status?.details.authenticated).toBe(true);
      });
      expect(windows.map((w) => w.client)).toEqual(clients);
      expect(keychain.get(keychainAccount(core))).toBe(fresh);
    });

    it("should keep a token saved between another window's read and delete", async () => {
      const { a, b, windows, keychain, race, core, clients } = await openWindows();
      race.held = true;
      await a.auth.passwords.set({ password: "new", currentPassword: "old" });
      const fresh = a.auth.token;
      await waitFor(() => expect(race.pending?.[1]).toBe(fresh));
      await b.auth.sessions.retrieve().catch(() => undefined);
      // window B read the dead token, so it logs in again; window A's token stays
      await waitFor(() =>
        expect(windows[1].status).toMatchObject({
          variant: "error",
          details: { reason: "auth" },
        }),
      );
      await expectAlways(() => expect(keychain.get(keychainAccount(core))).toBe(fresh));
      expect(b.auth.token).not.toBe(fresh);
      expect(windows.map((w) => w.client)).toEqual(clients);
    });

    it("should resume the session under the new username after a rename", async () => {
      const { admin, a, b, windows, keychain, store, core, u, clients } =
        await openWindows();
      const [p] = await admin.auth.passwords.retrieve({
        subjects: [user.resourceID(u.key)],
      });
      const username = `renamed-${id.create()}`;
      await a.auth.passwords.dispatch(p.key, auth.password.setUsername({ username }));
      windows[0].setUsername?.(username);
      const renamed = { ...core, username };
      await waitFor(() =>
        expect(Session.Core.selectState(store.getState(), core.key)?.username).toBe(
          username,
        ),
      );
      await waitFor(() => expect(keychain.has(keychainAccount(renamed))).toBe(true));
      const fresh = await setPassword(a, b);
      await waitFor(() => {
        expect(b.auth.token).toBe(fresh);
        expect(windows[1].status?.details.authenticated).toBe(true);
      });
      expect(windows.map((w) => w.client)).toEqual(clients);
      expect(keychain.get(keychainAccount(renamed))).toBe(fresh);
      expect(keychain.has(keychainAccount(core))).toBe(false);
    });
  });
});
