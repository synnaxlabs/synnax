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

import { AuthError, connection } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { expectAlways } from "@synnaxlabs/x/testutil";
import { fireEvent, render, screen } from "@testing-library/react";
import { assert, describe, expect, it, onTestFinished, vi } from "vitest";

// A serving Core skips the Core step, and detection reports one for every browser
// build. The module is the only seam.
vi.mock("@/platform/core/detectConnection", () => ({
  detectConnection: () => null,
}));

import { Login } from "@/feature/auth/Login";
import { Session } from "@/session";
import { createCore } from "@/session/core/testutil";
import {
  createSessionConsoleWrapper,
  createStatusConsoleWrapper,
  getBySelector,
} from "@/testutil";

// Dead ports: the list checks each Core, and no assertion here reads the result.
const ALPHA = createCore("alpha", { name: "Alpha", port: 9098, username: "ada" });
const BETA = createCore("beta", { name: "Beta", port: 9099, username: "grace" });

// No test Core turns passwords off, so a local server answers the methods request.
const serveMethods = async (
  methods: string[],
  onServed?: () => void,
): Promise<number> => {
  const server = createServer((req, res) => {
    req.resume();
    req.on("end", () => {
      if (req.url?.endsWith("/auth/methods") !== true) {
        res.statusCode = 404;
        res.end();
        return;
      }
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify({ methods }), onServed);
    });
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  onTestFinished(
    async () => await new Promise<void>((resolve) => server.close(() => resolve())),
  );
  return (server.address() as AddressInfo).port;
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
    const port = await serveMethods(["password", "api_key"], () => served());
    await renderLogin({ ...ALPHA, host: "127.0.0.1", port });
    fireEvent.click(await screen.findByText(ALPHA.name));
    await answered;
    await expectAlways(() => {
      expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).not.toBeNull();
      expect(screen.queryByText("Password login is off")).toBeNull();
    });
  });

  it("should hide the password form when the Core refuses passwords", async () => {
    const port = await serveMethods(["api_key"]);
    await renderLogin({ ...ALPHA, host: "127.0.0.1", port });
    fireEvent.click(await screen.findByText(ALPHA.name));
    expect(await screen.findByText("Password login is off")).toBeTruthy();
    expect(screen.queryByPlaceholderText(USERNAME_PLACEHOLDER)).toBeNull();
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
