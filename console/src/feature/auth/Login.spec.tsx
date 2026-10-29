// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createServer } from "node:http";
import { type AddressInfo } from "node:net";

import { type auth, AuthError, connection } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { expectAlways } from "@synnaxlabs/x/testutil";
import { clearMocks, mockIPC, mockWindows } from "@tauri-apps/api/mocks";
import { fireEvent, render, screen } from "@testing-library/react";
import {
  afterEach,
  assert,
  beforeEach,
  describe,
  expect,
  it,
  onTestFinished,
  vi,
} from "vitest";

// A serving Core skips the Core step, and detection reports one for every browser
// build. The module is the only seam.
vi.mock("@/platform/core/detectConnection", () => ({
  detectConnection: () => null,
}));

const runtime = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, runtime);
});

import { Login } from "@/feature/auth/Login";
import { Session } from "@/session";
import { createCore } from "@/session/core/testutil";
import {
  CaptureStatuses,
  createConsoleWrapper,
  createSessionConsoleWrapper,
  createStatusConsoleWrapper,
  disableActEnvironment,
  getBySelector,
} from "@/testutil";

// the local auth server answers outside any act
disableActEnvironment();

// Dead ports: the list checks each Core, and no assertion here reads the result.
const ALPHA = createCore("alpha", { name: "Alpha", port: 9098, username: "ada" });
const BETA = createCore("beta", { name: "Beta", port: 9099, username: "grace" });

// No test Core turns passwords off or accepts a provider, so a local server answers the
// methods request. The Console asks once for the methods and again for the providers;
// null providers fail that second request. A token opens a session for any code, and
// without one the server rejects every code.
const serveAuth = async (
  methods: string[],
  providers: auth.OIDCLogin[] | null = [],
  onServed?: () => void,
  token?: string,
): Promise<Session.Core.Core> => {
  let methodsRequests = 0;
  const server = createServer((req, res) => {
    req.resume();
    req.on("end", () => {
      if (token != null && req.url?.endsWith("/auth/session/create") === true) {
        res.setHeader("Content-Type", "application/json");
        res.end(JSON.stringify({ token, session: SESSION }));
        return;
      }
      const isMethods = req.url?.endsWith("/auth/methods") === true;
      if (isMethods) methodsRequests += 1;
      if (!isMethods || (providers == null && methodsRequests > 1)) {
        res.statusCode = 404;
        res.end();
        return;
      }
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify({ methods, providers: providers ?? [] }), onServed);
    });
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  onTestFinished(
    async () => await new Promise<void>((resolve) => server.close(() => resolve())),
  );
  return { ...ALPHA, host: "127.0.0.1", port: (server.address() as AddressInfo).port };
};

const SESSION = {
  key: "8b7c3e52-6f0a-4c1d-9e2b-3a4f5d6e7f80",
  subject: { type: "user", key: "1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f" },
  credential: "2d3e4f5a-6b7c-4d8e-9f0a-1b2c3d4e5f60",
  lastActive: 0,
};

const PROVIDER: auth.OIDCLogin = {
  provider: "company-sso",
  authorizationEndpoint: "https://sso.example.com/authorize",
  clientId: "synnax",
};

const REJECTED: connection.Status = {
  ...connection.DEFAULT_STATUS,
  variant: "error",
  name: ALPHA.name,
  message: "Failed to authenticate",
  description: "invalid credentials",
  details: {
    ...connection.DEFAULT_STATUS.details,
    reason: "auth",
    error: new AuthError("invalid credentials"),
  },
};

const BACK = ".console-login__back";
const USERNAME_PLACEHOLDER = "synnax";

// The desktop specs leave out the session's client: it connects through the keychain,
// which the session wrapper bypasses.
const renderLogin = async (
  alpha = ALPHA,
  createWrapper = createSessionConsoleWrapper,
) => {
  let statuses: Status.NotificationSpec[] = [];
  const { wrapper, store } = await createWrapper({
    client: null,
    preloadedState: {
      [Session.Core.SLICE_NAME]: {
        ...Session.Core.ZERO_SLICE_STATE,
        cores: { [alpha.key]: alpha, [BETA.key]: BETA },
        selected: undefined,
      },
    },
  });
  const { container } = render(
    <>
      <CaptureStatuses onStatuses={(next) => (statuses = next)} />
      <Login />
    </>,
    { wrapper },
  );
  return { container, store, statuses: () => statuses };
};

// The desktop shell of a dev build: its main window, keychain, loopback listener, and
// the browser a login hands the provider page to. The browser returns at once with the
// state the Console waits for.
const fakeDesktop = (onWait?: () => void) => {
  const ipc = { listens: 0, saved: new Map<string, string>() };
  mockWindows("main");
  mockIPC((cmd, args) => {
    const { account, secret, state } = args as Record<string, string>;
    switch (cmd) {
      case "plugin:app|version":
        return "0.0.0";
      case "keychain_set":
        ipc.saved.set(account, secret);
        return undefined;
      case "oidc_listen":
        ipc.listens += 1;
        return 4242;
      case "oidc_wait":
        onWait?.();
        return `code=abc&state=${state}`;
      default:
        return undefined;
    }
  });
  onTestFinished(clearMocks);
  return ipc;
};

describe("auth/Login", () => {
  // jsdom serves no secure context; provider login needs one
  const setSecureContext = (secure: boolean) =>
    Object.defineProperty(window, "isSecureContext", {
      value: secure,
      configurable: true,
    });

  beforeEach(() => setSecureContext(true));

  afterEach(() => setSecureContext(false));

  it("should start on the Core list when no Core is selected", async () => {
    await renderLogin();
    expect(await screen.findByText(ALPHA.name)).toBeTruthy();
    expect(screen.getByText(BETA.name)).toBeTruthy();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
  });

  it("should advance to the credentials with the Core's saved username", async () => {
    await renderLogin();
    fireEvent.click(await screen.findByText(BETA.name));
    const username = await screen.findByPlaceholderText(USERNAME_PLACEHOLDER);
    assert(username instanceof HTMLInputElement);
    expect(username.value).toBe(BETA.username);
    expect(screen.queryByText(ALPHA.name)).toBeNull();
  });

  it("should keep the password form when the Core accepts passwords", async () => {
    let served!: () => void;
    const answered = new Promise<void>((resolve) => (served = resolve));
    await renderLogin(await serveAuth(["password", "api_key"], [], () => served()));
    fireEvent.click(await screen.findByText(ALPHA.name));
    await answered;
    await expectAlways(() => {
      expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).not.toBeNull();
      expect(screen.queryByText("Password login is off")).toBeNull();
    });
  });

  it("should hide the password form when the Core refuses passwords", async () => {
    await renderLogin(await serveAuth(["api_key"]));
    fireEvent.click(await screen.findByText(ALPHA.name));
    expect(await screen.findByText("Password login is off")).toBeTruthy();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
  });

  it("should offer each provider the Core accepts", async () => {
    const core = await serveAuth(
      ["password", "oidc"],
      [
        {
          provider: "company-sso",
          authorizationEndpoint: "https://sso.example.com/authorize",
          clientId: "synnax",
        },
      ],
    );
    await renderLogin(core);
    fireEvent.click(await screen.findByText(ALPHA.name));
    expect(await screen.findByText("Log in with company-sso")).toBeTruthy();
    expect(screen.getByPlaceholderText(USERNAME_PLACEHOLDER)).toBeTruthy();
  });

  it("should show only providers when the Core refuses passwords", async () => {
    const core = await serveAuth(
      ["oidc"],
      [
        {
          provider: "company-sso",
          authorizationEndpoint: "https://sso.example.com/authorize",
          clientId: "synnax",
        },
      ],
    );
    await renderLogin(core);
    fireEvent.click(await screen.findByText(ALPHA.name));
    expect(await screen.findByText("Log in with company-sso")).toBeTruthy();
    expect(screen.queryByText("Password login is off")).toBeNull();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
  });

  it("should explain that provider login needs HTTPS", async () => {
    setSecureContext(false);
    const core = await serveAuth(
      ["oidc"],
      [
        {
          provider: "company-sso",
          authorizationEndpoint: "https://sso.example.com/authorize",
          clientId: "synnax",
        },
      ],
    );
    await renderLogin(core);
    fireEvent.click(await screen.findByText(ALPHA.name));
    expect(await screen.findByText("Provider login needs HTTPS")).toBeTruthy();
    expect(screen.queryByText("Log in with company-sso")).toBeNull();
  });

  it("should report a providers request that fails", async () => {
    const core = await serveAuth(["oidc"], null);
    let statuses: Status.NotificationSpec[] = [];
    const { wrapper } = await createSessionConsoleWrapper({
      client: null,
      preloadedState: {
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          cores: { [core.key]: core },
          selected: core.key,
        },
      },
    });
    render(
      <>
        <CaptureStatuses onStatuses={(next) => (statuses = next)} />
        <Login />
      </>,
      { wrapper },
    );
    await vi.waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain(
        "Failed to retrieve the login providers",
      ),
    );
    expect(screen.queryByText("Password login is off")).toBeNull();
  });

  it("should start a browser login through a provider", async () => {
    const core = await serveAuth(
      ["oidc"],
      [
        {
          provider: "company-sso",
          authorizationEndpoint: "https://sso.example.com/authorize",
          clientId: "synnax",
        },
      ],
    );
    await renderLogin(core);
    fireEvent.click(await screen.findByText(ALPHA.name));
    fireEvent.click(await screen.findByText("Log in with company-sso"));
    await vi.waitFor(() =>
      expect(
        JSON.parse(sessionStorage.getItem("synnax-oidc-login") ?? "{}"),
      ).toMatchObject({ core: ALPHA.key, provider: "company-sso" }),
    );
    sessionStorage.clear();
  });

  it("should return to the Core list from the credentials", async () => {
    const { container } = await renderLogin();
    fireEvent.click(await screen.findByText(ALPHA.name));
    await screen.findByPlaceholderText(USERNAME_PLACEHOLDER);
    fireEvent.click(getBySelector(container, BACK));
    expect(await screen.findByText(BETA.name)).toBeTruthy();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
  });

  it("should show why the Core rejected the connection", async () => {
    const { wrapper } = await createStatusConsoleWrapper({
      client: createTestClient(),
      status: REJECTED,
      preloadedState: {
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          cores: { [ALPHA.key]: ALPHA },
          selected: ALPHA.key,
        },
      },
    });
    render(<Login />, { wrapper });
    expect(await screen.findByText("Failed to authenticate")).toBeTruthy();
    expect(screen.getByText("invalid credentials")).toBeTruthy();
  });

  describe("on desktop", () => {
    beforeEach(() => {
      runtime.engine = "tauri";
    });

    afterEach(() => {
      runtime.engine = "web";
    });

    const FAILED = "Failed to log in with company-sso";

    it("should save the session and log in through a provider", async () => {
      const core = await serveAuth(["oidc"], [PROVIDER], undefined, "sys_token");
      const ipc = fakeDesktop();
      const { store } = await renderLogin(core, createConsoleWrapper);
      fireEvent.click(await screen.findByText(ALPHA.name));
      fireEvent.click(await screen.findByText("Log in with company-sso"));
      await vi.waitFor(() =>
        expect(store.getState().core.cores[core.key].lastLogin).toBeDefined(),
      );
      expect(store.getState().core.selected).toBe(core.key);
      expect(Object.fromEntries(ipc.saved)).toEqual({
        [`http://@127.0.0.1:${core.port}#${core.key}`]: "sys_token",
      });
    });

    it("should let the user retry a login the Core rejects", async () => {
      const core = await serveAuth(["oidc"], [PROVIDER]);
      const ipc = fakeDesktop();
      const { store, statuses } = await renderLogin(core, createConsoleWrapper);
      fireEvent.click(await screen.findByText(ALPHA.name));
      fireEvent.click(await screen.findByText("Log in with company-sso"));
      await vi.waitFor(() =>
        expect(statuses().map((s) => s.message)).toContain(FAILED),
      );
      fireEvent.click(screen.getByText("Log in with company-sso"));
      await vi.waitFor(() => expect(ipc.listens).toBe(2));
      expect(ipc.saved.size).toBe(0);
      expect(store.getState().core.cores[core.key].lastLogin).toBeUndefined();
    });

    it("should refuse a login when the Core's address changed during it", async () => {
      const core = await serveAuth(["oidc"], [PROVIDER], undefined, "sys_token");
      let move = (): void => {};
      fakeDesktop(() => move());
      const { store, statuses } = await renderLogin(core, createConsoleWrapper);
      move = () => store.dispatch(Session.Core.set({ ...core, port: core.port + 1 }));
      fireEvent.click(await screen.findByText(ALPHA.name));
      fireEvent.click(await screen.findByText("Log in with company-sso"));
      await vi.waitFor(() =>
        expect(statuses()).toContainEqual(
          expect.objectContaining({
            message: FAILED,
            description: expect.stringContaining("address changed"),
          }),
        ),
      );
      expect(store.getState().core.cores[core.key].lastLogin).toBeUndefined();
    });
  });
});
