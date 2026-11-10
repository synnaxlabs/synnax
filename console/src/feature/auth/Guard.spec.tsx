// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
} from "@synnaxlabs/client/testutil";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { clearMocks, mockIPC, mockWindows } from "@tauri-apps/api/mocks";
import { fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeEach, describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { Auth } from "@/feature/auth";
import { Core } from "@/feature/core";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import {
  CONNECTION_PARAMS,
  createCore,
  createCoreState,
} from "@/session/core/testutil";
import {
  createConsoleWrapper,
  createSessionConsoleWrapper,
  pinLocationOrigin,
  renderSuspended,
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

const DEAD_KEY = "dead";

const renderUnreachable = async (): Promise<void> => {
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

beforeEach(() => {
  mocks.engine = "web";
});

describe("auth guard", () => {
  it("should render children when a Core is already selected", async () => {
    await renderGuard(CORE_KEY);
    expect(screen.getByText("authenticated content")).toBeTruthy();
    expect(screen.queryByText("Log in")).toBeNull();
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
    expect(await screen.findByText("Incorrect username or password.")).toBeTruthy();
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
    expect(screen.queryByText("Incorrect username or password.")).toBeNull();
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
    await renderUnreachable();
    expect(await screen.findByText("Retry now", {}, { timeout: 10000 })).toBeTruthy();
    expect(screen.getAllByText("localhost:9098").length).toBeGreaterThan(0);
    expect(screen.queryByText("Preparing your workspace...")).toBeNull();
    expect(screen.queryByText("authenticated content")).toBeNull();
  });

  it("should not offer to edit the connection in a browser", async () => {
    await renderUnreachable();
    await screen.findByText("Retry now", {}, { timeout: 10000 });
    expect(screen.queryByText("Edit connection")).toBeNull();
  });

  it("should offer to edit the connection on desktop", async () => {
    mocks.engine = "tauri";
    mockWindows("main");
    // A dev build version skips the update check.
    mockIPC((cmd) => (cmd === "plugin:app|version" ? "0.0.0" : undefined));
    onTestFinished(clearMocks);
    await renderUnreachable();
    expect(
      await screen.findByText("Edit connection", {}, { timeout: 10000 }),
    ).toBeTruthy();
  });

  it("should disable the retry button while a check is in flight", async () => {
    await renderUnreachable();
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
        const granted = Access.useCreateGranted(channel.TYPE_RESOURCE_ID);
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

  it("should return to the login surface through Log out", async () => {
    const { proxy, client } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onRetry: () => {},
    });
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: {
        [Session.Core.SLICE_NAME]: { ...CoreState(), selected: CORE_KEY },
      },
    });
    // A Core that stays unreachable fails the permissions load.
    await proxy.sever();
    await renderSuspended(
      <Auth.Guard>
        <Auth.ConnectionGuard>
          <span>authenticated content</span>
        </Auth.ConnectionGuard>
      </Auth.Guard>,
      { wrapper, onCaughtError: () => {} },
    );
    expect(await screen.findByRole("button", { name: "Retry" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Log out" }));
    await waitFor(() =>
      expect(screen.getAllByText("Log in").length).toBeGreaterThan(0),
    );
    expect(screen.queryByText("authenticated content")).toBeNull();
  });

  const PASSWORD = "password123";

  const createUserWithRole = async (
    client: ReturnType<typeof createTestClient>,
    roleName: string,
  ): Promise<string> => {
    const roles = await client.access.roles.retrieve({});
    const role = roles.find(({ name }) => name === roleName);
    if (role == null) throw new Error(`${roleName} role not provisioned`);
    const username = uniqueName("role_user");
    await client.users.create({ username, password: PASSWORD, role: role.key });
    return username;
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
      { wrapper },
    );
  };

  it.each(["Owner", "Engineer", "Operator", "Viewer", "Driver"])(
    "should render the workspace for the %s role",
    async (roleName) => {
      const client = createTestClient();
      const username = await createUserWithRole(client, roleName);
      await renderGuardedContent();
      submitCredentials(username, PASSWORD);
      expect(
        await screen.findByText("authenticated content", {}, { timeout: 10000 }),
      ).toBeTruthy();
    },
  );
});
