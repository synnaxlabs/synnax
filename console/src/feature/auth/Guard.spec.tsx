// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, connection } from "@synnaxlabs/client";
import { createTestClient, createTestUser } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { Access, Synnax } from "@synnaxlabs/pluto";
import {
  act,
  fireEvent,
  render,
  renderHook,
  screen,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, onTestFinished } from "vitest";

import { Auth } from "@/feature/auth";
import { Core } from "@/feature/core";
import { LOCAL_SUBJECT, type OIDCServer, serveOIDC } from "@/platform/core/testutil";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import {
  CONNECTION_PARAMS,
  createCore,
  createCoreState,
} from "@/session/core/testutil";
import {
  CaptureStatuses,
  createSessionConsoleWrapper,
  createStatusConsoleWrapper,
  pinLocationOrigin,
  renderWithConsole,
  type TestStore,
  uniqueName,
} from "@/testutil";

const LOCAL = createCore("Local", { username: "", password: "" });
const CORE_KEY = LOCAL.key;

const CoreState = (): Session.Core.SliceState => ({
  ...Session.Core.ZERO_SLICE_STATE,
  cores: { [CORE_KEY]: LOCAL },
});

const renderGuard = async (selected?: string): Promise<TestStore> => {
  const { store } = await renderWithConsole(
    <Auth.Guard>
      <span>authenticated content</span>
    </Auth.Guard>,
    {
      preloadedState: {
        [Session.Core.SLICE_NAME]: { ...CoreState(), selected },
      },
    },
  );
  return store;
};

const submitCredentials = (username: string, password: string): void => {
  fireEvent.change(screen.getByPlaceholderText("synnax"), {
    target: { value: username },
  });
  fireEvent.change(screen.getByPlaceholderText("seldon"), {
    target: { value: password },
  });
  fireEvent.click(findButton("Log in"));
};

describe("auth guard", () => {
  it("should render children when a Core is already selected", async () => {
    await renderGuard(CORE_KEY);
    expect(screen.getByText("authenticated content")).toBeTruthy();
    expect(screen.queryByText("Log in")).toBeNull();
  });

  it("should finish a browser login even when a Core is already selected", async () => {
    sessionStorage.setItem(
      "synnax-oidc-login",
      JSON.stringify({
        purpose: "login",
        core: CORE_KEY,
        provider: "company-sso",
        verifier: "verifier",
        state: "state",
        redirectUri: "https://core.example.com/",
      }),
    );
    window.history.replaceState(null, "", "/?code=abc&state=state");
    onTestFinished(() => {
      sessionStorage.clear();
      window.history.replaceState(null, "", "/");
    });
    const store = await renderGuard(CORE_KEY);
    await waitFor(() =>
      expect(store.getState()[Session.Core.SLICE_NAME].cores[CORE_KEY].oidc).toEqual({
        provider: "company-sso",
        code: "abc",
        verifier: "verifier",
        redirectUri: "https://core.example.com/",
      }),
    );
    expect(window.location.search).toBe("");
  });

  describe("a provider link's return", () => {
    const PROVIDER = "company-sso";
    const VERIFIER = "verifier";
    const REDIRECT_URI = "https://core.example.com/";

    const returnFromProvider = (
      search: string,
      core = CORE_KEY,
      subject = LOCAL_SUBJECT.key,
    ): void => {
      sessionStorage.setItem(
        "synnax-oidc-login",
        JSON.stringify({
          purpose: "link",
          subject,
          core,
          provider: PROVIDER,
          verifier: VERIFIER,
          state: "state",
          redirectUri: REDIRECT_URI,
        }),
      );
      window.history.replaceState(null, "", search);
      onTestFinished(() => {
        sessionStorage.clear();
        window.history.replaceState(null, "", "/");
      });
    };

    const renderReturn = async ({ address }: OIDCServer) => {
      let statuses: Status.NotificationSpec[] = [];
      const { wrapper, store } = await createStatusConsoleWrapper({
        client: createTestClient(address),
        status: connection.DEFAULT_STATUS,
        preloadedState: {
          [Session.Core.SLICE_NAME]: { ...CoreState(), selected: CORE_KEY },
        },
      });
      render(
        <>
          <CaptureStatuses onStatuses={(next) => (statuses = next)} />
          <Auth.Guard>
            <span>authenticated content</span>
          </Auth.Guard>
        </>,
        { wrapper },
      );
      return { store, statuses: () => statuses.map((s) => s.message) };
    };

    it("should link the account and keep the session", async () => {
      const server = await serveOIDC();
      returnFromProvider("/?code=abc&state=state");
      const { store, statuses } = await renderReturn(server);
      await waitFor(() => expect(statuses()).toContain("Linked provider account"));
      expect(server.linked).toEqual([
        {
          provider: PROVIDER,
          code: "abc",
          verifier: VERIFIER,
          redirect_uri: REDIRECT_URI,
        },
      ]);
      const core = store.getState()[Session.Core.SLICE_NAME].cores[CORE_KEY];
      expect(core.oidc).toBeUndefined();
      expect(Session.Core.selectSelectedKey(store.getState())).toBe(CORE_KEY);
      expect(window.location.search).toBe("");
      expect(screen.getByText("authenticated content")).toBeTruthy();
    });

    it("should report a link the Core refuses", async () => {
      const server = await serveOIDC({ linkRefused: true });
      returnFromProvider("/?code=abc&state=state");
      const { statuses } = await renderReturn(server);
      await waitFor(() =>
        expect(statuses()).toContain("Failed to link provider account"),
      );
      expect(statuses()).not.toContain("Linked provider account");
    });

    it("should report the provider's refusal as a failed link", async () => {
      const server = await serveOIDC();
      returnFromProvider("/?error=access_denied&state=state");
      const { statuses } = await renderReturn(server);
      await waitFor(() =>
        expect(statuses()).toContain("Failed to link provider account"),
      );
      expect(server.linked).toEqual([]);
    });

    it("should refuse a link another user started", async () => {
      const server = await serveOIDC();
      returnFromProvider("/?code=abc&state=state", CORE_KEY, "another-user");
      const { statuses } = await renderReturn(server);
      await waitFor(() =>
        expect(statuses()).toContain("Failed to link provider account"),
      );
      expect(server.linked).toEqual([]);
    });

    it("should refuse a link that another Core started", async () => {
      const server = await serveOIDC();
      returnFromProvider("/?code=abc&state=state", "another-core");
      const { statuses } = await renderReturn(server);
      await waitFor(() =>
        expect(statuses()).toContain("Failed to link provider account"),
      );
      expect(server.linked).toEqual([]);
    });
  });

  describe("an invite's return", () => {
    const CODE = "syi_code";

    const renderReturn = async (server: OIDCServer) => {
      const core = { ...LOCAL, ...server.address };
      sessionStorage.setItem(
        "synnax-oidc-login",
        JSON.stringify({
          purpose: "invite",
          invite: CODE,
          core: CORE_KEY,
          provider: "company-sso",
          verifier: "verifier",
          state: "state",
          redirectUri: "https://core.example.com/",
        }),
      );
      window.history.replaceState(null, "", "/?code=abc&state=state");
      onTestFinished(() => {
        sessionStorage.clear();
        window.history.replaceState(null, "", "/");
      });
      let statuses: Status.NotificationSpec[] = [];
      const { wrapper, store } = await createStatusConsoleWrapper({
        client: null,
        status: connection.DEFAULT_STATUS,
        preloadedState: {
          [Session.Core.SLICE_NAME]: {
            ...Session.Core.ZERO_SLICE_STATE,
            cores: { [CORE_KEY]: core },
          },
        },
      });
      render(
        <>
          <CaptureStatuses onStatuses={(next) => (statuses = next)} />
          <Auth.Guard>
            <span>authenticated content</span>
          </Auth.Guard>
        </>,
        { wrapper },
      );
      return { core, store, statuses: () => statuses };
    };

    it("should redeem the code instead of logging in or linking", async () => {
      const server = await serveOIDC();
      const { store } = await renderReturn(server);
      await waitFor(() =>
        expect(Session.Core.selectSelectedKey(store.getState())).toBe(CORE_KEY),
      );
      expect(server.redeemed).toEqual([
        expect.objectContaining({
          token: CODE,
          cookie: false,
          credentials: expect.objectContaining({ code: "abc" }),
        }),
      ]);
      expect(server.sessionsCreated).toBe(0);
      expect(server.linked).toEqual([]);
      const record = store.getState()[Session.Core.SLICE_NAME].cores[CORE_KEY];
      expect(record.oidc).toBeUndefined();
      expect(record.token).toBe("sys_invited");
      expect(window.location.search).toBe("");
    });

    it("should leave the session to a cookie on a page the Core serves", async () => {
      const server = await serveOIDC();
      pinLocationOrigin(
        `http://${server.address.host}:${server.address.port}`,
        "?code=abc&state=state",
      );
      const { store } = await renderReturn(server);
      await waitFor(() =>
        expect(Session.Core.selectSelectedKey(store.getState())).toBe(CORE_KEY),
      );
      expect(server.redeemed[0].cookie).toBe(true);
      const record = store.getState()[Session.Core.SLICE_NAME].cores[CORE_KEY];
      expect(record.token).toBeUndefined();
    });

    it("should explain a refused code and open no session", async () => {
      const server = await serveOIDC({ inviteRefused: true });
      const { store, statuses } = await renderReturn(server);
      await waitFor(() =>
        expect(statuses()).toContainEqual(
          expect.objectContaining({
            message: "Failed to redeem invite code",
            description:
              "This invite code is not valid. Ask your administrator for a new one.",
          }),
        ),
      );
      expect(Session.Core.selectSelectedKey(store.getState())).toBeUndefined();
      const record = store.getState()[Session.Core.SLICE_NAME].cores[CORE_KEY];
      expect(record.token).toBeUndefined();
    });
  });

  it("should render the login screen when no Core is selected", async () => {
    await renderGuard();
    expect(screen.getAllByText("Log in").length).toBeGreaterThan(0);
    expect(screen.queryByText("authenticated content")).toBeNull();
  });

  it("should select the Core after a successful login", async () => {
    pinLocationOrigin(`http://${CONNECTION_PARAMS.host}:${CONNECTION_PARAMS.port}`);
    const store = await renderGuard();
    submitCredentials("synnax", "seldon");
    // A session sees one served Core, so every login lands on that one entry rather
    // than minting another.
    await waitFor(() =>
      expect(Session.Core.selectSelectedKey(store.getState())).toBe(
        Session.Core.SERVED_KEY,
      ),
    );
    expect(await screen.findByText("authenticated content")).toBeTruthy();
    const state = store.getState();
    const key = Session.Core.selectSelectedKey(state);
    if (key == null) throw new Error("no Core selected");
    const core = Session.Core.selectState(state, key);
    expect(core?.username).toBe("synnax");
  });

  it("should return to the login surface when credentials are rejected", async () => {
    pinLocationOrigin(`http://${CONNECTION_PARAMS.host}:${CONNECTION_PARAMS.port}`);
    const { wrapper, store } = await createSessionConsoleWrapper({ client: null });
    render(
      <Session.SettledProvider>
        <Auth.Guard>
          <Auth.ConnectionGuard>
            <Core.Badge />
            <span>authenticated content</span>
          </Auth.ConnectionGuard>
        </Auth.Guard>
      </Session.SettledProvider>,
      { wrapper },
    );
    submitCredentials("synnax", uniqueName("wrong"));
    await waitFor(() =>
      expect(Session.Core.selectSelectedKey(store.getState())).toBeDefined(),
    );
    const key = Session.Core.selectSelectedKey(store.getState());
    expect(await screen.findByText(/invalid credentials/i)).toBeTruthy();
    expect(screen.getAllByText("Log in").length).toBeGreaterThan(0);
    expect(screen.queryByText("authenticated content")).toBeNull();
    submitCredentials("synnax", "seldon");
    await waitFor(
      () => expect(document.querySelector(".lyra--status-success")).toBeTruthy(),
      { timeout: 10000 },
    );
    // The login sequence swaps the Core and project partitions in turn; the workspace
    // settles once the second swap hydrates.
    expect(
      await screen.findByText("authenticated content", {}, { timeout: 10000 }),
    ).toBeTruthy();
    expect(screen.queryByText(/invalid credentials/i)).toBeNull();
    // A retry after a rejected login reuses the served entry rather than minting a
    // duplicate.
    const selected = Session.Core.selectSelectedKey(store.getState());
    expect(selected).toBe(key);
    const added = Object.values(store.getState()[Session.Core.SLICE_NAME].cores)
      .map(({ key: k }) => k)
      .filter((k) => k !== "LOCAL" && k !== "DEMO");
    expect(added).toEqual([selected]);
  });

  it("should surface connection trouble at a cold start against an unreachable Core", async () => {
    const DEAD_KEY = "dead";
    const { wrapper } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: {
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          cores: {
            [DEAD_KEY]: createCore("Dead", {
              key: DEAD_KEY,
              host: "localhost",
              port: 9098,
            }),
          },
          selected: DEAD_KEY,
        },
      },
    });
    render(
      <Session.SettledProvider>
        <Auth.Guard>
          <Auth.ConnectionGuard>
            <span>authenticated content</span>
          </Auth.ConnectionGuard>
        </Auth.Guard>
      </Session.SettledProvider>,
      { wrapper },
    );
    expect(await screen.findByText("Retry now", {}, { timeout: 10000 })).toBeTruthy();
    expect(screen.getAllByText("localhost:9098").length).toBeGreaterThan(0);
    expect(screen.queryByText("Preparing your workspace...")).toBeNull();
    expect(screen.queryByText("authenticated content")).toBeNull();
  });

  it("should disable the retry button while a check is in flight", async () => {
    const DEAD_KEY = "dead";
    const { wrapper } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: {
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          cores: {
            [DEAD_KEY]: createCore("Dead", {
              key: DEAD_KEY,
              host: "localhost",
              port: 9098,
            }),
          },
          selected: DEAD_KEY,
        },
      },
    });
    render(
      <Session.SettledProvider>
        <Auth.Guard>
          <Auth.ConnectionGuard>
            <span>authenticated content</span>
          </Auth.ConnectionGuard>
        </Auth.Guard>
      </Session.SettledProvider>,
      { wrapper },
    );
    await screen.findByText("Retry now", {}, { timeout: 10000 });
    // The scheduled retries raise the same flag, so the assertion starts from a
    // gap between them rather than from whatever the countdown happens to be doing.
    await waitFor(
      () => expect(findButton("Retry now").getAttribute("aria-disabled")).toBeNull(),
      { timeout: 10000 },
    );
    fireEvent.click(findButton("Retry now"));
    await waitFor(() =>
      expect(findButton("Retry now").getAttribute("aria-disabled")).toBe("true"),
    );
  });
});

describe("connection guard permissions", () => {
  const createGuardWrapper = async (): Promise<FC<PropsWithChildren>> => {
    const { wrapper: SessionWrapper } = await createSessionConsoleWrapper({
      client: null,
      // The connection guard needs credentials, which LOCAL deliberately lacks.
      preloadedState: createCoreState(
        [createCore("Local", { key: CORE_KEY })],
        CORE_KEY,
      ),
    });
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <SessionWrapper>
        <Session.SettledProvider>
          <Auth.Guard>
            <Auth.ConnectionGuard>{children}</Auth.ConnectionGuard>
          </Auth.Guard>
        </Session.SettledProvider>
      </SessionWrapper>
    );
    Wrapper.displayName = "GuardWrapper";
    return Wrapper;
  };

  it("should never render a guarded child against an empty policy set", async () => {
    const wrapper = await createGuardWrapper();
    const verdicts: boolean[] = [];
    const { result } = renderHook(
      () => {
        const client = Synnax.use();
        const granted = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
        if (client != null) verdicts.push(granted);
        return granted;
      },
      { wrapper },
    );
    await waitFor(() => expect(result.current).toBe(true), { timeout: 10000 });
    expect(verdicts).not.toContain(false);
  });

  it("should hold the splash while the policies load", async () => {
    const wrapper = await createGuardWrapper();
    render(<span>authenticated content</span>, { wrapper });
    expect(screen.queryByText("authenticated content")).toBeNull();
    expect(
      await screen.findByText("authenticated content", {}, { timeout: 10000 }),
    ).toBeTruthy();
  });

  const PASSWORD = "password123";

  const assignRole = async (
    client: ReturnType<typeof createTestClient>,
    user: string,
    roleName: string,
  ): Promise<void> => {
    const roles = await client.access.roles.retrieve({});
    const role = roles.find(({ name }) => name === roleName);
    if (role == null) throw new Error(`${roleName} role not provisioned`);
    await client.access.roles.assign({ user, role: role.key });
  };

  const createUserWithRole = async (
    client: ReturnType<typeof createTestClient>,
    roleName: string,
  ): Promise<{ username: string; key: string }> => {
    const { user, username } = await createTestUser(client, {
      username: uniqueName("role_user"),
      password: PASSWORD,
    });
    await assignRole(client, user.key, roleName);
    return { username, key: user.key };
  };

  const renderGuardedContent = async (): Promise<void> => {
    pinLocationOrigin(`http://${CONNECTION_PARAMS.host}:${CONNECTION_PARAMS.port}`);
    const { wrapper } = await createSessionConsoleWrapper({ client: null });
    render(
      <Session.SettledProvider>
        <Auth.Guard>
          <Auth.ConnectionGuard>
            <span>authenticated content</span>
          </Auth.ConnectionGuard>
        </Auth.Guard>
      </Session.SettledProvider>,
      // A denied policy fetch reaches the denial boundary.
      { wrapper, onCaughtError: () => {} },
    );
  };

  it("should explain a denied policy fetch and recover through Retry", async () => {
    const client = createTestClient();
    // The Driver role cannot retrieve policies, so the permissions fetch is denied.
    const { username, key } = await createUserWithRole(client, "Driver");
    await renderGuardedContent();
    submitCredentials(username, PASSWORD);
    expect(
      await screen.findByText("Console access denied", {}, { timeout: 10000 }),
    ).toBeTruthy();
    expect(screen.getByText(`Check role permissions for ${username}.`)).toBeTruthy();
    // The denial renders the calm surface, not the crash fallback.
    expect(screen.queryByText("Stack trace")).toBeNull();
    expect(screen.queryByText("authenticated content")).toBeNull();
    await assignRole(client, key, "Operator");
    await act(async () => fireEvent.click(findButton("Retry")));
    expect(
      await screen.findByText("authenticated content", {}, { timeout: 10000 }),
    ).toBeTruthy();
  });

  it("should return to the login surface through Log out", async () => {
    const client = createTestClient();
    const { username } = await createUserWithRole(client, "Driver");
    await renderGuardedContent();
    submitCredentials(username, PASSWORD);
    await screen.findByText("Console access denied", {}, { timeout: 10000 });
    fireEvent.click(findButton("Log out"));
    await waitFor(() =>
      expect(screen.getAllByText("Log in").length).toBeGreaterThan(0),
    );
    expect(screen.queryByText("Console access denied")).toBeNull();
    expect(screen.queryByText("authenticated content")).toBeNull();
  });

  it.each(["Owner", "Engineer", "Operator", "Viewer"])(
    "should render the workspace for the %s role",
    async (roleName) => {
      const client = createTestClient();
      const { username } = await createUserWithRole(client, roleName);
      await renderGuardedContent();
      submitCredentials(username, PASSWORD);
      expect(
        await screen.findByText("authenticated content", {}, { timeout: 10000 }),
      ).toBeTruthy();
      expect(screen.queryByText("Console access denied")).toBeNull();
    },
  );
});
