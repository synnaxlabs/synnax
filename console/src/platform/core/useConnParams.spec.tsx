// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { auth, type connection, type Synnax as Client, user } from "@synnaxlabs/client";
import { createTestClient, createTestUser } from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { expectAlways } from "@synnaxlabs/x/testutil";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, render, waitFor } from "@testing-library/react";
import type { PropsWithChildren, ReactElement } from "react";
import { beforeEach, describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Core } from "@/platform/core";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import {
  createConsoleWrapper,
  disableActEnvironment,
  pinLocationOrigin,
  renderHookWithConsole,
} from "@/testutil";

const CORE = createCore("Alpha", { host: "example.com", port: 9095, password: "" });

const renderParams = async (core = CORE) =>
  await renderHookWithConsole(() => Core.useConnParams(), {
    preloadedState: createCoreState([core], core.key),
  });

describe("Core.useConnParams", () => {
  beforeEach(() => {
    mocks.engine = "web";
  });

  it("should give no parameters while no Core is selected", async () => {
    const { result } = await renderHookWithConsole(() => Core.useConnParams(), {
      preloadedState: createCoreState([CORE]),
    });
    expect(result.current).toBeUndefined();
  });

  it("should resume from the session cookie on a page the Core serves", async () => {
    pinLocationOrigin("http://example.com:9095");
    const { result } = await renderParams();
    expect(result.current).toMatchObject({
      host: "example.com",
      port: 9095,
      sessionCookie: true,
      sessionPersistent: true,
    });
    expect(result.current?.password).toBeUndefined();
    expect(result.current?.token).toBeUndefined();
  });

  it("should not ask for a cookie from a Core on another origin", async () => {
    pinLocationOrigin("http://localhost:5173");
    const { result } = await renderParams({ ...CORE, password: "typed" });
    expect(result.current?.sessionCookie).toBe(false);
    expect(result.current?.sessionPersistent).toBe(false);
  });

  it("should ask for a cookie from a secure Core on the page's default port", async () => {
    pinLocationOrigin("https://example.com");
    const { result } = await renderParams({ ...CORE, port: 443, secure: true });
    expect(result.current?.sessionCookie).toBe(true);
  });

  it("should not ask for a cookie when the page and the Core differ in scheme", async () => {
    pinLocationOrigin("https://example.com:9095");
    const { result } = await renderParams({ ...CORE, password: "typed" });
    expect(result.current?.sessionCookie).toBe(false);
  });

  it("should keep the Core's name and cluster out of the connection", async () => {
    const { result } = await renderParams({ ...CORE, password: "typed" });
    expect(result.current).not.toHaveProperty("name");
    expect(result.current).not.toHaveProperty("clusterKey");
    expect(result.current).not.toHaveProperty("key");
  });

  it("should log in with an OpenID Connect code the user just received", async () => {
    const oidc = {
      provider: "idp",
      code: "code",
      verifier: "verifier",
      redirectUri: "https://example.com:9095/",
    };
    const { result } = await renderParams({ ...CORE, username: "", oidc });
    expect(result.current).toMatchObject({ oidc });
    expect(result.current).not.toHaveProperty("username");
  });

  it("should leave out an empty username once the code is spent", async () => {
    pinLocationOrigin("http://example.com:9095");
    const { result } = await renderParams({ ...CORE, username: "" });
    expect(result.current).toBeDefined();
    expect(result.current?.username).toBeUndefined();
  });

  it("should give no parameters for a Core on another origin with no login", async () => {
    pinLocationOrigin("http://localhost:5173");
    const { result } = await renderParams();
    expect(result.current).toBeUndefined();
  });

  it("should log in with a password the user entered", async () => {
    const { result } = await renderParams({ ...CORE, password: "typed" });
    expect(result.current?.password).toBe("typed");
  });

  it("should wait for the keychain and resume from its token on desktop", async () => {
    mocks.engine = "tauri";
    const reads: unknown[] = [];
    mockIPC((cmd, args) => {
      if (cmd !== "keychain_get") return undefined;
      reads.push(args);
      return "sys_saved";
    });
    onTestFinished(clearMocks);
    const { result } = await renderParams();
    expect(result.current).toBeUndefined();
    await waitFor(() => expect(result.current?.token).toBe("sys_saved"));
    expect(result.current?.sessionCookie).toBeUndefined();
    expect(result.current?.sessionPersistent).toBe(true);
    expect(result.current).not.toHaveProperty("name");
    expect(reads).toEqual([{ account: `http://synnax@example.com:9095#${CORE.key}` }]);
  });

  it("should read the keychain again after a login in another window", async () => {
    mocks.engine = "tauri";
    let saved = "sys_old";
    mockIPC((cmd) => (cmd === "keychain_get" ? saved : undefined));
    onTestFinished(clearMocks);
    const { result, store } = await renderParams();
    await waitFor(() => expect(result.current?.token).toBe("sys_old"));
    saved = "sys_new";
    act(() => {
      store.dispatch(Session.Core.recordLogin({ key: CORE.key, at: Date.now() }));
    });
    await waitFor(() => expect(result.current?.token).toBe("sys_new"));
  });

  it("should not resume over plain text from a token saved for TLS", async () => {
    mocks.engine = "tauri";
    let reads = 0;
    mockIPC((cmd) => {
      if (cmd !== "keychain_get") return undefined;
      reads += 1;
      return reads === 1 ? "sys_saved" : null;
    });
    onTestFinished(clearMocks);
    const secure = { ...CORE, secure: true };
    const { result, store } = await renderParams(secure);
    await waitFor(() => expect(result.current?.token).toBe("sys_saved"));
    act(() => {
      store.dispatch(Session.Core.set({ ...secure, secure: false }));
    });
    await waitFor(() => expect(result.current?.secure).toBe(false));
    expect(result.current?.token).toBeUndefined();
  });

  it("should keep separate tokens for two saved Cores at one address", async () => {
    mocks.engine = "tauri";
    const alpha = { ...CORE, username: "" };
    const bravo = createCore("Bravo", {
      host: "example.com",
      port: 9095,
      password: "",
      username: "",
    });
    mockIPC((cmd, args) => {
      if (cmd !== "keychain_get") return undefined;
      const { account } = args as { account: string };
      if (account.endsWith(`#${alpha.key}`)) return "sys_alpha";
      if (account.endsWith(`#${bravo.key}`)) return "sys_bravo";
      return null;
    });
    onTestFinished(clearMocks);
    const { result, store } = await renderHookWithConsole(() => Core.useConnParams(), {
      preloadedState: createCoreState([alpha, bravo], alpha.key),
    });
    await waitFor(() => expect(result.current?.token).toBe("sys_alpha"));
    act(() => {
      store.dispatch(Session.Core.select(bravo.key));
    });
    await waitFor(() => expect(result.current?.token).toBe("sys_bravo"));
  });

  it("should connect without a token when the keychain holds none", async () => {
    mocks.engine = "tauri";
    mockIPC((cmd) => (cmd === "keychain_get" ? null : undefined));
    onTestFinished(clearMocks);
    const { result } = await renderParams();
    await waitFor(() => expect(result.current).toBeDefined());
    expect(result.current?.token).toBeUndefined();
  });

  it("should keep the parameters when the user's own password changes", async () => {
    const { result, store } = await renderParams({ ...CORE, password: "old" });
    const before = result.current;
    act(() => {
      store.dispatch(Session.Core.set({ ...CORE, password: "new" }));
    });
    expect(Session.Core.selectState(store.getState(), CORE.key)?.password).toBe("new");
    expect(result.current).toBe(before);
  });

  it("should give new parameters for a new login", async () => {
    const { result, store } = await renderParams({ ...CORE, password: "old" });
    act(() => {
      store.dispatch(
        Session.Core.set({ ...CORE, username: "other", password: "new", lastLogin: 1 }),
      );
    });
    expect(result.current).toMatchObject({ username: "other", password: "new" });
  });

  it("should keep the parameters on desktop while the user renames themselves", async () => {
    mocks.engine = "tauri";
    mockIPC((cmd) => (cmd === "keychain_get" ? "sys_saved" : undefined));
    onTestFinished(clearMocks);
    const seen: unknown[] = [];
    const { result, store } = await renderHookWithConsole(
      () => {
        const params = Core.useConnParams();
        seen.push(params);
        return { params, setUsername: Core.useSetUsername() };
      },
      { preloadedState: createCoreState([CORE], CORE.key) },
    );
    await waitFor(() => expect(result.current.params?.token).toBe("sys_saved"));
    const before = result.current.params;
    seen.length = 0;
    act(() => result.current.setUsername("after"));
    await waitFor(() =>
      expect(Session.Core.selectState(store.getState(), CORE.key)?.username).toBe(
        "after",
      ),
    );
    expect(seen.every((params) => params === before)).toBe(true);
  });

  describe("against a live Core", () => {
    interface Seen {
      clients: Set<Client | null>;
      status?: connection.Status;
      core?: Session.Core.Core;
      setUsername?: (username: string) => void;
    }

    const Observe = ({ seen }: { seen: Seen }): null => {
      seen.clients.add(Synnax.use());
      seen.status = Synnax.useConnectionStatus();
      seen.core = Session.Core.useSelectSelected();
      seen.setUsername = Core.useSetUsername();
      return null;
    };

    const Connect = ({ children }: PropsWithChildren): ReactElement => (
      <Synnax.Provider connParams={Core.useConnParams()}>{children}</Synnax.Provider>
    );

    // a user logged in through the selected Core's record, who may rename themselves
    const renderSession = async () => {
      disableActEnvironment();
      const admin = createTestClient();
      const roles = await admin.access.roles.retrieve({});
      const role = roles.find((r) => r.name === "Owner")?.key;
      if (role == null) throw new Error("no Owner role");
      const username = `self-${id.create()}`;
      const { user: u } = await createTestUser(admin, {
        role,
        username,
        password: "old",
      });
      const core = createCore("Local", { username, password: "old" });
      const { wrapper, store } = await createConsoleWrapper({
        client: null,
        preloadedState: createCoreState([core], core.key),
      });
      const seen: Seen = { clients: new Set() };
      render(
        <Connect>
          <Observe seen={seen} />
        </Connect>,
        { wrapper },
      );
      await waitFor(() => {
        expect(seen.status?.details.authenticated).toBe(true);
        expect(seen.status?.details.epoch).toBeGreaterThanOrEqual(1);
      });
      const client = [...seen.clients].find((c) => c != null);
      if (client == null) throw new Error("no client");
      seen.clients.clear();
      return { admin, client, core, seen, store, u };
    };

    it("should keep the client when the user sets their own password", async () => {
      const { client, core, seen, store } = await renderSession();
      const { epoch } = client.connection.status.details;
      await client.auth.passwords.set({ password: "new", currentPassword: "old" });
      store.dispatch(Session.Core.set({ ...core, password: "new" }));
      await waitFor(() => expect(seen.core?.password).toBe("new"));
      await expectAlways(() => {
        expect([...seen.clients]).toEqual([client]);
        expect(seen.status?.details.epoch).toBe(epoch);
      });
    });

    it("should keep the client when the user renames themselves", async () => {
      const { admin, client, seen, u } = await renderSession();
      const { epoch } = client.connection.status.details;
      const [p] = await admin.auth.passwords.retrieve({
        subjects: [user.resourceID(u.key)],
      });
      const username = `renamed-${id.create()}`;
      await client.auth.passwords.dispatch(
        p.key,
        auth.password.setUsername({ username }),
      );
      seen.setUsername?.(username);
      await waitFor(() => expect(seen.core?.username).toBe(username));
      await expectAlways(() => {
        expect([...seen.clients]).toEqual([client]);
        expect(seen.status?.details.epoch).toBe(epoch);
      });
    });
  });
});
