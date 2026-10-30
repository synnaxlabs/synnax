// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Account } from "@/feature/account";
import { License } from "@/platform/license";
import { Modals } from "@/platform/modals";
import { findButton, getSwitch } from "@/platform/modals/testutil";
import { Session } from "@/session";
import {
  CaptureStatuses,
  createConsoleWrapper,
  createStatusConsoleWrapper,
  type TestStore,
  UNLICENSED_STATUS,
} from "@/testutil";

interface RenderOptions {
  status?: connection.Status;
  account?: Session.Account.SliceState;
}

const renderGuard = async ({ status, account }: RenderOptions = {}): Promise<{
  client: Client;
  store: TestStore;
}> => {
  const client = createTestClient();
  const { wrapper, store } = await createStatusConsoleWrapper({
    client,
    status,
    preloadedState:
      account == null ? undefined : { [Session.Account.SLICE_NAME]: account },
  });
  render(
    <>
      <Modals.Stack />
      <Account.Guard>
        <span>licensed content</span>
      </Account.Guard>
    </>,
    { wrapper },
  );
  return { client, store };
};

describe("Account.Guard", () => {
  afterEach(() => vi.restoreAllMocks());

  it("should render children while the Core is not refused for a license", async () => {
    await renderGuard();
    expect(screen.getByText("licensed content")).toBeTruthy();
  });

  it("should ask for a login while the Core is unlicensed", async () => {
    await renderGuard({ status: UNLICENSED_STATUS });
    expect(screen.getByText("Log in to continue")).toBeTruthy();
    expect(screen.queryByText("licensed content")).toBeNull();
  });

  it("should render children once the Core is licensed", async () => {
    const client = createTestClient();
    const { wrapper } = await createConsoleWrapper({ client: null });
    const ui = (status: connection.Status): ReactElement => (
      <Synnax.TestProvider client={client} status={status}>
        <Account.Guard>
          <span>licensed content</span>
        </Account.Guard>
      </Synnax.TestProvider>
    );
    const { rerender } = render(ui(UNLICENSED_STATUS), { wrapper });
    expect(screen.getByText("Log in to continue")).toBeTruthy();
    rerender(ui(connection.DEFAULT_STATUS));
    expect(await screen.findByText("licensed content")).toBeTruthy();
    expect(screen.queryByText("Log in to continue")).toBeNull();
  });

  it("should say the login lapsed on a machine that was linked", async () => {
    await renderGuard({
      status: UNLICENSED_STATUS,
      account: {
        ...Session.Account.ZERO_SLICE_STATE,
        link: { secret: "shh", email: "someone@example.com", user: "user_a" },
      },
    });
    expect(screen.getByText("Your login has lapsed")).toBeTruthy();
    expect(screen.getByText(/someone@example\.com/)).toBeTruthy();
  });

  it("should open the usage data settings on a machine that was linked", async () => {
    await renderGuard({
      status: UNLICENSED_STATUS,
      account: {
        ...Session.Account.ZERO_SLICE_STATE,
        link: { secret: "shh", email: "someone@example.com", user: "user_a" },
        analyticsAllowed: true,
      },
    });
    fireEvent.click(findButton("Usage data"));
    await screen.findByText("Share usage data");
    expect(getSwitch("Share usage data").checked).toBe(true);
  });

  it("should not offer usage data on a machine that was never linked", async () => {
    await renderGuard({ status: UNLICENSED_STATUS });
    expect(screen.queryByText("Usage data")).toBeNull();
  });

  it("should open the hub with the state it minted", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    const { client, store } = await renderGuard({ status: UNLICENSED_STATUS });
    const login = findButton("Log in");
    await waitFor(() => expect(login.getAttribute("aria-disabled")).toBeNull());
    fireEvent.click(login);
    await waitFor(() => expect(open).toHaveBeenCalled());
    const url = new URL(String(open.mock.calls[0][0]));
    expect(url.origin + url.pathname).toBe(License.LOGIN_URL);
    const { fingerprint } = await client.license.retrieve();
    expect(url.searchParams.get("fp")).toBe(License.joinFingerprint(fingerprint));
    expect(url.searchParams.get("name")).toBe("Synnax Desktop");
    expect(url.searchParams.get("state")).toBe(
      Session.Account.selectPending(store.getState()),
    );
    expect(await screen.findByText("Finish in your browser")).toBeTruthy();
  });

  it("should send the same state when the page opens again", async () => {
    // A new state would make the first page's login link fail to finish.
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    const { store } = await renderGuard({ status: UNLICENSED_STATUS });
    const login = findButton("Log in");
    await waitFor(() => expect(login.getAttribute("aria-disabled")).toBeNull());
    fireEvent.click(login);
    expect(await screen.findByText("Finish in your browser")).toBeTruthy();
    fireEvent.click(findButton("Open the page again"));
    await waitFor(() => expect(open).toHaveBeenCalledTimes(2));
    const states = open.mock.calls.map(([url]) =>
      new URL(String(url)).searchParams.get("state"),
    );
    expect(states[1]).toBe(states[0]);
    expect(Session.Account.selectPending(store.getState())).toBe(states[0]);
  });

  it("should finish a login left pending from before", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    await renderGuard({
      status: UNLICENSED_STATUS,
      account: { ...Session.Account.ZERO_SLICE_STATE, pending: "minted" },
    });
    const login = findButton("Log in");
    await waitFor(() => expect(login.getAttribute("aria-disabled")).toBeNull());
    fireEvent.click(login);
    await waitFor(() => expect(open).toHaveBeenCalled());
    const url = new URL(String(open.mock.calls[0][0]));
    expect(url.searchParams.get("state")).toBe("minted");
  });

  it("should let the user retry a log in whose fingerprint read failed", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    let statuses: Status.NotificationSpec[] = [];
    const { wrapper } = await createConsoleWrapper({ client: null });
    const ui = (client: Client): ReactElement => (
      <Synnax.TestProvider client={client} status={UNLICENSED_STATUS}>
        <CaptureStatuses onStatuses={(next) => (statuses = next)} />
        <Account.Guard>
          <span>licensed content</span>
        </Account.Guard>
      </Synnax.TestProvider>
    );
    const { rerender } = render(ui(createTestClient({ port: 1 })), { wrapper });
    fireEvent.click(findButton("Log in"));
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to open login page"),
    );
    expect(open).not.toHaveBeenCalled();
    rerender(ui(createTestClient()));
    fireEvent.click(findButton("Log in"));
    await waitFor(() => expect(open).toHaveBeenCalledOnce());
  });

  it("should ask to try again while the machine is offline", async () => {
    vi.spyOn(navigator, "onLine", "get").mockReturnValue(false);
    await renderGuard({ status: UNLICENSED_STATUS });
    expect(screen.getByText("You are offline")).toBeTruthy();
    expect(findButton("Try again")).toBeTruthy();
    expect(screen.queryByText("Log in")).toBeNull();
  });

  it("should offer the license file screen and a way back", async () => {
    await renderGuard({ status: UNLICENSED_STATUS });
    fireEvent.click(findButton("Use a license file"));
    expect(screen.getByPlaceholderText("Paste a license key")).toBeTruthy();
    expect(screen.queryByText("Log out")).toBeNull();
    fireEvent.click(findButton("Back"));
    expect(screen.getByText("Log in to continue")).toBeTruthy();
  });
});
