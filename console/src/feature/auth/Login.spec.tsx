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

import { Login } from "@/feature/auth/Login";
import { Session } from "@/session";
import { createCore } from "@/session/core/testutil";
import {
  CaptureStatuses,
  createSessionConsoleWrapper,
  createStatusConsoleWrapper,
  getBySelector,
} from "@/testutil";

// Dead ports: the list checks each Core, and no assertion here reads the result.
const ALPHA = createCore("alpha", { name: "Alpha", port: 9098, username: "ada" });
const BETA = createCore("beta", { name: "Beta", port: 9099, username: "grace" });

// No test Core turns passwords off or accepts a provider, so a local server answers the
// methods request. The Console asks once for the methods and again for the providers;
// null providers fail that second request.
const serveAuth = async (
  methods: string[],
  providers: auth.OIDCLogin[] | null = [],
  onServed?: () => void,
): Promise<Session.Core.Core> => {
  let methodsRequests = 0;
  const server = createServer((req, res) => {
    req.resume();
    req.on("end", () => {
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

const renderLogin = async (alpha = ALPHA): Promise<HTMLElement> => {
  const { wrapper } = await createSessionConsoleWrapper({
    client: null,
    preloadedState: {
      [Session.Core.SLICE_NAME]: {
        ...Session.Core.ZERO_SLICE_STATE,
        cores: { [alpha.key]: alpha, [BETA.key]: BETA },
        selected: undefined,
      },
    },
  });
  const { container } = render(<Login />, { wrapper });
  return container;
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
    const container = await renderLogin();
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
});
