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

import {
  auth,
  AuthError,
  connection,
  ExpiredTokenError,
  InvalidCredentialsError,
  InvalidTokenError,
  MissingCredentialsError,
  SessionEndedError,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
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

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, runtime);
});

import { Auth } from "@/feature/auth";
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

const client = createTestClient();

// Dead ports: the list checks each Core, and no assertion here reads the result.
const ALPHA = createCore("alpha", { name: "Alpha", port: 9098, username: "ada" });
const BETA = createCore("beta", { name: "Beta", port: 9099, username: "grace" });

// No test Core turns passwords off or accepts a provider, so a local server answers the
// methods request. The Console asks once for the methods and again for the providers;
// null providers fail that second request, and null methods fail both, as a Core older
// than the methods endpoint does. A token opens a session for any code, and without one
// the server rejects every code.
const serveAuth = async (
  methods: string[] | null,
  providers: auth.OIDCLogin[] | null = [],
  onServed?: () => void,
  token?: string,
): Promise<Session.Core.Core> => {
  // the local auth server answers outside any act
  disableActEnvironment();
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
      if (methods == null || !isMethods || (providers == null && methodsRequests > 1)) {
        res.statusCode = 404;
        res.end(onServed);
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
  issuer: "https://sso.example.com",
  authorizationEndpoint: "https://sso.example.com/authorize",
  clientId: "synnax",
};

const rejected = (error: Error): connection.Status => ({
  ...connection.DEFAULT_STATUS,
  variant: "error",
  name: ALPHA.name,
  message: "Failed to authenticate",
  description: error.message,
  details: { ...connection.DEFAULT_STATUS.details, reason: "auth", error },
});

const BACK = ".console-login__back";
const ERROR = ".console-login__error";
const GHOST = "console--ghost";
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
        order: [alpha.key, BETA.key],
        selected: undefined,
      },
    },
  });
  const { container } = render(
    <>
      <CaptureStatuses onStatuses={(next) => (statuses = next)} />
      <Auth.Guard>
        <span>workspace</span>
      </Auth.Guard>
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

describe("Auth login", () => {
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
      expect(screen.queryByText("No way to log in")).toBeNull();
    });
  });

  it("should hide the password form when the Core refuses passwords", async () => {
    await renderLogin(await serveAuth(["api_key"]));
    fireEvent.click(await screen.findByText(ALPHA.name));
    expect(await screen.findByText("No way to log in")).toBeTruthy();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
  });

  it("should keep the password form when the Core has no methods endpoint", async () => {
    let served!: () => void;
    const answered = new Promise<void>((resolve) => (served = resolve));
    await renderLogin(await serveAuth(null, [], () => served()));
    fireEvent.click(await screen.findByText(ALPHA.name));
    await answered;
    await expectAlways(() => {
      expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).not.toBeNull();
      expect(screen.queryByText("No way to log in")).toBeNull();
    });
  });

  it("should drop the methods the client does not know", async () => {
    const { port } = await serveAuth(["api_key", "carrier_pigeon"]);
    const client = auth.createTokenlessClient({ host: "127.0.0.1", port });
    expect(await auth.retrieveMethods(client)).toEqual(["api_key"]);
  });

  it("should offer each provider the Core accepts", async () => {
    const core = await serveAuth(
      ["password", "oidc"],
      [
        {
          provider: "company-sso",
          issuer: "https://sso.example.com",
          authorizationEndpoint: "https://sso.example.com/authorize",
          clientId: "synnax",
        },
      ],
    );
    await renderLogin(core);
    fireEvent.click(await screen.findByText(ALPHA.name));
    expect(await screen.findByText("Log in with company-sso")).toBeTruthy();
    expect(screen.getByPlaceholderText(USERNAME_PLACEHOLDER)).toBeTruthy();
    expect(screen.getByText("or")).toBeTruthy();
  });

  it("should show only providers when the Core refuses passwords", async () => {
    const core = await serveAuth(
      ["oidc"],
      [
        {
          provider: "company-sso",
          issuer: "https://sso.example.com",
          authorizationEndpoint: "https://sso.example.com/authorize",
          clientId: "synnax",
        },
      ],
    );
    await renderLogin(core);
    fireEvent.click(await screen.findByText(ALPHA.name));
    expect(await screen.findByText("Log in with company-sso")).toBeTruthy();
    expect(screen.queryByText("No way to log in")).toBeNull();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
    expect(screen.queryByText("or")).toBeNull();
  });

  it("should name and mark the providers of known brands", async () => {
    const core = await serveAuth(
      ["oidc"],
      [
        {
          provider: "google",
          issuer: "https://accounts.google.com",
          authorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
          clientId: "synnax",
        },
        {
          provider: "corp-sso",
          issuer: "https://acme.okta.com",
          authorizationEndpoint: "https://acme.okta.com/oauth2/v1/authorize",
          clientId: "synnax",
        },
        {
          provider: "azure-ad",
          issuer: "https://login.microsoftonline.com",
          authorizationEndpoint: "https://login.microsoftonline.com/acme/authorize",
          clientId: "synnax",
        },
        {
          provider: "plant-sso",
          issuer: "https://sso.plant.example.com",
          authorizationEndpoint: "https://sso.plant.example.com/authorize",
          clientId: "synnax",
        },
      ],
    );
    await renderLogin(core);
    fireEvent.click(await screen.findByText(ALPHA.name));
    const google = await screen.findByRole("button", { name: "Log in with Google" });
    const corp = screen.getByRole("button", { name: "Log in with corp-sso" });
    const microsoft = screen.getByRole("button", { name: "Log in with Microsoft" });
    const plant = screen.getByRole("button", { name: "Log in with plant-sso" });
    for (const button of [google, corp, microsoft, plant])
      expect(button.querySelector("svg")).not.toBeNull();
  });

  it("should explain that provider login needs HTTPS", async () => {
    setSecureContext(false);
    const core = await serveAuth(
      ["oidc"],
      [
        {
          provider: "company-sso",
          issuer: "https://sso.example.com",
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

  it("should say there is no way to log in without providers", async () => {
    await renderLogin(await serveAuth(["oidc"], []));
    fireEvent.click(await screen.findByText(ALPHA.name));
    expect(await screen.findByText("No way to log in")).toBeTruthy();
  });

  it("should report a providers request that fails", async () => {
    const { statuses } = await renderLogin(await serveAuth(["oidc"], null));
    fireEvent.click(await screen.findByText(ALPHA.name));
    await vi.waitFor(() =>
      expect(statuses().map((s) => s.message)).toContain(
        "Failed to retrieve the login providers",
      ),
    );
    expect(screen.queryByText("No way to log in")).toBeNull();
  });

  it("should start a browser login through a provider", async () => {
    const core = await serveAuth(
      ["oidc"],
      [
        {
          provider: "company-sso",
          issuer: "https://sso.example.com",
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

  const renderRejected = async (error: Error): Promise<HTMLElement> => {
    const { wrapper } = await createStatusConsoleWrapper({
      client,
      status: rejected(error),
      preloadedState: {
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          cores: { [ALPHA.key]: ALPHA },
          order: [ALPHA.key],
          selected: ALPHA.key,
        },
      },
    });
    const { container } = render(
      <Auth.ConnectionGuard>
        <span>workspace</span>
      </Auth.ConnectionGuard>,
      { wrapper },
    );
    await screen.findByPlaceholderText(USERNAME_PLACEHOLDER);
    return container;
  };

  it.each([
    [
      "a wrong password",
      new InvalidCredentialsError("invalid credentials"),
      "Incorrect username or password.",
    ],
    [
      "an ended session",
      new SessionEndedError("session ended"),
      "Your session ended. Log in again.",
    ],
    [
      "an invalid token",
      new InvalidTokenError("invalid token"),
      "Your session ended. Log in again.",
    ],
    [
      "an expired token",
      new ExpiredTokenError("expired token"),
      "Your session ended. Log in again.",
    ],
    [
      "an unknown auth error",
      new AuthError("authentication service unavailable"),
      "authentication service unavailable",
    ],
  ])("should explain %s under the Log in button", async (_, error, message) => {
    const container = await renderRejected(error);
    const note = getBySelector(container, ERROR);
    expect(note.textContent).toBe(message);
    expect(note.classList).not.toContain(GHOST);
    expect(screen.getByRole("button", { name: /Log in/ }).nextElementSibling).toBe(
      note,
    );
  });

  it("should explain a rejected provider login on a passwordless Core", async () => {
    const core = await serveAuth(["oidc"], [PROVIDER]);
    const { wrapper } = await createStatusConsoleWrapper({
      client,
      status: rejected(new AuthError("invalid client secret")),
      preloadedState: {
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          cores: { [core.key]: core },
          order: [core.key],
          selected: core.key,
        },
      },
    });
    render(
      <Auth.ConnectionGuard>
        <span>workspace</span>
      </Auth.ConnectionGuard>,
      { wrapper },
    );
    expect(await screen.findByText("Log in with company-sso")).toBeTruthy();
    expect(screen.getByText("invalid client secret")).toBeTruthy();
  });

  it("should keep the error space empty before any login", async () => {
    const container = await renderRejected(
      new MissingCredentialsError("no credentials, API key, token, or cookie"),
    );
    expect(getBySelector(container, ERROR).classList).toContain(GHOST);
    expect(screen.queryByText(/no credentials/)).toBeNull();
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

    it("should let a new provider login replace one still waiting", async () => {
      const core = await serveAuth(["oidc"], [PROVIDER], undefined, "sys_token");
      const saved = new Map<string, string>();
      let replace: (() => void) | undefined;
      let waits = 0;
      mockWindows("main");
      mockIPC(async (cmd, args) => {
        const { account, secret, state } = args as Record<string, string>;
        switch (cmd) {
          case "plugin:app|version":
            return "0.0.0";
          case "keychain_set":
            saved.set(account, secret);
            return undefined;
          case "oidc_listen":
            replace?.();
            return 4242;
          case "oidc_wait":
            waits += 1;
            if (waits > 1) return `code=abc&state=${state}`;
            return await new Promise<string>((_, reject) => {
              replace = () => reject(new Error("a newer login replaced this one"));
            });
          default:
            return undefined;
        }
      });
      onTestFinished(clearMocks);
      const { store, statuses } = await renderLogin(core, createConsoleWrapper);
      fireEvent.click(await screen.findByText(ALPHA.name));
      fireEvent.click(await screen.findByText("Log in with company-sso"));
      await vi.waitFor(() => expect(waits).toBe(1));
      fireEvent.click(screen.getByText("Log in with company-sso"));
      await vi.waitFor(() =>
        expect(store.getState().core.cores[core.key].lastLogin).toBeDefined(),
      );
      expect(statuses().map((s) => s.message)).not.toContain(FAILED);
      expect(Object.fromEntries(saved)).toEqual({
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

    it("should name a failed provider login by its brand", async () => {
      const google = { ...PROVIDER, provider: "google" };
      const core = await serveAuth(["oidc"], [google]);
      fakeDesktop();
      const { statuses } = await renderLogin(core, createConsoleWrapper);
      fireEvent.click(await screen.findByText(ALPHA.name));
      fireEvent.click(await screen.findByText("Log in with Google"));
      await vi.waitFor(() =>
        expect(statuses().map((s) => s.message)).toContain(
          "Failed to log in with Google",
        ),
      );
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
